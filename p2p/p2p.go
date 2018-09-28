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
var errSvrStarted = errors.New("server has started")
var errSvrStopped = errors.New("server has stopped")

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
}

type Server struct {
	*Config
	running        int32          // atomic
	wg             sync.WaitGroup // Wait for all jobs done
	term           chan struct{}
	pending        chan struct{} // how many connection can wait for handshake
	addPeer        chan *conn
	delPeer        chan *Peer
	discv          Discovery
	handShakeBytes []byte // handshake data, after signature
	BootNodes      []*discovery.Node
	peers          *PeerSet
	blockList      *block.CuckooSet
	self           *discovery.Node
	agent          *agent
	log            log15.Logger
	ln             net.Listener
}

func New(cfg Config) (svr *Server, err error) {
	safeCfg := EnsureConfig(cfg)

	ID, err := discovery.Priv2NodeID(cfg.PrivateKey)
	if err != nil {
		return
	}

	addr := "0.0.0.0:" + strconv.FormatUint(uint64(cfg.Port), 10)
	// udp discover
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return
	}

	// tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return
	}

	svr = &Server{
		Config:    safeCfg,
		log:       log15.New("module", "p2p/server"),
		peers:     NewPeerSet(),
		term:      make(chan struct{}),
		pending:   make(chan struct{}, cfg.MaxPendingPeers),
		addPeer:   make(chan *conn, 1),
		delPeer:   make(chan *Peer, 1),
		BootNodes: addFirmNodes(cfg.BootNodes),
		blockList: block.NewCuckooSet(100),
		ln:        listener,
		self: &discovery.Node{
			ID:  ID,
			IP:  udpAddr.IP,
			UDP: uint16(udpAddr.Port),
			TCP: uint16(tcpAddr.Port),
		},
	}

	svr.discv = discovery.New(&discovery.Config{
		Priv:      svr.PrivateKey,
		DBPath:    svr.Database,
		BootNodes: svr.BootNodes,
		Conn:      udpConn,
		Self:      svr.self,
	})

	svr.agent = newAgent(svr)

	return
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

func (svr *Server) URL() string {
	return svr.self.String()
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

	err := svr.setHandshake()
	if err != nil {
		return err
	}

	// mapping udp and tcp
	go nat.Map(svr.term, "udp", int(svr.self.UDP), int(svr.self.UDP), "vite p2p udp", 0, svr.updateNode)
	go nat.Map(svr.term, "tcp", int(svr.self.TCP), int(svr.self.TCP), "vite p2p tcp", 0, svr.updateNode)

	svr.discv.Start()

	// tcp listener
	svr.wg.Add(1)
	go svr.listenLoop()

	// peer manager
	svr.wg.Add(1)
	go svr.loop()

	svr.log.Info("p2p server started")
	return nil
}

func (svr *Server) updateNode(addr *nat.Addr) {
	if addr.Proto == "tcp" {
		svr.self.TCP = uint16(addr.Port)
	} else {
		svr.self.UDP = uint16(addr.Port)
	}
}

func (svr *Server) setHandshake() error {
	cmdsets := make([]*CmdSet, len(svr.Protocols))
	for i, p := range svr.Protocols {
		cmdsets[i] = p.CmdSet()
	}
	ours := &Handshake{
		Version: Version,
		Name:    svr.Name,
		NetID:   svr.NetID,
		ID:      svr.self.ID,
		CmdSets: cmdsets,
	}

	data, err := ours.Serialize()
	if err != nil {
		return err
	}

	sig := ed25519.Sign(svr.PrivateKey, data)
	data = append(sig, data...)

	svr.handShakeBytes = data
	return nil
}

func (svr *Server) listenLoop() {
	defer svr.wg.Done()
	defer svr.ln.Close()

	var conn net.Conn
	var err error
	var tempDelay time.Duration
	maxDelay := time.Second

	for {
		select {
		case svr.pending <- struct{}{}:
			for {
				conn, err = svr.ln.Accept()

				if err != nil {
					svr.log.Error(fmt.Sprintf("tcp accept error %v", err))

					if err, ok := err.(net.Error); ok && err.Temporary() {
						if tempDelay == 0 {
							tempDelay = 5 * time.Millisecond
						} else {
							tempDelay *= 2
						}

						if tempDelay > maxDelay {
							tempDelay = maxDelay
						}

						svr.log.Info(fmt.Sprintf("tcp accept tempError, wait %s", tempDelay))

						time.Sleep(tempDelay)

						continue
					}

					return
				}

				break
			}

			go svr.setupConn(conn, inbound)
		case <-svr.term:
			return
		}
	}
}

func (svr *Server) setupConn(c net.Conn, flag connFlag) {
	ts := &conn{
		AsyncMsgConn: NewAsyncMsgConn(c, nil),
		flags:        flag,
	}

	svr.log.Info(fmt.Sprintf("begin handshake with %s", c.RemoteAddr()))

	their, err := ts.Handshake(svr.handShakeBytes)

	if err != nil {
		ts.Close(err)
		svr.log.Error(fmt.Sprintf("handshake with %s error: %v", c.RemoteAddr(), err))
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

	shouldSchedule := make(chan struct{})

	go svr.agent.scheduleTasks(svr.term, shouldSchedule)

loop:
	for {
		select {
		case <-svr.term:
			break loop
		case <-shouldSchedule:
			svr.agent.createTasks()
		case c := <-svr.addPeer:
			err := svr.checkConn(c)

			if err == nil {
				if p, err := NewPeer(c, svr.Protocols); err == nil {
					svr.peers.Add(p)

					peersCount := svr.peers.Size()
					svr.log.Info("create new peer", "ID", c.id.String(), "total", peersCount)
					monitor.LogDuration("p2p/peer", "add", int64(peersCount))

					go svr.runPeer(p)
				} else {
					svr.log.Error(fmt.Sprintf("create new peer error: %v", err))
				}
			} else {
				c.Close(err)
				svr.log.Error("cannot create new peer", "error", err)
			}

		case p := <-svr.delPeer:
			svr.peers.Del(p)

			peersCount := svr.peers.Size()
			svr.log.Info("delete peer", "ID", p.ID().String(), "total", peersCount)
			monitor.LogDuration("p2p/peer", "del", int64(peersCount))
		}
	}

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

	close(svr.term)
	svr.discv.Stop()
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
