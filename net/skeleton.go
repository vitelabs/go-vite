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

	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

const getHashHeightListTimeout = 10 * time.Second

var errTimeout = errors.New("timeout")

type hashHeightPeers struct {
	*HashHeightPoint
	ps map[peerId]*Peer
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

func (t *hashHeightNode) addBranch(list []*HashHeightPoint, sender *Peer) {
	var tree = t
	var subTree *hashHeightNode
	var ok bool
	for _, h := range list {
		subTree, ok = tree.nodes[h.Hash]
		if ok {
			subTree.ps[sender.Id] = sender
		} else {
			subTree = &hashHeightNode{
				hashHeightPeers{
					h,
					map[peerId]*Peer{
						sender.Id: sender,
					},
				},
				make(map[types.Hash]*hashHeightNode),
			}
			tree.nodes[h.Hash] = subTree
		}

		tree = subTree
	}
}

func (t *hashHeightNode) bestBranch() (list []*HashHeightPoint) {
	var tree = t
	var subTree *hashHeightNode
	var weight int

	for {
		if tree == nil || len(tree.nodes) == 0 {
			return
		}

		weight = 0

		for _, n := range tree.nodes {
			if len(n.ps) > weight {
				weight = len(n.ps)
				subTree = n
			}
		}

		list = append(list, subTree.HashHeightPoint)

		tree = subTree
	}
}

type pending struct {
	wg   *sync.WaitGroup
	tree *hashHeightNode
}

func (p *pending) done(msg Msg, sender *Peer, err error) {
	p.wg.Done()
	if err != nil {
		netLog.Warn(fmt.Sprintf("failed to get HashHeight list from %s: %v", sender, err))
	} else {
		var hh = &HashHeightPointList{}
		err = hh.Deserialize(msg.Payload)
		if err != nil {
			return
		}

		p.tree.addBranch(hh.Points, sender)
	}
}

type skeleton struct {
	checking int32

	tree *hashHeightNode

	blackBlocks map[types.Hash]struct{}

	peers *peerSet
	idGen MsgIder

	mu      sync.Mutex
	pending map[MsgId]*Peer

	wg sync.WaitGroup
}

func newSkeleton(peers *peerSet, idGen MsgIder, blackBlocks map[types.Hash]struct{}) *skeleton {
	return &skeleton{
		peers:       peers,
		idGen:       idGen,
		pending:     make(map[MsgId]*Peer),
		blackBlocks: blackBlocks,
	}
}

// construct return a slice of HashHeight, every 100 step
func (sk *skeleton) construct(start []*ledger.HashHeight, end uint64) (list []*HashHeightPoint) {
	if false == atomic.CompareAndSwapInt32(&sk.checking, 0, 1) {
		netLog.Warn(fmt.Sprintf("skeleton is checking"))
		return
	}
	defer atomic.StoreInt32(&sk.checking, 0)

	sk.tree = newHashHeightTree()

	ps := sk.peers.pickReliable(end)
	if len(ps) > 0 {
		msg := &GetHashHeightList{
			From: start,
			Step: syncTaskSize,
			To:   end,
		}

		// max 10 peers
		if len(ps) > 10 {
			ps = ps[:10]
		}

		for _, p := range ps {
			sk.wg.Add(1)
			sk.getHashList(p, msg)
		}
	}

	sk.wg.Wait()

	sk.mu.Lock()
	list = sk.tree.bestBranch()
	sk.mu.Unlock()

	return
}

func (sk *skeleton) getHashList(p *Peer, msg *GetHashHeightList) {
	mid := sk.idGen.MsgID()
	err := p.send(CodeGetHashList, mid, msg)
	if err != nil {
		sk.wg.Done()
		p.catch(err)
	} else {
		// add pending
		sk.mu.Lock()
		sk.pending[mid] = p
		sk.mu.Unlock()

		time.AfterFunc(getHashHeightListTimeout, func() {
			sk.getHashListFailed(mid, p, errTimeout)
		})
	}
}

func (sk *skeleton) receiveHashList(msg Msg, sender *Peer) {
	if atomic.LoadInt32(&sk.checking) == 1 {
		var hh = &HashHeightPointList{}
		err := hh.Deserialize(msg.Payload)
		if err != nil {
			sk.getHashListFailed(msg.Id, sender, err)
			return
		}

		sk.removePending(msg.Id)

		// is in black list
		if len(sk.blackBlocks) > 0 {
			for _, p := range hh.Points {
				if _, ok := sk.blackBlocks[p.Hash]; ok {
					sender.setReliable(false)
					return
				}
			}
		}

		sk.mu.Lock()
		sk.tree.addBranch(hh.Points, sender)
		sk.mu.Unlock()
	}
}

func (sk *skeleton) getHashListFailed(id MsgId, sender *Peer, err error) {
	sk.removePending(id)
	netLog.Warn(fmt.Sprintf("failed to get HashHeight list from %s: %v", sender, err))
}

func (sk *skeleton) removePending(id MsgId) {
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
	for id := range sk.pending {
		delete(sk.pending, id)
		sk.wg.Done()
	}
	sk.pending = make(map[MsgId]*Peer)
	sk.tree = newHashHeightTree()
	sk.mu.Unlock()
}
