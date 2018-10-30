// Package p2p implements the vite P2P network

package p2p

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p/block"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/nat"
	"github.com/vitelabs/go-vite/p2p/network"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var errSvrStarted = errors.New("server has started")
var errNoNodes = errors.New("no nodes to connect")

type Discovery interface {
	Start() error
	Stop()
	SubNodes(ch chan<- *discovery.Node)
	UnSubNodes(ch chan<- *discovery.Node)
	Mark(id discovery.NodeID, lifetime int64)
	Block(id discovery.NodeID, ip net.IP)
	Need(n uint)
	Nodes() []*discovery.Node
}

type Config struct {
	Discovery       bool
	Name            string
	NetID           network.ID         // which network server runs on
	MaxPeers        uint               // max peers can be connected
	MaxPendingPeers uint               // max peers waiting for connect
	MaxInboundRatio uint               // max inbound peers: MaxPeers / MaxInboundRatio
	Port            uint               // TCP and UDP listen port
	DataDir         string             // the directory for storing node table, default is "~/viteisbest/p2p"
	PrivateKey      ed25519.PrivateKey // use for encrypt message, the corresponding public key use for NodeID
	Protocols       []*Protocol        // protocols server supported
	BootNodes       []string           // nodes as discovery seed
	StaticNodes     []string           // nodes to connect
}

type Server struct {
	*Config
	addr        *net.TCPAddr
	StaticNodes []*discovery.Node

	running   int32          // atomic
	wg        sync.WaitGroup // Wait for all jobs done
	term      chan struct{}
	pending   chan struct{} // how many connection can wait for handshake
	addPeer   chan *transport
	delPeer   chan *Peer
	discv     Discovery
	handshake *Handshake
	peers     *PeerSet
	blockList *block.Set
	self      *discovery.Node
	ln        net.Listener
	nodeChan  chan *discovery.Node // sub discovery nodes
	log       log15.Logger

	dialer *net.Dialer
}

func New(cfg *Config) (svr *Server, err error) {
	cfg = EnsureConfig(cfg)

	addr := "0.0.0.0:" + strconv.FormatUint(uint64(cfg.Port), 10)

	// tcp listener
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}

	ID, err := discovery.Priv2NodeID(cfg.PrivateKey)
	if err != nil {
		return
	}

	node := &discovery.Node{
		ID:  ID,
		IP:  tcpAddr.IP,
		UDP: uint16(tcpAddr.Port),
		TCP: uint16(tcpAddr.Port),
	}

	svr = &Server{
		Config:      cfg,
		addr:        tcpAddr,
		StaticNodes: parseNodes(cfg.StaticNodes),
		peers:       NewPeerSet(),
		pending:     make(chan struct{}, cfg.MaxPendingPeers),
		addPeer:     make(chan *transport, 1),
		delPeer:     make(chan *Peer, 1),
		blockList:   block.New(100),
		self:        node,
		nodeChan:    make(chan *discovery.Node, 10),
		log:         log15.New("module", "p2p/server"),
		dialer:      &net.Dialer{Timeout: 3 * time.Second},
	}

	if cfg.Discovery {
		// udp discover
		var udpAddr *net.UDPAddr
		udpAddr, err = net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return
		}

		svr.discv = discovery.New(&discovery.Config{
			Priv:      cfg.PrivateKey,
			DBPath:    cfg.DataDir,
			BootNodes: parseNodes(cfg.BootNodes),
			Addr:      udpAddr,
			Self:      node,
			NetID:     cfg.NetID,
		})
	}

	return
}

