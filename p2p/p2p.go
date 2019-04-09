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

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"github.com/vitelabs/go-vite/p2p/netool"
	"github.com/vitelabs/go-vite/p2p/protos"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

var errP2PAlreadyRunning = errors.New("p2p is already running")
var errP2PNotRunning = errors.New("p2p is not running")
var errInvalidProtocolID = errors.New("protocol id must larger than 0")
var errProtocolExisted = errors.New("protocol has existed")
var errPeerNotExist = errors.New("peer not exist")
var errLevelIsFull = errors.New("level is full")

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
	Info() NodeInfo
	Register(pt Protocol) error
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
	setManager(pm levelManager)
}

type peerManager interface {
	register(p PeerMux)
	changeLevel(p PeerMux, old Level) error
}

type p2p struct {
	cfg *Config

	node vnode.Node

	discv discovery.Discovery

	mu sync.Mutex
	dialer
	staticNodes []*vnode.Node

	ptMap map[ProtocolID]Protocol

	*peers

	handshaker Handshaker

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
	staticNodes := make([]*vnode.Node, 0, len(cfg.staticNodes))
	for _, u := range cfg.staticNodes {
		n, err := vnode.ParseNode(u)
		if err != nil {
			panic(err)
		}

		staticNodes = append(staticNodes, n)
	}

	log := log15.New("module", "p2p")

	ptMap := make(map[ProtocolID]Protocol)
	hkr := &handshaker{
		version: version,
		netId:   uint32(cfg.NetID),
		name:    cfg.name,
		id:      cfg.Node().ID,
		priv:    cfg.PrivateKey(),
		codecFactory: &transportFactory{
			minCompressLength: 100,
			readTimeout:       readMsgTimeout,
			writeTimeout:      writeMsgTimeout,
		},
		ptMap: ptMap,
		log:   log.New("module", "handshaker"),
	}

	var p = &p2p{
		cfg:         cfg,
		staticNodes: staticNodes,
		ptMap:       ptMap,
		peers:       newPeers(cfg.maxPeers),
		handshaker:  hkr,
		blackList:   netool.NewBlackList(strategy),
		dialer:      newDialer(5*time.Second, 5, hkr),
		log:         log,
	}

	if cfg.discover {
		p.discv = discovery.New(cfg.Config)
	}

	p.server = newServer(retryStartDuration, retryStartCount, cfg.maxPeers[Inbound], cfg.maxPendingPeers, p.handshaker, p, cfg.ListenAddress, p.log.New("module", "server"))

	return p
}

func (p *p2p) check(peer PeerMux) error {
	if peer.ID() == p.node.ID {
		return PeerConnectSelf
	}

	if pe, ok := p.peers.add(peer); ok {
		return nil
	} else {
		return pe
	}
}

func (p *p2p) Start() (err error) {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})

		if err = p.server.Start(); err != nil {
			return err
		}

		if p.cfg.discover {
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

		if p.cfg.discover {
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
	pts := make([]string, 0, len(p.ptMap))
	for _, pt := range p.ptMap {
		pts = append(pts, pt.Name())
	}

	return NodeInfo{
		ID:        p.cfg.Node().ID.String(),
		Name:      p.cfg.name,
		NetID:     p.cfg.NetID,
		Version:   version,
		Address:   p.cfg.ListenAddress,
		Protocols: pts,
		PeerCount: p.peers.count(),
		Peers:     p.peers.info(),
	}
}

func (p *p2p) Register(pt Protocol) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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
	return *p.cfg
}

// register and run peer, blocked, should invoke by goroutine
func (p *p2p) register(peer PeerMux) {
	p.wg.Add(1)
	defer p.wg.Done()

	var err error
	if err = p.check(peer); err != nil {
		if pe, ok := err.(PeerError); ok {
			_ = peer.Close(pe)
		}

		p.log.Error(fmt.Sprintf("failed to add peer %s: %v", peer, err))
		return
	}

	peer.setManager(p.peers)

	// run
	if err = peer.run(); err != nil {
		p.log.Error(fmt.Sprintf("peer %s run error: %v", peer, err))
	} else {
		p.log.Warn(fmt.Sprintf("peer %s run done", peer))
	}

	// clean
	if err = p.peers.remove(peer); err != nil {
		p.log.Warn(fmt.Sprintf("failed to unregister peer %s: %v", peer, err))
	}

	return
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
				pid:     baseProtocolID,
				Code:    baseHeartBeat,
				Payload: data,
			})
		}
	}
}

func (p *p2p) dialStatic() {
	for _, n := range p.staticNodes {
		p.connect(n)
	}
}

func (p *p2p) findLoop() {
	defer p.wg.Done()

	need := p.cfg.minPeers

	var duration = 10 * time.Second
	var maxDuration = 4 * time.Minute
	var timer = time.NewTimer(time.Hour)
	defer timer.Stop()

Loop:
	for {
		p.dialStatic()

		for {
			if p.peers.count() > p.cfg.minPeers {
				duration *= 2
				if duration > maxDuration {
					duration = maxDuration
				}
			} else {
				break
			}

			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(duration)

			select {
			case <-timer.C:
				continue
			case <-p.term:
				break Loop
			}
		}

		if atomic.LoadInt32(&p.running) == 0 {
			return
		}

		if p.cfg.discover {
			need *= 2
			max := p.peers.max()
			if need > max {
				need = max
			}

			nodes := p.discv.GetNodes(need)
			for _, n := range nodes {
				p.connect(&n)
			}
		}

		time.Sleep(duration)
	}
}

func (p *p2p) connect(node *vnode.Node) {
	if p.peers.has(node.ID) {
		return
	}

	peer, err := p.dialer.dialNode(node)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to dail %s: %v", node.String(), err))
	} else {
		go p.register(peer)
	}
}
