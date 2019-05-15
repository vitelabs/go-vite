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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

const getHashHeightListTimeout = 10 * time.Second

type hashHeightPeers struct {
	*ledger.HashHeight
	ps map[peerId]Peer
}

type hashHeightNode struct {
	hashHeightPeers
	nodes map[types.Hash]*hashHeightNode
}

func newHashHeightTree() *hashHeightNode {
	return &hashHeightNode{
		nodes: make(map[types.Hash]*hashHeightNode),
	}
}

func (t *hashHeightNode) addBranch(list []*ledger.HashHeight, sender Peer) {
	var tree = t
	var subTree *hashHeightNode
	var ok bool
	for _, h := range list {
		subTree, ok = tree.nodes[h.Hash]
		if ok {
			subTree.ps[sender.ID()] = sender
		} else {
			subTree = &hashHeightNode{
				hashHeightPeers{
					h,
					map[peerId]Peer{
						sender.ID(): sender,
					},
				},
				make(map[types.Hash]*hashHeightNode),
			}
			tree.nodes[h.Hash] = subTree
		}

		tree = subTree
	}
}

func (t *hashHeightNode) bestBranch() (list []*ledger.HashHeight) {
	var tree = t
	var subTree *hashHeightNode
	var weight int

	for {
		if len(tree.nodes) == 0 {
			return
		}

		weight = 0

		for _, n := range tree.nodes {
			if len(n.ps) > weight {
				weight = len(n.ps)
				subTree = n
			}
		}

		list = append(list, subTree.HashHeight)

		tree = subTree
	}
}

type skeleton struct {
	chain syncChain

	batchHead   types.Hash
	batchHeight uint64

	checking int32

	tree *hashHeightNode

	peers syncPeerSet
	idGen MsgIder

	mu      sync.Mutex
	pending map[p2p.MsgId]Peer

	wg sync.WaitGroup
}

func newSkeleton(chain syncChain, peers syncPeerSet, idGen MsgIder) *skeleton {
	return &skeleton{
		chain:   chain,
		peers:   peers,
		idGen:   idGen,
		pending: make(map[p2p.MsgId]Peer),
	}
}

// construct return a slice of HashHeight, every 100 step
func (sk *skeleton) construct(start []*ledger.HashHeight, end uint64) (list []*ledger.HashHeight) {
	atomic.StoreInt32(&sk.checking, 1)

	sk.tree = newHashHeightTree()

	ps := sk.peers.pick(end)
	if len(ps) > 0 {
		msg := &message.GetHashHeightList{
			From: start,
			Step: syncTaskSize,
			To:   end,
		}

		for _, p := range ps {
			sk.wg.Add(1)
			sk.getHashList(p, msg)
		}
	}

	sk.wg.Wait()
	atomic.StoreInt32(&sk.checking, 0)

	sk.mu.Lock()
	list = sk.tree.bestBranch()
	sk.mu.Unlock()

	return
}

func (sk *skeleton) getHashList(p Peer, msg *message.GetHashHeightList) {
	mid := sk.idGen.MsgID()
	err := p.send(p2p.CodeGetHashList, mid, msg)
	if err != nil {
		p.catch(err)
	} else {
		time.AfterFunc(getHashHeightListTimeout, func() {
			sk.removePending(mid)
		})

		// add pending
		sk.mu.Lock()
		sk.pending[mid] = p
		sk.mu.Unlock()
	}
}

func (sk *skeleton) receiveHashList(msg p2p.Msg, sender Peer) {
	if atomic.LoadInt32(&sk.checking) == 1 {
		var hh = &message.HashHeightList{}
		err := hh.Deserialize(msg.Payload)
		if err != nil {
			sk.getHashListFailed(msg.Id, sender, err)
			return
		}

		sk.mu.Lock()
		sk.tree.addBranch(hh.Points, sender)
		sk.mu.Unlock()
	}
}

func (sk *skeleton) getHashListFailed(id p2p.MsgId, sender Peer, err error) {
	sk.removePending(id)
	netLog.Warn(fmt.Sprintf("failed to get HashHeight list from %s: %v", sender, err))
}

func (sk *skeleton) removePending(id p2p.MsgId) {
	sk.mu.Lock()
	if _, ok := sk.pending[id]; ok {
		delete(sk.pending, id)
		sk.mu.Unlock()
		sk.wg.Done()

		// todo handle response error
	} else {
		sk.mu.Unlock()
	}
}

func (sk *skeleton) reset() {
	sk.mu.Lock()
	sk.pending = make(map[p2p.MsgId]Peer)
	sk.tree = nil
	sk.mu.Unlock()
}
