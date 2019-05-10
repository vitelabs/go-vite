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
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/netool"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

var errP2PAlreadyRunning = errors.New("p2p is already running")
var errP2PNotRunning = errors.New("p2p is not running")
var errPeerNotExist = errors.New("peer not exist")
var errLevelIsFull = errors.New("level is full")

var p2pLog = log15.New("module", "p2p")

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
	PeerCount int        `json:"peerCount"`
	Peers     []PeerInfo `json:"peers"`
}

type P2P interface {
	Config() Config
	Start() error
	Stop() error
	Connect(node string) error
	ConnectNode(node *vnode.Node) error
	Info() NodeInfo
	Register(pt Protocol) error
	Discovery() discovery.Discovery
	Node() *vnode.Node
}

type Handshaker interface {
	Handshake(c Codec, level Level) (peer PeerMux, err error)
}

type basePeer interface {
	MsgWriter
	ID() vnode.NodeID
	String() string
	Address() net.Addr
	Info() PeerInfo
	Close(err error) error
	Level() Level
	SetLevel(level Level) error
	Height() uint64
	Head() types.Hash
	SetHead(head types.Hash, height uint64)
	FileAddress() string
	weight() int64
}

type Peer interface {
	basePeer
}

type PeerMux interface {
	basePeer
	run() error
	setManager(pm levelManager)
}

type peerManager interface {
	register(p PeerMux)
	changeLevel(p PeerMux, old Level) error
}

type p2p struct {
	cfg *Config

	node *vnode.Node

	discv discovery.Discovery

	db *nodeDB

	mu sync.Mutex
	dialer
	staticNodes []*vnode.Node

	protocol Protocol

	*peers

	handshaker *handshaker

	blackList netool.BlackList

	server Server

	wg sync.WaitGroup

	running int32
	term    chan struct{}

	log log15.Logger
}

func strategy(t time.Time, count int) bool {
	now := time.Now()

	if now.Sub(t) < 5*time.Second {
		return true
	}

	if now.Sub(t) > 5*time.Minute {
		return false
	}

	if count > 10 {
		return true
	}

	return false
}

func New(cfg *Config) P2P {
	staticNodes := make([]*vnode.Node, 0, len(cfg.StaticNodes))
	for _, u := range cfg.StaticNodes {
		n, err := vnode.ParseNode(u)
		if err != nil {
			panic(err)
		}

		staticNodes = append(staticNodes, n)
	}

	hkr := &handshaker{
		version:     version,
		netId:       uint32(cfg.NetID),
		name:        cfg.Name,
		id:          cfg.Node().ID,
		genesis:     types.Hash{},
		fileAddress: cfg.fileAddress,
		priv:        cfg.PrivateKey(),
		protocol:    nil, // will be set when protocol registered
		log:         p2pLog.New("module", "handshaker"),
	}

	codecFactory := &transportFactory{
		minCompressLength: 100,
		readTimeout:       readMsgTimeout,
		writeTimeout:      writeMsgTimeout,
	}

	var p = &p2p{
		cfg:         cfg,
		staticNodes: staticNodes,
		peers:       newPeers(cfg.maxPeers),
		handshaker:  hkr,
		blackList:   netool.NewBlackList(strategy),
		dialer:      newDialer(5*time.Second, 5, hkr, codecFactory),
		log:         p2pLog,
		node:        cfg.Node(),
	}

	// open database
	var err error
	p.db, err = newNodeDB(path.Join(cfg.DataDir, DBDirName), 1, p.node.ID)
	if err != nil {
		panic(fmt.Errorf("failed to create database: %v", err))
	}

	if cfg.Discover {
		p.discv = discovery.New(cfg.Config, p.db)
	}

	p.server = newServer(retryStartDuration, retryStartCount, cfg.maxPeers[Inbound], cfg.MaxPendingPeers, p.handshaker, p, cfg.ListenAddress, p.blackList, codecFactory)

	return p
}

func (p *p2p) Node() *vnode.Node {
	return p.node
}

func (p *p2p) Discovery() discovery.Discovery {
	return p.discv
}

// add success return true
func (p *p2p) tryAdd(peer PeerMux) (PeerError, bool) {
	if peer.ID() == p.node.ID {
		return PeerConnectSelf, false
	}

	return p.peers.add(peer)
}

func (p *p2p) Start() (err error) {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})

		if err = p.server.Start(); err != nil {
			return err
		}

		if p.cfg.Discover {
			if err = p.discv.Start(); err != nil {
				return err
			}
		}

		p.wg.Add(1)
		go p.findLoop()

		p.wg.Add(1)
		go p.beatLoop()

		return nil
	}

	return errP2PAlreadyRunning
}

func (p *p2p) Stop() (err error) {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		close(p.term)

		p.peers.close()

		p.wg.Wait()

		if p.cfg.Discover {
			err = p.discv.Stop()
		}

		err = p.server.Stop()

		return
	}

	return errP2PNotRunning
}

