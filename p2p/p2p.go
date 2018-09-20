// Package p2p implements the vite P2P network

package p2p

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/block"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/nat"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var p2pServerLog = log15.New("module", "p2p/server")

var errSvrStarted = errors.New("Server has started")
var errSvrStopped = errors.New("Server has stopped")

type Discovery interface {
	Lookup(discovery.NodeID) []*discovery.Node
	Resolve(discovery.NodeID) *discovery.Node
	RandomNodes([]*discovery.Node) int
	Start()
	Stop()
}

type Config struct {
	Name            string
	NetID           NetworkID          // which network server runs on
	MaxPeers        uint               // max peers can be connected
	MaxPendingPeers uint               // max peers waiting for connect
	MaxInboundRatio uint               // max inbound peers: MaxPeers / MaxInboundRatio
	Port            uint               // TCP and UDP listen port
	Database        string             // the directory for storing node table
	PrivateKey      ed25519.PrivateKey // use for encrypt message, the corresponding public key use for NodeID
	Protocols       []*Protocol        // protocols server supported
	BootNodes       []string
	KafKa           []string
}

type Server struct {
	*Config
	running      int32          // atomic
	wg           sync.WaitGroup // Wait for all jobs done
	term         chan struct{}
	pending      chan struct{} // how many connection can wait for handshake
	addPeer      chan *conn
	delPeer      chan *Peer
	discv        Discovery
	ourHandshake *Handshake
	BootNodes    []*discovery.Node
	peers        *PeerSet
	blockList    *block.CuckooSet
	topo         *topoHandler
	self         *discovery.Node
	agent        *agent
	log          log15.Logger
	producer     *producer
}

func New(cfg Config) *Server {
	safeCfg := EnsureConfig(cfg)

	svr := &Server{
		Config:    safeCfg,
		log:       p2pServerLog.New("module", "p2p/server"),
		peers:     newPeerSet(),
		term:      make(chan struct{}),
		pending:   make(chan struct{}, cfg.MaxPendingPeers),
		addPeer:   make(chan *conn, 1),
		delPeer:   make(chan *Peer, 1),
		BootNodes: addFirmNodes(cfg.BootNodes),
		blockList: block.NewCuckooSet(100),
		topo:      newTopoHandler(),
	}

	if svr.KafKa != nil {
		producer, err := newProducer(svr.KafKa)
		if err == nil {
			svr.producer = producer
		}
	}

	return svr
}

func (svr *Server) Peers() []*PeerInfo {
	return svr.peers.Info()
}

func (svr *Server) PeersCount() (amount int) {
	return svr.peers.Size()
}

func (svr *Server) NodeInfo() *NodeInfo {
	protocols := make([]string, len(svr.Protocols))
	for i, protocol := range svr.Protocols {
		protocols[i] = protocol.String()
	}

	return &NodeInfo{
		ID:    svr.self.ID.String(),
		Name:  svr.Name,
		Url:   svr.self.String(),
		NetID: svr.NetID,
		Address: &address{
			IP:  svr.self.IP,
			TCP: svr.self.TCP,
			UDP: svr.self.UDP,
		},
		Protocols: protocols,
	}
}

// the first item is self url
func (svr *Server) Topology() *Topo {
	count := svr.PeersCount()

	topo := &Topo{
		Pivot: svr.self.String(),
		Peers: make([]string, count),
	}

	svr.peers.Traverse(func(id discovery.NodeID, p *Peer) {
		topo.Peers = append(topo.Peers, p.String())
	})

	return topo
}

func (svr *Server) Available() bool {
	return svr.PeersCount() > 0
}

func (svr *Server) maxOutboundPeers() uint {
	return svr.MaxPeers - svr.maxInboundPeers()
}

func (svr *Server) maxInboundPeers() uint {
	return svr.MaxPeers / svr.MaxInboundRatio
}

func (svr *Server) Start() error {
	if !atomic.CompareAndSwapInt32(&svr.running, 0, 1) {
		return errSvrStarted
	}

	ID, err := discovery.Priv2NodeID(svr.Config.PrivateKey)
	if err != nil {
		return err
	}

	svr.setHandshake(ID)

	addr := "0.0.0.0:" + strconv.FormatUint(uint64(svr.Port), 10)
	// udp discover
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	// tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		svr.log.Crit("tcp listening error", "err", err)
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		svr.log.Crit("tcp listening error", "err", err)
	} else {
		svr.log.Info(fmt.Sprintf("tcp listening at %s", tcpAddr))
	}

	node := &discovery.Node{
		ID:  ID,
		IP:  udpAddr.IP,
		UDP: uint16(udpAddr.Port),
		TCP: uint16(tcpAddr.Port),
	}
	svr.self = node
	// mapping udp and tcp
	go nat.Map(svr.term, "udp", int(svr.self.UDP), int(svr.self.UDP), "vite p2p udp", 0, svr.updateNode)
	go nat.Map(svr.term, "tcp", int(svr.self.TCP), int(svr.self.TCP), "vite p2p tcp", 0, svr.updateNode)

	svr.discv = discovery.New(&discovery.Config{
		Priv:      svr.PrivateKey,
		DBPath:    svr.Database,
		BootNodes: svr.BootNodes,
		Conn:      conn,
		Self:      node,
	})

	svr.discv.Start()

	svr.agent = newAgent(svr)
	svr.agent.start()

	// tcp listener
	svr.wg.Add(1)
	go svr.listenLoop(listener)

	// task loop
	svr.wg.Add(1)
	go svr.loop()

	p2pServerLog.Info("p2p server started")
	return nil
}