func (svr *Server) Start() error {
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
	if svr.Discovery {
		// mapping udp
		svr.wg.Add(1)
		common.Go(func() {
			nat.Map(svr.term, "udp", int(svr.self.UDP), int(svr.self.UDP), "vite p2p udp", 0, svr.updateNode)
			svr.wg.Done()
		})

		// subscribe nodes
		svr.discv.SubNodes(svr.nodeChan)

		err = svr.discv.Start()
		if err != nil {
			svr.ln.Close()
			return err
		}
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

func (svr *Server) Stop() {
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

func (svr *Server) updateNode(addr *nat.Addr) {
	if addr.Proto == "tcp" {
		svr.self.TCP = uint16(addr.Port)
	} else {
		svr.self.UDP = uint16(addr.Port)
	}
}

func (svr *Server) setHandshake() {
	cmdsets := make([]CmdSet, len(svr.Protocols))
	for i, pt := range svr.Protocols {
		cmdsets[i] = pt.ID
	}

	svr.handshake = &Handshake{
		Name:    svr.Name,
		ID:      svr.self.ID,
		CmdSets: cmdsets,
		Port:    uint16(svr.Port),
	}
}

func (svr *Server) dialLoop() {
	defer svr.wg.Done()

	// connect to static node first
	for _, node := range svr.StaticNodes {
		svr.dial(node.ID, node.TCPAddr(), static)
	}

	for {
		select {
		case <-svr.term:
			return
		case node := <-svr.nodeChan:
			svr.dial(node.ID, node.TCPAddr(), outbound)
		}
	}
}

// when peer is disconnected, maybe we want to reconnect it.
// we can get ID and addr only from peer, but not Node
// so dial(id, addr, flag) not dial(Node, flag)
func (svr *Server) dial(id discovery.NodeID, addr *net.TCPAddr, flag connFlag) {
	// has been blocked
	//if svr.blockList.Has(id[:]) {
	//	return
	//}

	if err := svr.checkConn(id, flag); err != nil {
		return
	}

	svr.pending <- struct{}{}
	if conn, err := svr.dialer.Dial("tcp", addr.String()); err == nil {
		common.Go(func() {
			svr.setupConn(conn, flag)
		})
	} else {
		<-svr.pending
		svr.log.Warn(fmt.Sprintf("dial node %s@%s failed: %v", id, addr, err))
		//svr.blockList.Add(id[:])
	}
}

// TCPListener will be closed in method: Server.Stop()
func (svr *Server) listenLoop() {
	defer svr.wg.Done()

	var conn net.Conn
	var err error
	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		select {
		case svr.pending <- struct{}{}:
			for {
				if conn, err = svr.ln.Accept(); err == nil {
					break
				}

				// temporary error
				if err, ok := err.(net.Error); ok && err.Temporary() {
					svr.log.Warn(fmt.Sprintf("listen temp error: %v", err))

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

				// critical error, may be return
				svr.log.Warn(fmt.Sprintf("listen error: %v", err))
				return
			}

			common.Go(func() {
				svr.setupConn(conn, inbound)
			})
		case <-svr.term:
			return
		}
	}
}

func (svr *Server) releasePending() {
	<-svr.pending
}

func (svr *Server) setupConn(c net.Conn, flag connFlag) {
	defer svr.releasePending()

	head, err := headShake(c, &headMsg{
		Version: Version,
		NetID:   svr.NetID,
	})

	if err != nil {
		c.Close()
		svr.log.Warn(fmt.Sprintf("headShake with %s error: %v", c.RemoteAddr(), err))
		return
	}

	if svr.NetID != head.NetID {
		c.Close()
		svr.log.Warn(fmt.Sprintf("different NetID %s: our %s, their %s", c.RemoteAddr(), svr.NetID, head.NetID))
		return
	}

	if head.Version < Version {
		c.Close()
		svr.log.Warn(fmt.Sprintf("different Version %s: our %s, their %s", c.RemoteAddr(), Version, head.Version))
		return
	}

	ts := &transport{
		Conn:  c,
		flags: flag,
	}

	// handshake data, add remoteIP and remotePort
	// handshake is not same for every peer
	handshake := *svr.handshake
	tcpAddr := c.RemoteAddr().(*net.TCPAddr)
	handshake.RemoteIP = tcpAddr.IP
	handshake.RemotePort = uint16(tcpAddr.Port)

	their, err := ts.Handshake(svr.PrivateKey, &handshake)

	if err != nil {
		ts.Close()
		svr.log.Warn(fmt.Sprintf("handshake with %s error: %v", c.RemoteAddr(), err))
	} else {
		ts.name = their.Name
		ts.cmdSets = their.CmdSets

		// use to discribe the connection
		ts.remoteID = their.ID
		ts.remoteIP = handshake.RemoteIP
		ts.remotePort = handshake.RemotePort

		ts.localID = svr.self.ID
		ts.localIP = their.RemoteIP
		ts.localPort = their.RemotePort

		svr.addPeer <- ts
	}
}

func (svr *Server) checkConn(id discovery.NodeID, flag connFlag) error {
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

	if uint(svr.peers.Size()) >= svr.MaxPeers {
		return DiscTooManyPeers
	}

	if flag.is(inbound) && uint(svr.peers.inbound) >= svr.maxInboundPeers() {
		return DiscTooManyInboundPeers
	}

	return nil
}

func (svr *Server) loop() {
	defer svr.wg.Done()

	var peersCount uint
loop:
	for {
		select {
		case <-svr.term:
			break loop
		case c := <-svr.addPeer:
			err := svr.checkConn(c.remoteID, c.flags)

			if err == nil {
				if p, err := NewPeer(c, svr.Protocols); err == nil {
					svr.peers.Add(p)
					peersCount = svr.peers.Size()
					svr.log.Info(fmt.Sprintf("create new peer %s, total: %d", p, peersCount))

					monitor.LogDuration("p2p/peer", "count", int64(peersCount))
					monitor.LogEvent("p2p/peer", "create")

					common.Go(func() {
						svr.runPeer(p)
					})
				} else {
					svr.log.Error(fmt.Sprintf("create new peer error: %v", err))
				}
			} else {
				c.Close()
				svr.log.Warn(fmt.Sprintf("can`t create new peer: %v", err))
			}

		case p := <-svr.delPeer:
			svr.peers.Del(p)
			peersCount = svr.peers.Size()
			svr.log.Error(fmt.Sprintf("delete peer %s, total: %d", p, peersCount))

			monitor.LogDuration("p2p/peer", "count", int64(peersCount))
			monitor.LogEvent("p2p/peer", "delete")

			if p.ts.is(static) {
				svr.dial(p.ID(), p.RemoteAddr(), static)
			}

			if peersCount == 0 && svr.discv != nil {
				svr.discv.Need(svr.MaxPeers)
			}
		}
	}

	svr.peers.Traverse(func(id discovery.NodeID, p *Peer) {
		p.Disconnect(DiscQuitting)
	})
}

func (svr *Server) runPeer(p *Peer) {
	err := p.run()
	if err != nil {
		svr.log.Error(fmt.Sprintf("run peer %s error: %v", p, err))
	}
	svr.delPeer <- p
}

func (svr *Server) Peers() []*PeerInfo {
	return svr.peers.Info()
}

func (svr *Server) PeersCount() uint {
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

func (svr *Server) Nodes() (urls []string) {
	nodes := svr.discv.Nodes()
	for _, node := range nodes {
		urls = append(urls, node.String())
	}

	return
}

// @section NodeInfo
type NodeInfo struct {
	ID        string     `json:"remoteID"`
	Name      string     `json:"name"`
	Url       string     `json:"url"`
	NetID     network.ID `json:"netId"`
	Address   *address   `json:"address"`
	Protocols []string   `json:"protocols"`
}

type address struct {
	IP  net.IP `json:"ip"`
	TCP uint16 `json:"tcp"`
	UDP uint16 `json:"udp"`
}
