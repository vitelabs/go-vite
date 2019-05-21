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
	mrand "math/rand"
	"net"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/discovery"
)

type mockServer struct {
	mk   *mocker
	rec  chan<- *discovery.Node
	term chan struct{}
}

func (ms *mockServer) Start() error {
	ms.term = make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		ip := make([]byte, 4)
		var port uint16
		var addr types.Address

	Loop:
		for {
			select {
			case <-ms.term:
				break Loop
			case <-ticker.C:
				rand.Read(ip)
				addr = ms.mk.randAddr()
				port = uint16(mrand.Intn(65535))
				ms.rec <- &discovery.Node{
					ID:  ms.mk.randID(),
					IP:  ip,
					UDP: port,
					TCP: port,
					Net: 3,
					Ext: addr[:],
				}
			}
		}
	}()

	return nil
}

func (ms *mockServer) Stop() {
	select {
	case <-ms.term:
	default:
		close(ms.term)
	}
}

func (ms *mockServer) AddPlugin(plugin p2p.Plugin) {
	panic("implement me")
}

func (ms *mockServer) Connect(id discovery.NodeID, addr *net.TCPAddr) {
}

func (ms *mockServer) Peers() []*p2p.PeerInfo {
	panic("implement me")
}

func (ms *mockServer) PeersCount() uint {
	panic("implement me")
}

func (ms *mockServer) NodeInfo() p2p.NodeInfo {
	panic("implement me")
}

func (ms *mockServer) Available() bool {
	panic("implement me")
}

func (ms *mockServer) Nodes() (urls []string) {
	panic("implement me")
}

func (ms *mockServer) SubNodes(ch chan<- *discovery.Node) {
	ms.rec = ch
}

func (ms *mockServer) UnSubNodes(ch chan<- *discovery.Node) {
	ms.rec = nil
}

func (ms *mockServer) URL() string {
	panic("implement me")
}

func (ms *mockServer) Config() *p2p.Config {
	panic("implement me")
}
