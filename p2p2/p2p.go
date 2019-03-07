/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

// Package p2p implements the vite P2P network

package p2p

import (
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/p2p2/vnode"

	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p/network"
)

var errInvalidProtocolID = errors.New("protocol id must larger than 0")
var errProtocolExisted = errors.New("protocol has existed")

// Config is the essential configuration to create a p2p server
type Config struct {
	Discovery bool // whether discover other nodes in the networks

	Name  string     // our node name, NO need to be unique in the whole network, just for readability
	NetID network.ID // which network server runs on

	MaxPeers        int // max peers can be connected
	minPeers        int // server will keep finding nodes until number of peers is larger than `minPeers`
	MaxPendingPeers int // max peers can be connect concurrently, for defence DDOS
	MaxInboundRatio int // max inbound peers: MaxPeers / MaxInboundRatio

	ListenAddress string // TCP and UDP listen address
	PublicAddress string // our public address, can be access by other nodes

	DataDir string // the directory for storing p2p data, like nodes

	PeerKey   ed25519.PrivateKey // use for encrypt message, the corresponding public key use for NodeID
	PublicKey ed25519.PublicKey  // our miner public key

	BootNodes   []string // nodes as discovery seed
	StaticNodes []string // nodes to connect

	dialConcurr int // dial node concurrently
}

// NodeInfo represent current p2p node
type NodeInfo struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	NetID     network.ID `json:"netId"`
	Version   int        `json:"version"`
	Address   string     `json:"address"`
	Protocols []string   `json:"protocols"`
	Peers     []PeerInfo `json:"peers"`
}

type P2P interface {
	Config() Config
	Start() error
	Stop() error
	Connect(address string) error
	Ban(ip net.IP)
	Unban(ip net.IP)
	Info() NodeInfo
	Register(pt Protocol) error
}

type discover interface {
	Start() error
	Stop() error
	// GetNodes get a batch nodes to connect
	GetNodes() []vnode.Node
	// VerifyNode if id or addr or other info get from TCP is different with UDP
	VerifyNode(id vnode.NodeID, addr string)
}

type netutils interface {
	ban(ip net.IP)
	unban(ip net.IP)
}

type handshaker interface {
	initiateHandshake(c net.Conn) (peer Peer, err error)
	receiveHandshake(c net.Conn) (peer Peer, err error)
}

type Peer interface {
	MsgWriter
	ID() string
	String() string
	Info() PeerInfo
	is(flag connFlag) bool
	run() error
	Close(err PeerError) error
}

type peerManager interface {
	register(p Peer)
	check(id vnode.NodeID, flag connFlag) error
}

type p2p struct {
	cfg Config

	self vnode.Node

	rw sync.RWMutex

	discv discover

	mode vnode.NodeMode

	// condition for loop find, if number of peers is little than cfg.minPeers
	cond sync.Cond
	dialer
	staticNodes []vnode.Node

	ptMap map[ProtocolID]Protocol

	peerMap map[string]Peer

	netTool netutils

	wg sync.WaitGroup

	log log15.Logger
}

func (p *p2p) check(id string, addr string, flag connFlag) error {
	if id == p.self.ID.String() {
		return PeerConnectSelf
	}

	p.rw.RLock()
	defer p.rw.RUnlock()
	if _, ok := p.peerMap[id]; ok {
		return PeerAlreadyConnected
	}

	if flag.is(prior) {
		return nil
	}

	if len(p.peerMap) > p.cfg.MaxPeers {
		return PeerTooManyPeers
	}

	if flag.is(inbound) {
		in := 0
		for _, pe := range p.peerMap {
			if pe.is(inbound) {
				in++
			}
		}

		if in < p.maxInbound() {
			return nil
		}

		return PeerTooManyInboundPeers
	}

	return nil
}

func (p *p2p) Start() error {
	panic("implement me")
}

func (p *p2p) Stop() error {
	panic("implement me")
}

func (p *p2p) Connect(address string) error {
	panic("implement me")
}

func (p *p2p) Ban(ip net.IP) {
	panic("implement me")
}

func (p *p2p) Unban(ip net.IP) {
	panic("implement me")
}

func (p *p2p) Info() NodeInfo {
	panic("implement me")
}

func (p *p2p) maxInbound() int {
	return p.cfg.MaxPeers / p.cfg.MaxInboundRatio
}

func New() P2P {
	return &p2p{}
}

func (p *p2p) Register(pt Protocol) error {
	p.rw.Lock()
	defer p.rw.Unlock()

	pid := pt.ID()

	if pid < 1 {
		return errInvalidProtocolID
	}

	if _, ok := p.ptMap[pid]; ok {
		return errProtocolExisted
	}

	p.ptMap[pid] = pt

	return nil
}

func (p *p2p) Config() Config {
	return p.cfg
}

func (p *p2p) register(peer Peer) {
	defer p.wg.Done()

	id := peer.ID()

	// add
	p.rw.Lock()
	if _, ok := p.peerMap[id]; ok {
		panic(fmt.Errorf("peer %s already exist", peer))
	} else {
		p.peerMap[id] = peer
	}
	p.rw.Unlock()

	// run
	if err := peer.run(); err != nil {
		p.log.Error(fmt.Sprintf("peer %s run error: %v", peer, err))
	} else {
		p.log.Error(fmt.Sprintf("peer %s run done", peer))
	}

	// remove
	p.rw.Lock()
	delete(p.peerMap, id)
	p.rw.Unlock()

	// notify findLoop
	p.cond.Signal()
}

func (p *p2p) count() int {
	p.rw.RLock()
	defer p.rw.RUnlock()
	return len(p.peerMap)
}

func (p *p2p) findLoop() {
	defer p.wg.Done()

	for {
		p.rw.RLock()
		for len(p.peerMap) >= p.cfg.minPeers {
			p.cond.Wait()
		}
		p.rw.RUnlock()

		nodes := p.discv.GetNodes()
		for _, node := range nodes {
			peer, err := p.dialer.dialNode(node)
			if err != nil {
				p.log.Error(fmt.Sprintf("failed to dail %s: %v", node.Host(), err))
			} else {
				p.wg.Add(1)
				go p.register(peer)
			}
		}
	}
}

//type Server interface {
//	Start() error
//	Stop()
//	AddPlugin(plugin Plugin)
//	Connect(id discovery.NodeID, addr *net.TCPAddr)
//	Peers() []*PeerInfo
//	PeersCount() uint
//	NodeInfo() NodeInfo
//	Available() bool
//	Nodes() (urls []string)
//	SubNodes(ch chan<- *discovery.Node)
//	UnSubNodes(ch chan<- *discovery.Node)
//	URL() string
//	Config() *Config
//	Block(id discovery.NodeID, ip net.IP, err error)
//}

/*
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
		peers:       NewPeerSet(),
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
	var checkTicker = time.NewTicker(30 * time.Second)
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
					svr.peers.Add(p)
					peersCount = svr.peers.Size()
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
			svr.peers.Del(p)
			peersCount = svr.peers.Size()
			svr.log.Error(fmt.Sprintf("delete peer %s, total: %d", p, peersCount))

			monitor.LogDuration("p2p/peer", "count", int64(peersCount))
			monitor.LogEvent("p2p/peer", "delete")

			if p.ts.is(static) {
				svr.dial(p.ID(), p.RemoteAddr(), static, nil)
			}

		case <-checkTicker.C:
			if peersCount < DefaultMinPeers {
				run()
			}
			svr.markPeers()
		}
	}

	svr.peers.DisconnectAll()

	svr.markPeers()
}
*/