func (p *p2p) Connect(node string) error {
	n, err := vnode.ParseNode(node)
	if err != nil {
		return err
	}

	return p.ConnectNode(n)
}

func (p *p2p) Info() NodeInfo {
	return NodeInfo{
		ID:        p.cfg.Node().ID.String(),
		Name:      p.cfg.Name,
		NetID:     p.cfg.NetID,
		Version:   version,
		Address:   p.cfg.ListenAddress,
		PeerCount: p.peers.count(),
		Peers:     p.peers.info(),
	}
}

func (p *p2p) Register(pt Protocol) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.protocol != nil {
		return errors.New("has protocol")
	}

	p.protocol = pt
	p.handshaker.protocol = pt

	return nil
}

func (p *p2p) Config() Config {
	return *p.cfg
}

// register and run peer, blocked, should invoke by goroutine
func (p *p2p) register(peer PeerMux) {
	p.wg.Add(1)
	defer p.wg.Done()

	if pe, ok := p.tryAdd(peer); !ok {
		_ = peer.Close(pe)

		p.log.Error(fmt.Sprintf("failed to add peer %s: %v", peer, pe))
		return
	}

	peer.setManager(p.peers)
	p.log.Info(fmt.Sprintf("register peer %s, total: %d", peer, p.peers.count()))

	var err error
	// run
	if err = peer.run(); err != nil {
		p.log.Error(fmt.Sprintf("peer %s run error: %v", peer, err))
		_ = peer.Close(err)

		if pe, ok := err.(PeerError); ok {
			if pe != PeerQuitting {
				p.banPeer(peer)
			}
		} else {
			p.banPeer(peer)
		}
	} else {
		p.log.Warn(fmt.Sprintf("peer %s run done", peer))
		_ = peer.Close(PeerQuitting)
	}

	// clean
	if err = p.peers.remove(peer); err != nil {
		p.log.Warn(fmt.Sprintf("failed to unregister peer %s: %v", peer, err))
	}

	return
}

func (p *p2p) banPeer(peer basePeer) {
	p.blackList.Ban(peer.ID().Bytes())
	if addr, ok := peer.Address().(*net.TCPAddr); ok {
		p.blackList.Ban(addr.IP)
	}
}

func (p *p2p) beatLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.term:
			return
		case <-ticker.C:
		}

		for _, pe := range p.peers.peers() {
			_ = pe.WriteMsg(Msg{
				Code:    CodeHeartBeat,
				Payload: p.protocol.State(),
			})
		}
	}
}

func (p *p2p) connectStaticNodes() {
	for _, n := range p.staticNodes {
		_ = p.ConnectNode(n)
	}
}

func (p *p2p) findLoop() {
	defer p.wg.Done()

	need := p.cfg.MinPeers

	var initDuration = 10 * time.Second
	var maxDuration = 160 * time.Second
	var duration = initDuration
	var timer = time.NewTimer(initDuration)
	defer timer.Stop()

	var markChan <-chan time.Time
	var db *nodeDB

	if p.db != nil {
		db = p.db
		nodes := db.RetrieveEndPoints(p.cfg.MinPeers)
		for _, n := range nodes {
			err := p.ConnectNode(n)
			if err != nil {
				db.RemoveEndPoint(n.ID)
			}
		}

		markTicker := time.NewTicker(time.Minute)
		markChan = markTicker.C
		defer markTicker.Stop()
	}

Loop:
	for {
		p.connectStaticNodes()

		select {
		case <-timer.C:
			if p.peers.count() < p.cfg.MinPeers && p.cfg.Discover {
				need *= 2
				max := p.peers.max()
				if need > max {
					need = max
				}

				nodes := p.discv.GetNodes(need)
				for _, n := range nodes {
					_ = p.ConnectNode(n)
				}
			}

			if duration < maxDuration {
				duration *= 2
			} else {
				duration = initDuration
			}

			timer.Reset(duration)

		case <-markChan:
			ps := p.peers.peers()
			for _, peer := range ps {
				addr := peer.Address().String()
				ep, err := vnode.ParseEndPoint(addr)
				if err != nil {
					continue
				}

				db.StoreEndPoint(peer.ID(), ep, peer.weight())
			}

		case <-p.term:
			break Loop
		}
	}
}

func (p *p2p) ConnectNode(node *vnode.Node) error {
	if node.ID == p.node.ID {
		return PeerConnectSelf
	}

	if p.peers.has(node.ID) {
		return PeerAlreadyConnected
	}

	if p.blackList.Banned(node.ID.Bytes()) || p.blackList.Banned(node.EndPoint.Host) {
		return PeerBanned
	}

	peer, err := p.dialer.dialNode(node)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to dail %s: %v", node.String(), err))
		return err
	}

	go p.register(peer)

	return nil
}
