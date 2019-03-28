// Package p2p implements the vite P2P network

package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/block"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/nat"
	"github.com/vitelabs/go-vite/p2p/network"
)

var errSvrStarted = errors.New("server has started")
var blockMinExpired = time.Minute
var blockMaxExpired = 5 * time.Minute

const blockCount = 5

func blockPolicy(t time.Time, count int) bool {
	del := time.Now().Sub(t)

	if del < blockMinExpired {
		return true
	}

	if del > blockMaxExpired {
		return false
	}

	if count < blockCount {
		return false
	}

	return true
}

// Config is the essential configuration to create a p2p.server
type Config struct {
	Discovery       bool
	Name            string
	NetID           network.ID         // which network server runs on
	MaxPeers        uint               // max peers can be connected
	MaxPendingPeers uint               // max peers waiting for connect
	MaxInboundRatio uint               // max inbound peers: MaxPeers / MaxInboundRatio
	Addr            string             // TCP and UDP listen port
	DataDir         string             // the directory for storing node table, default is "~/viteisbest/p2p"
	PeerKey         ed25519.PrivateKey // use for encrypt message, the corresponding public key use for NodeID
	ExtNodeData     []byte             // extension data for Node
	Protocols       []*Protocol        // protocols server supported
	BootNodes       []string           // nodes as discovery seed
	StaticNodes     []string           // nodes to connect
}

type Server interface {
	Start() error
	Stop()
	AddPlugin(plugin Plugin)
	Connect(id discovery.NodeID, addr *net.TCPAddr)
	Peers() []*PeerInfo
	PeersCount() uint
	NodeInfo() NodeInfo
	Available() bool
	Nodes() (urls []string)
	SubNodes(ch chan<- *discovery.Node)
	UnSubNodes(ch chan<- *discovery.Node)
	URL() string
	Config() *Config
	Block(id discovery.NodeID, ip net.IP, err error)
}

type server struct {
	config      *Config
	addr        *net.TCPAddr
	staticNodes []*discovery.Node

	running   int32          // atomic
	wg        sync.WaitGroup // Wait for all jobs done
	term      chan struct{}
	pending   chan struct{} // how many connection can wait for handshake
	addPeer   chan *transport
	delPeer   chan *Peer
	discv     discovery.Discovery
	handshake *Handshake
	peers     *PeerSet
	blockUtil *block.Block
	self      *discovery.Node
	ln        net.Listener
	nodeChan  chan *discovery.Node // sub discovery nodes
	log       log15.Logger

	rw     sync.RWMutex // for block
	dialer *net.Dialer

	plugins []Plugin
}

func (svr *server) Config() *Config {
	return svr.config
}

func New(cfg *Config) (Server, error) {
	cfg = EnsureConfig(cfg)

	// tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp", cfg.Addr)
	if err != nil {
		return nil, err
	}

	ID, err := discovery.Priv2NodeID(cfg.PeerKey)
	if err != nil {
		return nil, err
	}

	node := &discovery.Node{
		ID:  ID,
		IP:  tcpAddr.IP,
		UDP: uint16(tcpAddr.Port),
		TCP: uint16(tcpAddr.Port),
		Net: cfg.NetID,
		Ext: cfg.ExtNodeData,
	}

	svr := &server{
		config:      cfg,
		addr:        tcpAddr,
		staticNodes: parseNodes(cfg.StaticNodes),
		peers:       NewPeerSet(DefaultMinPeers),
		pending:     make(chan struct{}, cfg.MaxPendingPeers),
		addPeer:     make(chan *transport, 5),
		delPeer:     make(chan *Peer, 5),
		blockUtil:   block.New(blockPolicy),
		self:        node,
		nodeChan:    make(chan *discovery.Node, cfg.MaxPendingPeers),
		log:         log15.New("module", "p2p/server"),
		dialer: &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 10 * time.Second,
		},
	}

	if cfg.Discovery {
		svr.discv = discovery.New(&discovery.Config{
			PeerKey:   cfg.PeerKey,
			DBPath:    cfg.DataDir,
			BootNodes: parseNodes(cfg.BootNodes),
			Addr:      cfg.Addr,
			Self:      node,
			NetID:     cfg.NetID,
		})
	}

	return svr, nil
}

