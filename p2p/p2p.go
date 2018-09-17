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
	"sync"
	"time"
)

var p2pServerLog = log15.New("module", "p2p/server")

const (
	defaultMaxPeers             = 50
	defaultDialTimeout          = 10 * time.Second
	defaultMaxPendingPeers uint = 20
	defaultMaxActiveDail   uint = 16
)

var errSvrStopped = errors.New("Server has stopped")

type Discovery interface {
	Lookup(discovery.NodeID) []*discovery.Node
	Resolve(discovery.NodeID) *discovery.Node
	RandomNodes([]*discovery.Node) int
	ID() discovery.NodeID
	Start()
	Stop()
	SetNode(ip net.IP, udp, tcp uint16)
	Self() *discovery.Node
}

type Config struct {
	Name            string
	NetID           NetworkID          // which network server runs on
	MaxPeers        uint               // max peers can be connected
	MaxPendingPeers uint               // max peers waiting for connect
	MaxInboundRatio uint               // max inbound peers: MaxPeers / MaxInboundRatio
	Addr            string             // TCP listen address
	Database        string             // the directory for storing node table
	PrivateKey      ed25519.PrivateKey // use for encrypt message, the corresponding public key use for NodeID
	Protocols       []*Protocol        // protocols server supported
	BootNodes       []*discovery.Node
}

type Server struct {
	*Config

	running bool

	lock sync.Mutex

	// Wait for all jobs done
	wg sync.WaitGroup

	Dialer *NodeDailer

	// TCP listener
	listener *net.TCPListener

	// Indicate whether the server has term. If term, zero-value can be read from this channel.
	term chan struct{}

	pending chan struct{}

	addPeer chan *conn

	delPeer chan *Peer

	discv Discovery

	ourHandshake *Handshake

	BootNodes []*discovery.Node

	log log15.Logger

	peers *PeerSet

	blockList *block.CuckooSet
}

