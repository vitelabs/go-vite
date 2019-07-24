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

package net

import (
	"sync"
	"time"

	"github.com/vitelabs/go-vite/net/database"

	"github.com/vitelabs/go-vite/net/discovery"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/net/vnode"
)

const extLen = 32 + 64

type Connector interface {
	ConnectNode(node *vnode.Node) error
}

type finder struct {
	self       types.Address
	_selfIsSBP bool

	db *database.DB

	rw          sync.RWMutex
	targets     map[types.Address]*vnode.Node
	subId       int // table sub
	minPeers    int
	staticNodes []*vnode.Node
	resolver    interface {
		GetNodes(n int) []*vnode.Node
	}

	peers     *peerSet
	connect   Connector
	consensus Consensus

	dialing map[peerId]struct{}

	sbps map[types.Address]int64

	_subId    int
	observers map[int]func(_selfIsSBP bool)

	term chan struct{}
}

func (f *finder) FindNeighbors(fromId, target vnode.NodeID, count int) (eps []*vnode.EndPoint) {
	f.rw.RLock()
	defer f.rw.RUnlock()

	eps = make([]*vnode.EndPoint, 0, len(f.targets))
	for _, n := range f.targets {
		eps = append(eps, &n.EndPoint)
	}

	return
}

func (f *finder) SetResolver(discv interface {
	GetNodes(n int) []*vnode.Node
}) {
	f.resolver = discv
}

func (f *finder) Sub(sub discovery.Subscriber) {
	f.subId = sub.Sub(f.receiveNode)
}

func (f *finder) UnSub(sub discovery.Subscriber) {
	sub.UnSub(f.subId)
}

func newFinder(self types.Address, peers *peerSet, minPeers int, staticNodes []string, db *database.DB, connect Connector, consensus Consensus) (f *finder, err error) {
	f = &finder{
		self:      self,
		targets:   make(map[types.Address]*vnode.Node),
		peers:     peers,
		minPeers:  minPeers,
		connect:   connect,
		consensus: consensus,
		dialing:   make(map[peerId]struct{}),
		sbps:      make(map[types.Address]int64),
		observers: make(map[int]func(_selfIsSBP bool)),
		db:        db,
	}

	f.staticNodes = make([]*vnode.Node, 0, len(staticNodes))
	for _, str := range staticNodes {
		var node *vnode.Node
		node, err = vnode.ParseNode(str)
		if err != nil {
			return
		}
		f.staticNodes = append(f.staticNodes, node)
	}

	consensus.SubscribeProducers(types.SNAPSHOT_GID, "sbpn", f.receiveProducers)

	return
}

func (f *finder) sub(fn func(_selfIsSBP bool)) (subId int) {
	f.rw.Lock()
	defer f.rw.Unlock()

	subId = f._subId
	f.observers[subId] = fn
	f._subId++

	return
}

func (f *finder) unSub(subId int) {
	f.rw.Lock()
	defer f.rw.Unlock()

	delete(f.observers, subId)
}

func (f *finder) notify() {
	f.rw.RLock()
	defer f.rw.RUnlock()

	for _, fn := range f.observers {
		fn(f._selfIsSBP)
	}
}

func (f *finder) start() {
	f.term = make(chan struct{})

	// should invoked after consensus.Init()
	details, _, err := f.consensus.API().ReadVoteMap(time.Now())
	if err == nil {
		now := time.Now().Unix()
		f.rw.Lock()
		for _, d := range details {
			f.sbps[d.CurrentAddr] = now
			if d.CurrentAddr == f.self {
				f._selfIsSBP = true
			}
		}
		f.rw.Unlock()
	}

	go f.loop()
}

func (f *finder) stop() {
	select {
	case <-f.term:
	default:
		close(f.term)
	}
}

func (f *finder) selfIsSBP() bool {
	return f._selfIsSBP
}

func (f *finder) isSBP(addr types.Address) bool {
	f.rw.RLock()
	defer f.rw.RUnlock()

	_, ok := f.sbps[addr]
	return ok
}

func (f *finder) clean() {
	f.consensus.UnSubscribe(types.SNAPSHOT_GID, "sbpn")
}

func (f *finder) receiveProducers(event consensus.ProducersEvent) {
	now := time.Now().Unix()

	f.rw.Lock()

	var selfIsSBP bool
	for _, addr := range event.Addrs {
		if addr == f.self {
			selfIsSBP = true
		}
	}
	f._selfIsSBP = selfIsSBP

	for _, addr := range event.Addrs {
		f.sbps[addr] = now

		if node, ok := f.targets[addr]; ok {
			if p := f.peers.get(node.ID); p != nil {
				_ = p.SetSuperior(true)
				continue
			}

			if f._selfIsSBP {
				f.dial(node)
			}
		}
	}

	f.rw.Unlock()

	f.notify()
}

func (f *finder) dial(node *vnode.Node) {
	if _, ok := f.dialing[node.ID]; ok {
		return
	}
	f.dialing[node.ID] = struct{}{}
	go f.doDial(node)
}

func (f *finder) doDial(node *vnode.Node) {
	_ = f.connect.ConnectNode(node)

	f.rw.Lock()
	delete(f.dialing, node.ID)
	f.rw.Unlock()
}

func (f *finder) receiveNode(node *vnode.Node) {
	addr, ok := parseNodeExt(node)
	if ok {
		f.rw.Lock()
		f.targets[addr] = node
		f.rw.Unlock()
	}

	if f.total() < f.minPeers {
		f.rw.Lock()
		f.dial(node)
		f.rw.Unlock()
	}
}

func (f *finder) total() int {
	f.rw.RLock()
	defer f.rw.RUnlock()

	return f.peers.countWithoutSBP() + len(f.dialing)
}

func (f *finder) loop() {
	checkTicker := time.NewTicker(5 * time.Second)
	defer checkTicker.Stop()

	f.rw.Lock()
	nodes := f.db.ReadMarkNodes(30) // more than sbp count
	for _, node := range nodes {
		f.dial(node)
	}
	f.rw.Unlock()

	for {
		select {
		case <-checkTicker.C:
			f.rw.Lock()
			for _, n := range f.staticNodes {
				f.dial(n)
			}

			if f._selfIsSBP {
				for _, t := range f.targets {
					f.dial(t)
				}
			}
			f.rw.Unlock()

			if f.resolver != nil {
				total := f.total()
				if total < f.minPeers {
					nodes = f.resolver.GetNodes((f.minPeers - total) * 2)
					f.rw.Lock()
					for _, node := range nodes {
						f.dial(node)
					}
					f.rw.Unlock()
				}
			}
		case <-f.term:
			return
		}
	}
}

func setNodeExt(mineKey ed25519.PrivateKey, node *vnode.Node) {
	// minePUB + minePriv.Sign(node.ID)
	node.Ext = make([]byte, extLen)
	copy(node.Ext[:32], mineKey.PubByte())
	sign := ed25519.Sign(mineKey, node.ID.Bytes())
	copy(node.Ext[32:], sign)
}

func parseNodeExt(node *vnode.Node) (addr types.Address, ok bool) {
	if len(node.Ext) < extLen {
		ok = false
		return
	}

	pub := node.Ext[:32]
	ok = ed25519.Verify(pub, node.ID.Bytes(), node.Ext[32:])
	if ok {
		addr = types.PubkeyToAddress(pub)
	}

	return
}
