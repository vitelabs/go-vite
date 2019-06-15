/*
 * Copyright 2018 The go-vite Authors
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
 * You should have received chain copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/p2p/discovery"

	"github.com/vitelabs/go-vite/consensus"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite/net/sbpn"
)

type mockInformer struct {
	mk   *mocker
	sbps []types.Address
	cb   func(event consensus.ProducersEvent)
	term chan struct{}
}

func (mi *mockInformer) SubscribeProducers(gid types.Gid, id string, fn func(event consensus.ProducersEvent)) {
	mi.cb = fn
}

func (mi *mockInformer) UnSubscribe(gid types.Gid, id string) {
	mi.cb = nil
}

func (mi *mockInformer) start() {
	mi.term = make(chan struct{})

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

	Loop:
		for {
			select {
			case <-mi.term:
				break Loop
			case <-ticker.C:
				mi.sbps = mi.mockAddresses(5)
				mi.cb(consensus.ProducersEvent{
					Addrs: mi.sbps,
				})
			}
		}
	}()
}

func (mi *mockInformer) mockAddresses(n int) []types.Address {
	ret := make([]types.Address, n)
	for i := 0; i < n; i++ {
		ret[i] = mi.mk.randAddr()
	}
	return ret
}

func (mi *mockInformer) stop() {
	select {
	case <-mi.term:
	default:
		close(mi.term)
	}
}

type mockListener struct {
	self types.Address
	pool sync.Map
	bps  []types.Address
}

func (ml *mockListener) GotNodeCallback(node *discovery.Node) {
	var addr types.Address
	copy(addr[:], node.Ext)
	ml.pool.Store(addr, node.ID)
	fmt.Println("node", addr.String(), node.ID.String())
}

func (ml *mockListener) GotSBPSCallback(addrs []types.Address) {
	for _, addr := range addrs {
		fmt.Println("addr", addr)
	}
	ml.bps = addrs
}

func (ml *mockListener) ConnectCallback(addr types.Address, id discovery.NodeID) {
	v, ok := ml.pool.Load(addr)
	if ok && v.(discovery.NodeID) == id {
		fmt.Println("ok", id, addr)
	} else {
		fmt.Println("error", id, addr)
	}
}

func main() {
	var selfAddr types.Address
	rand.Read(selfAddr[:])

	mk := &mocker{
		self: selfAddr,
	}

	mi := &mockInformer{
		mk: mk,
	}
	ml := &mockListener{
		self: selfAddr,
	}
	ms := &mockServer{
		mk: mk,
	}

	finder := sbpn.New(selfAddr, mi)
	finder.SetListener(ml)

	finder.Start(ms)
	mi.start()
	ms.Start()

	time.Sleep(5 * time.Minute)
}