func New(cfg Config) *Server {
	//if p2pCfg.PrivateKey != "" {
	//	priv, err := hex.DecodeString(p2pCfg.PrivateKey)
	//	if err == nil {
	//		cfg.PrivateKey = ed25519.PrivateKey(priv)
	//		cfg.PublicKey = cfg.PrivateKey.PubByte()
	//	} else {
	//		p2pServerLog.Error("privateKey decode", "err", err)
	//		cfg.PublicKey, cfg.PrivateKey, err = getServerKey(cfg.Database)
	//	}
	//} else {
	//	cfg.PublicKey, cfg.PrivateKey, err = getServerKey(cfg.Database)
	//}

	safeCfg := EnsureConfig(cfg)

	svr := &Server{
		Config:    safeCfg,
		log:       p2pServerLog.New("module", "p2p/server"),
		peers:     new(PeerSet),
		term:      make(chan struct{}),
		pending:   make(chan struct{}, cfg.MaxPendingPeers),
		addPeer:   make(chan *conn),
		delPeer:   make(chan *Peer),
		BootNodes: addFirmNodes(cfg.BootNodes),
		blockList: block.NewCuckooSet(1000),
	}

	if svr.Dialer == nil {
		svr.Dialer = &NodeDailer{
			&net.Dialer{
				Timeout: defaultDialTimeout,
			},
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

func (svr *Server) Node() *discovery.Node {
	return svr.discv.Self()
}

func (svr *Server) ID() discovery.NodeID {
	return svr.Node().ID
}

func (svr *Server) NodeInfo() *NodeInfo {
	// todo

	protocols := make([]string, len(svr.Protocols))
	for i, protocol := range svr.Protocols {
		protocols[i] = protocol.String()
	}

	return &NodeInfo{
		ID:   svr.ID().String(),
		Name: svr.Name,
		Url:  svr.Node().String(),
		IP:   svr.Addr,
		Ports: ports{
			Discovery: 0,
			Listener:  0,
		},
		Address:   "",
		Protocols: protocols,
	}
}

// the first item is self url
func (svr *Server) Topology() *Topo {
	count := svr.PeersCount()

	topo := &Topo{
		Pivot: svr.Node().String(),
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

func (svr *Server) MaxOutboundPeers() uint {
	return svr.MaxPeers - svr.MaxInboundPeers()
}

func (svr *Server) MaxInboundPeers() uint {
	return svr.MaxPeers / svr.MaxInboundRatio
}

func (svr *Server) Start() error {
	svr.lock.Lock()
	if svr.running {
		svr.lock.Unlock()
		return nil
	}
	svr.running = true
	svr.lock.Unlock()

	p2pServerLog.Info("p2p server start")

	// udp discover
	udpAddr, err := net.ResolveUDPAddr("udp", svr.Addr)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	svr.discv = discovery.New(&discovery.Config{
		Priv:      svr.PrivateKey,
		DBPath:    svr.Database,
		BootNodes: svr.BootNodes,
		Conn:      conn,
	})

	svr.setHandshake()

	go svr.discv.Start()

	// tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp", svr.Addr)
	if err != nil {
		return err
	}
	go svr.Listen(tcpAddr)

	svr.wg.Add(1)
	go svr.mapping(tcpAddr.Port)

	// task loop
	svr.wg.Add(1)
	go svr.loop()
	p2pServerLog.Info("p2p server started")
	return nil
}

func (svr *Server) mapping(lport int) {
	out := make(chan *nat.Addr)
	go nat.Map(svr.term, "tcp", lport, lport, "vite p2p", 0, out)

loop:
	for {
		select {
		case <-svr.term:
			break loop
		case addr := <-out:
			if addr.IsValid() {
				svr.discv.SetNode(nil, 0, uint16(addr.Port))
			}
		}
	}

	svr.wg.Done()
}

func (svr *Server) setHandshake() {
	cmdsets := make([]*CmdSet, len(svr.Protocols))
	for i, p := range svr.Protocols {
		cmdsets[i] = p.CmdSet()
	}
	svr.ourHandshake = &Handshake{
		Version: Version,
		Name:    svr.Name,
		NetID:   svr.NetID,
		ID:      svr.Node().ID,
		CmdSets: cmdsets,
	}
}

func (svr *Server) Listen(addr *net.TCPAddr) {
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		svr.log.Crit("tcp listen", "err", err)
	} else {
		svr.log.Info("tcp listening", "addr", addr.String())
	}

	svr.listener = listener

	svr.wg.Add(1)
	go svr.handleConn()
}

func (svr *Server) handleConn() {
	defer svr.wg.Done()

	var conn net.Conn
	var err error

	for {
		select {
		case svr.pending <- struct{}{}:
			for {
				conn, err = svr.listener.Accept()

				if err != nil {
					svr.log.Error("tcp accept error", "error", err)
					continue
				}
				break
			}
			go svr.SetupConn(conn, inbound)
		case <-svr.term:
			close(svr.pending)
		}
	}
}

func (svr *Server) SetupConn(c net.Conn, flag connFlag) {
	svr.log.Info("new tcp conn", "from", c.RemoteAddr().String())

	defer func() {
		<-svr.pending
	}()

	ts := &conn{
		fd:        c,
		transport: newProtoX(c),
		flags:     flag,
		term:      make(chan struct{}),
	}

	svr.log.Info("begin handshake", "with", c.RemoteAddr().String())

	peerhandshake, err := ts.Handshake(svr.ourHandshake)

	if err != nil {
		svr.log.Error("handshake error", "with", c.RemoteAddr(), "error", err)
		ts.close(err)
	} else {
		ts.id = peerhandshake.ID
		ts.name = peerhandshake.Name
		ts.cmdSets = peerhandshake.CmdSets

		svr.log.Info(fmt.Sprintf("handshake with %s@%s\n", ts.id, c.RemoteAddr()))
	}

	svr.addPeer <- ts
}

func (svr *Server) CheckConn(peers map[discovery.NodeID]*Peer, c *conn, passivePeersCount uint) error {
	if uint(len(peers)) >= svr.MaxPeers {
		return DiscTooManyPeers
	}
	if passivePeersCount >= svr.MaxInboundPeers() {
		return DiscTooManyPassivePeers
	}
	if peers[c.id] != nil {
		return DiscAlreadyConnected
	}
	if c.id == svr.discv.ID() {
		return DiscSelf
	}
	return nil
}

type blockNode struct {
	node      *discovery.Node
	blockTime time.Time
}

var defaultBlockTimeout = 2 * time.Minute
var topoTicker = time.Minute

func (svr *Server) loop() {
	defer svr.wg.Done()

	// broadcast topo to peers
	if svr.NetID == MainNet {
		topoTicker = 10 * time.Minute
	}
	topoTicker := time.NewTicker(topoTicker)
	defer topoTicker.Stop()

	dm := NewDialManager(svr.discv, svr.MaxOutboundPeers(), svr.BootNodes)
	peers := make(map[discovery.NodeID]*Peer)
	taskHasDone := make(chan Task, defaultMaxActiveDail)

	var passivePeersCount uint = 0
	var activeTasks []Task
	var taskQueue []Task

	blocknodes := make(map[discovery.NodeID]*blockNode)
	cleanBlockTicker := time.NewTicker(defaultBlockTimeout)
	defer cleanBlockTicker.Stop()

	delActiveTask := func(t Task) {
		for i, at := range activeTasks {
			if at == t {
				activeTasks = append(activeTasks[:i], activeTasks[i+1:]...)
			}
		}
	}
	runTasks := func(ts []Task) (rest []Task) {
		i := 0
		for ; uint(len(activeTasks)) < defaultMaxActiveDail && i < len(ts); i++ {
			t := ts[i]
			go func() {
				t.Perform(svr)
				taskHasDone <- t
			}()
			activeTasks = append(activeTasks, t)
		}
		return ts[i:]
	}
	scheduleTasks := func() {
		taskQueue = runTasks(taskQueue)
		p2pServerLog.Info("server run tasks", "tasks", len(taskQueue))
		if uint(len(activeTasks)) < defaultMaxActiveDail {
			newTasks := dm.CreateTasks(peers, blocknodes)
			if len(newTasks) > 0 {
				taskQueue = append(taskQueue, runTasks(newTasks)...)
			}
		}
	}

loop:
	for {
		scheduleTasks()

		select {
		case <-svr.term:
			break loop
		case t := <-taskHasDone:
			dm.TaskDone(t)
			delActiveTask(t)
		case c := <-svr.addPeer:
			err := svr.CheckConn(peers, c, passivePeersCount)
			if err == nil {
				if p, err := NewPeer(c, svr.Protocols); err != nil {
					peers[p.ID()] = p
					svr.log.Info("create new peer", "ID", c.id.String())
					monitor.LogDuration("p2p/peer", "add", int64(len(peers)))

					go svr.runPeer(p)

					if c.is(inbound) {
						passivePeersCount++
					}
				}
			} else {
				c.close(err)
				svr.log.Error("create new peer error", "error", err)
			}
		case p := <-svr.delPeer:
			delete(peers, p.ID())
			svr.log.Info("delete peer", "ID", p.ID().String())
			monitor.LogDuration("p2p/peer", "del", int64(len(peers)))

			if p.rw.is(inbound) {
				passivePeersCount--
			}
		case <-topoTicker.C:
			topo := svr.Topology()
			go svr.peers.Traverse(func(id discovery.NodeID, p *Peer) {
				err := Send(p.rw, baseProtocolCmdSet, topoCmd, topo)
				if err != nil {
					p.protoErr <- err
				}
			})
		}
	}

	svr.log.Info("out of tcp task loop")

	if svr.discv != nil {
		svr.discv.Stop()
	}

	for _, p := range peers {
		p.Disconnect(DiscQuitting)
	}

	// wait for peers work down.
	for p := range svr.delPeer {
		delete(peers, p.ID())
	}
}

func (svr *Server) runPeer(p *Peer) {
	err := p.start()
	if err != nil {
		svr.log.Error("run peer error", "error", err)
	}
	svr.peers.Del(p)
}

func (svr *Server) Stop() {
	svr.lock.Lock()
	defer svr.lock.Unlock()

	if !svr.running {
		return
	}

	svr.running = false

	if svr.listener != nil {
		svr.listener.Close()
	}

	if svr.discv != nil {
		svr.discv.Stop()
	}

	close(svr.term)
	svr.wg.Wait()
}

// @section Dialer
type NodeDailer struct {
	*net.Dialer
}

func (d *NodeDailer) DailNode(target *discovery.Node) (net.Conn, error) {
	p2pServerLog.Info("tcp dial", "node", target)
	return d.Dialer.Dial("tcp", target.TCPAddr().String())
}

// @section NodeInfo
type NodeInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Url       string    `json:"url"`
	NetID     NetworkID `json:"netId"`
	IP        string    `json:"ip"`
	Ports     ports     `json:"ports"`
	Address   string    `json:"address"`
	Protocols []string  `json:"protocols"`
}

type ports struct {
	Discovery uint16 `json:"discovery"`
	Listener  uint16 `json:"listener"`
}
