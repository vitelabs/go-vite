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
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/p2p/discovery"

	"github.com/golang/protobuf/proto"

	"github.com/vitelabs/go-vite/p2p/protos"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/p2p/vnode"
)

var errP2PAlreadyRunning = errors.New("p2p is already running")
var errP2PNotRunning = errors.New("p2p is not running")
var errInvalidProtocolID = errors.New("protocol id must larger than 0")
var errProtocolExisted = errors.New("protocol has existed")
var errPeerNotExist = errors.New("peer not exist")

// Authenticator will authenticate all inbound connection whether can access our server
type Authenticator interface {
	// Authenticate the connection, connection will be disconnected if return false
	Authenticate() bool
}

// NodeInfo represent current p2p node
type NodeInfo struct {
	// ID is the hex-encoded NodeID
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	NetID     int        `json:"netId"`
	Version   int        `json:"version"`
	Address   string     `json:"address"`
	Protocols []string   `json:"protocols"`
	PeerCount int        `json:"peerCount"`
	Peers     []PeerInfo `json:"peers"`
}

type P2P interface {
	Config() Config
	Start() error
	Stop() error
	Connect(node string) error
	Ban(ip net.IP)
	Unban(ip net.IP)
	Info() NodeInfo
	Register(pt Protocol) error
}

type netutils interface {
	ban(ip net.IP)
	unban(ip net.IP)
}

type Handshaker interface {
	Handshake(conn net.Conn, level Level) (peer PeerMux, err error)
}

type basePeer interface {
	MsgWriter
	ID() vnode.NodeID
	String() string
	Address() string
	Info() PeerInfo
	Close(err PeerError) error
	Level() Level
	SetLevel(level Level) error
}

type Peer interface {
	basePeer
	State() interface{}
	SetState(state interface{})
}

type PeerMux interface {
	basePeer
	run() error
	setManager(pm peerLeveler)
}

type peerManager interface {
	register(p PeerMux) error
	changeLevel(p PeerMux, old Level) error
}

type p2p struct {
	cfg Config

	self vnode.Node

	rw sync.RWMutex

	discv discovery.Discover

	// condition for loop find, if number of peers is little than cfg.minPeers
	cond *sync.Cond
	dialer
	staticNodes []vnode.Node

	ptMap map[ProtocolID]Protocol

	peerMap   map[vnode.NodeID]PeerMux
	peerLevel map[Level][]PeerMux

	handshaker Handshaker

	netTool netutils

	server Server

	wg sync.WaitGroup

	running int32
	term    chan struct{}

	log log15.Logger
}

func New(cfg Config) P2P {
	cfg.ensure()

	log := log15.New("module", "p2p")

	ptMap := make(map[ProtocolID]Protocol)
	var p = &p2p{
		cfg:       cfg,
		ptMap:     ptMap,
		peerMap:   make(map[vnode.NodeID]PeerMux),
		peerLevel: make(map[Level][]PeerMux),
		handshaker: &handshaker{
			version: version,
			netId:   cfg.NetID,
			name:    cfg.Name,
			id:      cfg.Node.ID,
			priv:    cfg.PeerKey,
			codecFactory: &transportFactory{
				minCompressLength: 100,
				readTimeout:       readMsgTimeout,
				writeTimeout:      writeMsgTimeout,
			},
			ptMap: ptMap,
			log:   log.New("module", "handshaker"),
		},
	}

	if cfg.Discovery {
		p.discv = discovery.New(cfg.Config)
	}

	p.cond = sync.NewCond(&p.rw)

	p.server = newServer(retryStartDuration, retryStartCount, cfg.MaxPeers[Inbound], cfg.MaxPendingPeers, p.handshaker, p, cfg.ListenAddress, p.log.New("module", "server"))

	return p
}

func (p *p2p) check(peer PeerMux) error {
	id := peer.ID()

	if id == p.self.ID {
		return PeerConnectSelf
	}

	p.rw.RLock()
	defer p.rw.RUnlock()

	if _, ok := p.peerMap[id]; ok {
		return PeerAlreadyConnected
	}

	plevel := peer.Level()
	if len(p.peerLevel[plevel]) >= p.cfg.MaxPeers[plevel] {
		return PeerTooManyPeers
	}

	p.peerLevel[plevel] = append(p.peerLevel[plevel], peer)

	return nil
}