func (svr *server) Start() error {
	if !atomic.CompareAndSwapInt32(&svr.running, 0, 1) {
		return errSvrStarted
	}

	svr.term = make(chan struct{})

	// setHandshake in method Start, because svr.Protocols may be modified
	svr.setHandshake()

	listener, err := net.ListenTCP("tcp", svr.addr)
	if err != nil {
		return err
	}
	svr.log.Info(fmt.Sprintf("tcp listen at %s", svr.addr))
	svr.ln = listener

	svr.term = make(chan struct{})

	svr.log.Info("p2p server start")

	// mapping tcp
	svr.wg.Add(1)
	common.Go(func() {
		nat.Map(svr.term, "tcp", int(svr.self.TCP), int(svr.self.TCP), "vite p2p tcp", 0, svr.updateNode)
		svr.wg.Done()
	})

	// discovery
	if svr.config.Discovery {
		// mapping udp
		svr.wg.Add(1)
		common.Go(func() {
			nat.Map(svr.term, "udp", int(svr.self.UDP), int(svr.self.UDP), "vite p2p udp", 0, svr.updateNode)
			svr.wg.Done()
		})

		// subscribe nodes
		svr.discv.SubNodes(svr.nodeChan, true)

		err = svr.discv.Start()
		if err != nil {
			svr.ln.Close()
			return err
		}
	}

	err = svr.startPlugins()
	if err != nil {
		return err
	}

	svr.wg.Add(1)
	common.Go(svr.dialLoop)

	// tcp listener
	svr.wg.Add(1)
	common.Go(svr.listenLoop)

	// peer manager
	svr.wg.Add(1)
	common.Go(svr.loop)

	svr.log.Info("p2p server started")
	return nil
}

func (svr *server) Stop() {
	if svr.term == nil {
		return
	}

	select {
	case <-svr.term:
	default:
		svr.log.Warn("p2p server stop")

		close(svr.term)

		if svr.ln != nil {
			svr.ln.Close()
		}

		if svr.discv != nil {
			svr.discv.Stop()
			svr.discv.UnSubNodes(svr.nodeChan)
		}

		svr.wg.Wait()

		svr.log.Warn("p2p server stopped")
	}
}

func (svr *server) AddPlugin(plugin Plugin) {
	svr.plugins = append(svr.plugins, plugin)
}

func (svr *server) startPlugins() (err error) {
	for _, plugin := range svr.plugins {
		if err = plugin.Start(svr); err != nil {
			return
		}
	}
	return nil
}

func (svr *server) updateNode(addr *nat.Addr) {
	if addr.Proto == "tcp" {
		svr.self.TCP = uint16(addr.Port)
	} else {
		svr.self.UDP = uint16(addr.Port)
	}
}

func (svr *server) setHandshake() {
	config := svr.config
	cmds := make([]CmdSet, len(config.Protocols))
	for i, pt := range config.Protocols {
		cmds[i] = pt.ID
	}

	svr.handshake = &Handshake{
		Name:    config.Name,
		ID:      svr.self.ID,
		CmdSets: cmds,
		Port:    uint16(svr.addr.Port),
	}
}

func (svr *server) blocked(buf []byte) bool {
	svr.rw.RLock()
	defer svr.rw.RUnlock()

	return svr.blockUtil.Blocked(buf)
}

func (svr *server) Block(id discovery.NodeID, ip net.IP, err error) {
	svr.rw.Lock()
	defer svr.rw.Unlock()

	svr.blockUtil.Block(id[:])
	svr.blockUtil.Block(ip)
	svr.log.Warn(fmt.Sprintf("block %s@%s: %v", id, ip, err))

	if svr.discv != nil && id != discovery.ZERO_NODE_ID {
		svr.discv.Delete(id)
	}
}

func (svr *server) unblock(id discovery.NodeID, ip net.IP) {
	svr.rw.Lock()
	defer svr.rw.Unlock()

	svr.blockUtil.UnBlock(id[:])
	svr.blockUtil.UnBlock(ip)
	svr.log.Warn(fmt.Sprintf("unblock %s@%s", id, ip))
}

func (svr *server) dialStatic() {
	for _, node := range svr.staticNodes {
		svr.dial(node.ID, node.TCPAddr(), static, nil)
	}
}

func (svr *server) dialLoop() {
	defer svr.wg.Done()

	dialing := make(map[discovery.NodeID]struct{})

	var node *discovery.Node

	// connect to static node first
	svr.dialStatic()

	dialDone := make(chan discovery.NodeID, svr.config.MaxPendingPeers)

	for {
		select {
		case <-svr.term:
			return
		case node = <-svr.nodeChan:
			if _, ok := dialing[node.ID]; ok {
				break
			}

			if svr.blocked(node.ID[:]) {
				break
			}

			// will be blocked if has enough peers
			svr.peers.CheckWait()

			dialing[node.ID] = struct{}{}
			svr.dial(node.ID, node.TCPAddr(), outbound, dialDone)

		case id := <-dialDone:
			delete(dialing, id)
		}
	}
}

