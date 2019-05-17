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

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/crypto/ed25519"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/vnode"
)

const extLen = 32 + 64

type target struct {
	*vnode.Node
	address types.Address
}

type Connector interface {
	ConnectNode(node *vnode.Node) error
}

type sbpn struct {
	self      types.Address
	rw        sync.RWMutex
	targets   map[types.Address]*target
	peers     *peerSet
	connect   Connector
	consensus Consensus
	dialing   map[peerId]struct{}
}

func newSbpn(self types.Address, peers *peerSet, connect Connector, consensus Consensus) *sbpn {
	sn := &sbpn{
		self:      self,
		targets:   make(map[types.Address]*target),
		peers:     peers,
		connect:   connect,
		consensus: consensus,
		dialing:   make(map[peerId]struct{}),
	}
	consensus.SubscribeProducers(types.SNAPSHOT_GID, "sbpn", sn.receiveProducers)

	return sn
}

func (f *sbpn) isSbp(addr types.Address) bool {
	f.rw.RLock()
	defer f.rw.RUnlock()

	_, ok := f.targets[addr]
	return ok
}

func (f *sbpn) clean() {
	f.consensus.UnSubscribe(types.SNAPSHOT_GID, "sbpn")

	f.rw.Lock()
	defer f.rw.Unlock()
	f.targets = make(map[types.Address]*target)
}

func (f *sbpn) receiveProducers(event consensus.ProducersEvent) {
	f.rw.Lock()
	defer f.rw.Unlock()

	for _, addr := range event.Addrs {
		if addr == f.self {
			continue
		}

		if t, ok := f.targets[addr]; ok {
			if p := f.peers.get(t.ID); p != nil {
				_ = p.SetLevel(p2p.Superior)
				continue
			}

			f.dial(t.Node)
		}
	}
}

func (f *sbpn) dial(node *vnode.Node) {
	if _, ok := f.dialing[node.ID]; ok {
		return
	}
	f.dialing[node.ID] = struct{}{}
	go f.doDial(node)
}

func (f *sbpn) doDial(node *vnode.Node) {
	_ = f.connect.ConnectNode(node)

	f.rw.Lock()
	delete(f.dialing, node.ID)
	f.rw.Unlock()
}

func (f *sbpn) receiveNode(node *vnode.Node) {
	addr, ok := parseNodeExt(node)
	if ok {
		t := &target{
			Node:    node,
			address: addr,
		}

		f.rw.Lock()
		f.targets[addr] = t
		f.rw.Unlock()
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