func (svr *Server) updateNode(addr *nat.Addr) {
	if addr.Proto == "tcp" {
		svr.self.TCP = uint16(addr.Port)
	} else {
		svr.self.UDP = uint16(addr.Port)
	}
}

func (svr *Server) setHandshake(ID discovery.NodeID) {
	cmdsets := make([]*CmdSet, len(svr.Protocols))
	for i, p := range svr.Protocols {
		cmdsets[i] = p.CmdSet()
	}
	svr.ourHandshake = &Handshake{
		Version: Version,
		Name:    svr.Name,
		NetID:   svr.NetID,
		ID:      ID,
		CmdSets: cmdsets,
	}
}

func (svr *Server) listenLoop(listener *net.TCPListener) {
	defer svr.wg.Done()

	var conn net.Conn
	var err error
	for {
		select {
		case svr.pending <- struct{}{}:
			for {
				conn, err = listener.Accept()

				if err != nil {
					svr.log.Error("tcp accept error", "error", err)
					continue
				}
				break
			}
			go svr.setupConn(conn, inbound)
		case <-svr.term:
			close(svr.pending)
			goto END
		}
	}

END:
	listener.Close()
}

func (svr *Server) setupConn(c net.Conn, flag connFlag) {
	ts := &conn{
		fd:        c,
		transport: newProtoX(c),
		flags:     flag,
		term:      make(chan struct{}),
	}

	svr.log.Info(fmt.Sprintf("begin handshake with %s", c.RemoteAddr()))

	their, err := ts.Handshake(svr.ourHandshake)

	if err != nil {
		ts.close(err)
		svr.log.Error(fmt.Sprintf("handshake error with %s: %v", c.RemoteAddr(), err))
	} else {
		ts.id = their.ID
		ts.name = their.Name
		ts.cmdSets = their.CmdSets

		svr.log.Info(fmt.Sprintf("handshake with %s@%s done", ts.id, c.RemoteAddr()))
		svr.addPeer <- ts
	}

	<-svr.pending
}

func (svr *Server) checkConn(c *conn) error {
	if uint(svr.peers.Size()) >= svr.MaxPeers {
		return DiscTooManyPeers
	}

	if uint(svr.peers.inbound) >= svr.maxInboundPeers() {
		return DiscTooManyPassivePeers
	}

	if svr.peers.Has(c.id) {
		return DiscAlreadyConnected
	}

	if c.id == svr.self.ID {
		return DiscSelf
	}

	return nil
}

func (svr *Server) loop() {
	defer svr.wg.Done()

	// broadcast topo to peers
	topoInterval := time.Minute
	if svr.NetID == MainNet {
		topoInterval = 10 * time.Minute
	}
	topoTicker := time.NewTicker(topoInterval)
	defer topoTicker.Stop()

	for {
		select {
		case <-svr.term:
			goto END
		case c := <-svr.addPeer:
			err := svr.checkConn(c)
			if err == nil {
				if p, err := NewPeer(c, svr.Protocols, svr.topo.rec); err != nil {
					svr.peers.Add(p)

					peersCount := svr.peers.Size()
					svr.log.Info("create new peer", "ID", c.id.String(), "total", peersCount)
					monitor.LogDuration("p2p/peer", "add", int64(peersCount))

					go svr.runPeer(p)
				}
			} else {
				c.close(err)
				svr.log.Error("cannot create new peer", "error", err)
			}

		case p := <-svr.delPeer:
			svr.peers.Del(p)

			peersCount := svr.peers.Size()
			svr.log.Info("delete peer", "ID", p.ID().String(), "total", peersCount)
			monitor.LogDuration("p2p/peer", "del", int64(peersCount))

		case <-topoTicker.C:
			topo := svr.Topology()
			go svr.peers.Traverse(func(id discovery.NodeID, p *Peer) {
				err := Send(p.rw, baseProtocolCmdSet, topoCmd, 0, topo)
				if err != nil {
					p.protoErr <- err
				}
			})
		case e := <-svr.topo.rec:
			monitor.LogEvent("p2p", "topo")
			svr.topo.Handle(e, svr)
		}
	}

END:
	svr.peers.Traverse(func(id discovery.NodeID, p *Peer) {
		p.Disconnect(DiscQuitting)
	})
}

func (svr *Server) runPeer(p *Peer) {
	err := p.run()
	if err != nil {
		svr.log.Error("run peer error", "error", err)
	}
	svr.delPeer <- p
}

func (svr *Server) Stop() {
	if !atomic.CompareAndSwapInt32(&svr.running, 1, 0) {
		return
	}

	svr.discv.Stop()
	svr.agent.stop()

	close(svr.term)
	svr.wg.Wait()
}

// @section NodeInfo
type NodeInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Url       string    `json:"url"`
	NetID     NetworkID `json:"netId"`
	Address   *address  `json:"address"`
	Protocols []string  `json:"protocols"`
}

type address struct {
	IP  net.IP `json:"ip"`
	TCP uint16 `json:"tcp"`
	UDP uint16 `json:"udp"`
}