// when peer is disconnected, maybe we want to reconnect it.
// we can get ID and addr only from peer, but not Node
// so dial(id, addr, flag) not dial(Node, flag)
func (svr *server) dial(id discovery.NodeID, addr *net.TCPAddr, flag connFlag, done chan<- discovery.NodeID) {
	if err := svr.checkConn(id, flag); err != nil {
		if done != nil {
			done <- id
		}

		return
	}

	common.Go(func() {
		if conn, err := svr.dialer.Dial("tcp", addr.String()); err == nil {
			svr.setupConn(conn, flag, id)
		} else {
			svr.Block(id, addr.IP, err)
			svr.log.Warn(fmt.Sprintf("dial node %s@%s failed: %v", id, addr, err))
		}

		if done != nil {
			done <- id
		}
	})
}

func (svr *server) Connect(id discovery.NodeID, addr *net.TCPAddr) {
	svr.dial(id, addr, static, nil)
}

// TCPListener will be closed in method: server.Stop()
func (svr *server) listenLoop() {
	defer svr.wg.Done()

	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		select {
		case svr.pending <- struct{}{}:
			// for goroutine setupConn catch, so can`t declare them out of select
			var conn net.Conn
			var err error

			for {
				if conn, err = svr.ln.Accept(); err != nil {
					// temporary error
					if ne, ok := err.(net.Error); ok && ne.Temporary() {
						svr.log.Warn(fmt.Sprintf("listen temp error: %v", ne))

						if tempDelay == 0 {
							tempDelay = 5 * time.Millisecond
						} else {
							tempDelay *= 2
						}

						if tempDelay > maxDelay {
							tempDelay = maxDelay
						}

						time.Sleep(tempDelay)

						continue
					}

					svr.log.Warn(fmt.Sprintf("listen error: %v", err))
					return
				}

				// next Accept
				break
			}

			if addr := conn.RemoteAddr().(*net.TCPAddr); svr.blocked(addr.IP) {
				svr.log.Warn(fmt.Sprintf("%s has been blocked, will not setup", addr))
				conn.Close()
				// next pending
				<-svr.pending
			} else {
				common.Go(func() {
					svr.setupConn(conn, inbound, discovery.ZERO_NODE_ID)
					<-svr.pending
				})
			}

		case <-svr.term:
			return
		}
	}
}

func (svr *server) setupConn(c net.Conn, flag connFlag, id discovery.NodeID) {
	var err error
	if err = svr.checkHead(c); err != nil {
		svr.log.Warn(fmt.Sprintf("HeadShake with %s error: %v, block it", c.RemoteAddr(), err))
		c.Close()
		svr.Block(id, c.RemoteAddr().(*net.TCPAddr).IP, err)
		return
	}

	ts := &transport{
		Conn:  c,
		flags: flag,
	}

	if err = svr.handleTS(ts, id); err != nil {
		svr.log.Warn(fmt.Sprintf("HandShake with %s error: %v, block it", c.RemoteAddr(), err))
		ts.Close()
		svr.Block(id, c.RemoteAddr().(*net.TCPAddr).IP, err)
		return
	}

	svr.addPeer <- ts
}

func (svr *server) handleTS(ts *transport, id discovery.NodeID) error {
	// handshake data, add remoteIP and remotePort
	// handshake is not same for every peer
	handshake := *svr.handshake
	tcpAddr := ts.RemoteAddr().(*net.TCPAddr)
	handshake.RemoteIP = tcpAddr.IP
	handshake.RemotePort = uint16(tcpAddr.Port)

	their, err := ts.Handshake(svr.config.PeerKey, &handshake)

	if err != nil {
		return err
	}

	if id != discovery.ZERO_NODE_ID && their.ID != id {
		return fmt.Errorf("unmatched server ID, dial %s got %s", id, their.ID)
	}

	if err = svr.checkConn(id, ts.flags); err != nil {
		return err
	}

	ts.name = their.Name
	ts.cmdSets = their.CmdSets

	// use to describe the connection
	ts.remoteID = their.ID
	ts.remoteIP = handshake.RemoteIP
	ts.remotePort = handshake.RemotePort

	ts.localID = svr.self.ID
	ts.localIP = their.RemoteIP
	ts.localPort = their.RemotePort

	return nil
}

func (svr *server) checkHead(c net.Conn) error {
	head, err := headShake(c, &headMsg{
		Version: Version,
		NetID:   svr.config.NetID,
	})

	if err != nil {
		return err
	}

	if svr.config.NetID != head.NetID {
		return fmt.Errorf("different NetID: our %s, their %s", svr.config.NetID, head.NetID)
	}

	// todo compatibility
	if head.Version < Version {
		return fmt.Errorf("P2P version too low: our %d, their %d", Version, head.Version)
	}

	return nil
}

func (svr *server) checkConn(id discovery.NodeID, flag connFlag) error {
	if id == svr.self.ID {
		return DiscSelf
	}

	if svr.peers.Has(id) {
		return DiscAlreadyConnected
	}

	// static can be connected even if peers too many
	if flag == static {
		return nil
	}

	if uint(svr.peers.Size()) >= svr.config.MaxPeers {
		return DiscTooManyPeers
	}

	if flag.is(inbound) && uint(svr.peers.inbound) >= svr.maxInboundPeers() {
		return DiscTooManyInboundPeers
	}

	return nil
}