func (p *p2p) Start() (err error) {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})

		if err = p.server.Start(); err != nil {
			return err
		}

		if err = p.discv.Start(); err != nil {
			return err
		}

		p.wg.Add(1)
		go p.findLoop()

		p.wg.Add(1)
		go p.beatLoop()
	}

	return errP2PAlreadyRunning
}

func (p *p2p) Stop() error {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		close(p.term)

		p.rw.RLock()
		for _, peer := range p.peerMap {
			_ = peer.Close(PeerQuitting)
		}
		p.rw.RUnlock()

		p.wg.Wait()
	}

	return errP2PNotRunning
}

func (p *p2p) Connect(node string) error {
	n, err := vnode.ParseNode(node)
	if err != nil {
		return err
	}

	p.connect(n)
	return nil
}

func (p *p2p) Ban(ip net.IP) {
	panic("implement me")
}

func (p *p2p) Unban(ip net.IP) {
	panic("implement me")
}

func (p *p2p) Info() NodeInfo {
	return NodeInfo{
		ID:        "",
		Name:      "",
		NetID:     0,
		Version:   0,
		Address:   "",
		Protocols: nil,
		Peers:     nil,
	}
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

// register and run peer, blocked, should invoke by goroutine
func (p *p2p) register(peer PeerMux) (err error) {
	defer p.wg.Done()

	peer.setManager(p)

	err = p.check(peer)
	if err != nil {
		if pe, ok := err.(PeerError); ok {
			_ = peer.Close(pe)
		}

		return err
	}

	// run
	if err = peer.run(); err != nil {
		p.log.Error(fmt.Sprintf("peer %s run error: %v", peer, err))
	} else {
		p.log.Error(fmt.Sprintf("peer %s run done", peer))
	}

	err = p.unregister(peer.ID())

	// notify findLoop
	p.cond.Signal()

	return
}

func (p *p2p) unregister(id vnode.NodeID) (err error) {
	p.rw.Lock()
	defer p.rw.Unlock()

	if peer, ok := p.peerMap[id]; ok {
		delete(p.peerMap, id)

		level := peer.Level()
		total := len(p.peerLevel[level]) - 1
		for i, peer2 := range p.peerLevel[level] {
			if peer2.ID() == id {
				if i != total {
					copy(p.peerLevel[level][i:], p.peerLevel[level][i+1:])
				}
				p.peerLevel[level] = p.peerLevel[level][:total]
			}
		}

		return nil
	}

	return errPeerNotExist
}

func (p *p2p) changeLevel(peer PeerMux, oldLevel Level) error {
	p.rw.Lock()
	defer p.rw.Unlock()

	level := peer.Level()
	id := peer.ID()
	if _, ok := p.peerMap[id]; ok {

		// remove from oldLevel
		total := len(p.peerLevel[oldLevel]) - 1
		for i, peer2 := range p.peerLevel[oldLevel] {
			if peer2.ID() == id {
				if i != total {
					copy(p.peerLevel[oldLevel][i:], p.peerLevel[oldLevel][i+1:])
				}
				p.peerLevel[oldLevel] = p.peerLevel[oldLevel][:total]
			}
		}

		p.peerLevel[level] = append(p.peerLevel[level], peer)

		return nil
	}

	return errPeerNotExist
}

func (p *p2p) beatLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var now time.Time

	for {
		select {
		case <-p.term:
			return
		case now = <-ticker.C:
		}

		var heartBeat = &protos.HeartBeat{
			State:     make(map[int32][]byte),
			Timestamp: now.Unix(),
		}

		for pid, pt := range p.ptMap {
			heartBeat.State[int32(pid)] = pt.State()
		}

		data, err := proto.Marshal(heartBeat)
		if err != nil {
			p.log.Error(fmt.Sprintf("Failed to marshal heartbeat data: %v", err))
			continue
		}

		for _, pe := range p.peerMap {
			_ = pe.WriteMsg(Msg{
				Pid:     baseProtocolID,
				Code:    baseHeartBeat,
				Payload: data,
			})
		}
	}
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
		for len(p.peerMap) >= p.cfg.MinPeers {
			p.cond.Wait()
		}
		p.rw.RUnlock()

		if atomic.LoadInt32(&p.running) == 0 {
			return
		}

		nodes := p.discv.GetNodes()
		for _, n := range nodes {
			p.connect(n)
		}
	}
}

func (p *p2p) connect(node *vnode.Node) {
	peer, err := p.dialer.dialNode(node)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to dail %s: %v", node.String(), err))
	} else {
		p.wg.Add(1)
		go p.register(peer)
	}
}

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
*/
