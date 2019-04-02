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

package discovery

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/crypto/ed25519"
)

var errIncompleteMessage = errors.New("incomplete message")
var errSocketIsRunning = errors.New("udp socket is listening")
var errSocketIsNotRunning = errors.New("udp socket is not running")

const socketQueueLength = 10

type sender interface {
	ping(n *Node, ch chan<- *Node) (err error)
	pong(echo []byte, n *Node) (err error)
	findNode(target vnode.NodeID, count uint32, n *Node, ch chan<- []vnode.EndPoint) (err error)
	sendNodes(eps []vnode.EndPoint, to *net.UDPAddr) (err error)
}

type receiver interface {
	start() error
	stop() error
}

type socket interface {
	sender
	receiver
}

type agent struct {
	self    *vnode.Node
	socket  *net.UDPConn
	peerKey ed25519.PrivateKey
	queue   chan *packet
	handler func(*packet)
	pool    pool
	running int32
	term    chan struct{}
	wg      sync.WaitGroup
}

func newAgent(peerKey ed25519.PrivateKey, self *vnode.Node, handler func(*packet)) *agent {
	return &agent{
		self:    self,
		peerKey: peerKey,
		handler: handler,
		pool:    &wtPool{},
	}
}

func (a *agent) start() (err error) {
	if atomic.CompareAndSwapInt32(&a.running, 0, 1) {
		var udp *net.UDPAddr
		udp, err = net.ResolveUDPAddr("udp", a.self.Address())
		if err != nil {
			return
		}

		a.socket, err = net.ListenUDP("udp", udp)
		if err != nil {
			return
		}

		a.term = make(chan struct{})
		a.queue = make(chan *packet, socketQueueLength)

		a.pool.start()

		a.wg.Add(1)
		go a.readLoop()

		a.wg.Add(1)
		go a.handleLoop()

		return nil
	}

	return errSocketIsRunning
}

func (a *agent) stop() (err error) {
	if atomic.CompareAndSwapInt32(&a.running, 1, 0) {
		err = a.socket.Close()
		close(a.term)
		a.wg.Wait()
	}

	return errSocketIsNotRunning
}

func (a *agent) ping(n *Node, ch chan<- *Node) (err error) {
	udp, err := n.udpAddr()
	if err != nil {
		if ch != nil {
			ch <- nil
		}

		return
	}

	now := time.Now()
	hash, err := a.write(message{
		c:  codePing,
		id: a.self.ID,
		body: &ping{
			from: &a.self.EndPoint,
			to:   &n.EndPoint,
			net:  a.self.Net,
			ext:  a.self.Ext,
			time: now,
		},
	}, udp)

	if err != nil {
		if ch != nil {
			ch <- nil
		}
		return
	}

	a.pool.add(&wait{
		expectAddr: udp.String(),
		expectCode: codePong,
		handler: &pingWait{
			hash: hash,
			done: ch,
		},
		expiration: now.Add(expiration),
	})

	return
}

func (a *agent) pong(echo []byte, n *Node) (err error) {
	udp, err := n.udpAddr()
	if err != nil {
		return
	}

	_, err = a.write(message{
		c:  codePong,
		id: a.self.ID,
		body: &pong{
			from: &a.self.EndPoint,
			to:   &n.EndPoint,
			net:  a.self.Net,
			ext:  a.self.Ext,
			echo: echo,
			time: time.Now(),
		},
	}, udp)

	return
}

func (a *agent) findNode(target vnode.NodeID, count uint32, n *Node, ch chan<- []vnode.EndPoint) (err error) {
	udp, err := n.udpAddr()
	if err != nil {
		ch <- nil
		return
	}

	_, err = a.write(message{
		c:  codeFindnode,
		id: a.self.ID,
		body: &findnode{
			target: target,
			count:  count,
			time:   time.Now(),
		},
	}, udp)

	if err != nil {
		ch <- nil
		return
	}

	a.pool.add(&wait{
		expectID:   n.ID,
		expectAddr: udp.String(),
		expectCode: codeNeighbors,
		handler: &findnodeWait{
			count: count,
			rec:   nil,
			ch:    ch,
		},
		expiration: time.Now().Add(expiration),
	})

	return
}

func (a *agent) sendNodes(eps []vnode.EndPoint, to *net.UDPAddr) (err error) {
	var msg = message{
		c:  codeNeighbors,
		id: a.self.ID,
	}

	n := &neighbors{
		endpoints: nil,
		last:      false,
		time:      time.Now(),
	}
	msg.body = n

	var mark, bytes int
	for i, ep := range eps {
		bytes += ep.Length()

		if bytes+20 > maxPayloadLength {
			mark = i
			n.endpoints = append(eps[mark:i])
			bytes = 0

			_, err = a.write(msg, to)
			if err != nil {
				return
			}
		}
	}

	n.endpoints = eps[mark:]
	_, err = a.write(msg, to)

	return
}

func (a *agent) readLoop() {
	defer a.wg.Done()
	defer close(a.queue)

	buf := make([]byte, maxPacketLength)

	var tempDelay time.Duration
	var maxDelay = time.Second

	for {
		n, addr, err := a.socket.ReadFromUDP(buf)

		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}

				if tempDelay > maxDelay {
					tempDelay = maxDelay
				}

				time.Sleep(tempDelay)

				continue
			}
			return
		}

		tempDelay = 0

		if n == 0 {
			continue
		}

		p, err := unPacket(buf[:n])
		if err != nil {
			continue
		}

		if p.expired() {
			continue
		}

		p.from = addr

		want := true
		if p.c == codePong || p.c == codeNeighbors {
			want = a.pool.rec(p)
		}

		if want {
			a.handler(p)
		}
	}
}

func (a *agent) handleLoop() {
	defer a.wg.Done()

	var pkt *packet
	for {
		select {
		case <-a.term:
			return
		case pkt = <-a.queue:
			a.handler(pkt)
		}
	}
}

func (a *agent) write(msg message, addr *net.UDPAddr) (hash []byte, err error) {
	data, hash, err := msg.pack(a.peerKey)

	if err != nil {
		return
	}

	n, err := a.socket.WriteToUDP(data, addr)

	if err != nil {
		return
	}

	if n != len(data) {
		err = errIncompleteMessage
		return
	}

	return
}