func (svr *server) loop() {
	defer svr.wg.Done()

	var peersCount int
	var checkTicker = time.NewTicker(10 * time.Second)
	defer checkTicker.Stop()
	run := func() {
		svr.dialStatic()
		if svr.discv != nil {
			svr.discv.More(svr.nodeChan, (DefaultMinPeers-peersCount)*4)
		}
	}

loop:
	for {
		select {
		case <-svr.term:
			break loop
		case c := <-svr.addPeer:
			err := svr.checkConn(c.remoteID, c.flags)

			if err == nil {
				var p *Peer
				if p, err = NewPeer(c, svr.config.Protocols); err == nil {
					peersCount = svr.peers.Add(p)
					svr.log.Info(fmt.Sprintf("create new peer %s, total: %d", p, peersCount))

					monitor.LogDuration("p2p/peer", "count", int64(peersCount))
					monitor.LogEvent("p2p/peer", "create")

					common.Go(func() {
						svr.runPeer(p)
					})
				}
			}

			if err != nil {
				c.Close()
				svr.Block(c.remoteID, c.remoteIP, err)
				svr.log.Warn(fmt.Sprintf("can`t create new peer: %v", err))
			}

		case p := <-svr.delPeer:
			svr.unblock(p.ID(), p.ts.remoteIP)
			peersCount = svr.peers.Del(p)
			svr.log.Error(fmt.Sprintf("delete peer %s, total: %d", p, peersCount))

			monitor.LogDuration("p2p/peer", "count", int64(peersCount))
			monitor.LogEvent("p2p/peer", "delete")

		case <-checkTicker.C:
			if peersCount < DefaultMinPeers {
				run()
			}
			svr.markPeers()
		}
	}

	svr.markPeers()

	svr.peers.DisconnectAll()
}

func (svr *server) markPeers() {
	if svr.discv != nil {
		now := time.Now()

		svr.peers.Range(func(id discovery.NodeID, p *Peer) bool {
			svr.discv.Mark(id, now.Sub(p.Created).Nanoseconds())
			return true
		})
	}
}

func (svr *server) runPeer(p *Peer) {
	err := p.run()
	if err != nil {
		svr.log.Error(fmt.Sprintf("run peer %s error: %v", p, err))
	}
	select {
	case svr.delPeer <- p:
	case <-svr.term:
	}
}

func (svr *server) Peers() []*PeerInfo {
	return svr.peers.Info()
}

func (svr *server) PeersCount() uint {
	return uint(svr.peers.Size())
}

func (svr *server) NodeInfo() NodeInfo {
	protocols := make([]string, len(svr.config.Protocols))
	for i, protocol := range svr.config.Protocols {
		protocols[i] = protocol.String()
	}

	var plugins []interface{}
	for _, plg := range svr.plugins {
		plugins = append(plugins, plg.Info())
	}

	return NodeInfo{
		ID:    svr.self.ID.String(),
		Name:  svr.config.Name,
		Url:   svr.self.String(),
		NetID: svr.config.NetID,
		Address: address{
			IP:  svr.self.IP,
			TCP: svr.self.TCP,
			UDP: svr.self.UDP,
		},
		Protocols: protocols,
		Plugins:   plugins,
	}
}

func (svr *server) URL() string {
	return svr.self.String()
}

func (svr *server) Available() bool {
	return svr.PeersCount() > 0
}

func (svr *server) maxOutboundPeers() uint {
	return svr.config.MaxPeers - svr.maxInboundPeers()
}

func (svr *server) maxInboundPeers() uint {
	return svr.config.MaxPeers / svr.config.MaxInboundRatio
}

func (svr *server) Nodes() (urls []string) {
	if svr.discv == nil {
		return nil
	}

	return svr.discv.Nodes()
}

func (svr *server) SubNodes(ch chan<- *discovery.Node) {
	if svr.discv == nil {
		return
	}
	svr.discv.SubNodes(ch, false)
}
func (svr *server) UnSubNodes(ch chan<- *discovery.Node) {
	if svr.discv == nil {
		return
	}
	svr.discv.UnSubNodes(ch)
}

// NodeInfo represent current p2p node
type NodeInfo struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Url       string        `json:"url"`
	NetID     network.ID    `json:"netId"`
	Address   address       `json:"address"`
	Protocols []string      `json:"protocols"`
	Plugins   []interface{} `json:"plugins"`
}

type address struct {
	IP  net.IP `json:"ip"`
	TCP uint16 `json:"tcp"`
	UDP uint16 `json:"udp"`
}
