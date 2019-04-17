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

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/crypto/ed25519"
)

var errIncompleteMessage = errors.New("incomplete message")
var errSocketIsRunning = errors.New("udp socket is listening")
var errSocketIsNotRunning = errors.New("udp socket is not running")

const socketQueueLength = 10

// sender return err is not nil if one of the following scene occur:
// 1. failed to resolve net.UDPAddr of n
// 2. send message error
type sender interface {
	// ping n, extract node from the pong response and put the node, nil if has error, into ch, ch MUST not be nil.
	ping(n *Node, ch chan<- *Node) (err error)
	// pong respond the last ping message from n, echo is the hash of ping message payload
	pong(echo []byte, n *Node) (err error)
	// findNode find count nodes near the target to n, put the responsive nodes into ch, ch MUST no be nil
	findNode(target vnode.NodeID, count int, n *Node, ch chan<- []*vnode.EndPoint) (err error)
	// sendNodes to addr, if eps is too many, the response message will be split to multiple message,
	// every message is small than maxPacketLength.
	sendNodes(eps []*vnode.EndPoint, addr *net.UDPAddr) (err error)
}

type receiver interface {
	start() error
	stop() error
}

type socket interface {
	sender
	receiver
}

// packet is a parsed message received from socket
type packet struct {
	message
	from *net.UDPAddr
	hash []byte
}

type agent struct {
	node          *vnode.Node
	listenAddress string
	socket        *net.UDPConn
	peerKey       ed25519.PrivateKey
	queue         chan *packet
	handler       func(*packet)
	pool          requestPool
	running       int32
	term          chan struct{}
	wg            sync.WaitGroup
	log           log15.Logger
}

func newAgent(peerKey ed25519.PrivateKey, self *vnode.Node, listenAddress string, handler func(*packet)) *agent {
	return &agent{
		node:          self,
		listenAddress: listenAddress,
		peerKey:       peerKey,
		handler:       handler,
		pool:          newRequestPool(),
		log:           discvLog.New("module", "socket"),
	}
}

func (a *agent) start() (err error) {
	if atomic.CompareAndSwapInt32(&a.running, 0, 1) {
		var udp *net.UDPAddr
		udp, err = net.ResolveUDPAddr("udp", a.listenAddress)
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

func pingnil(ch chan<- *Node) {
	ch <- nil
}

func (a *agent) ping(n *Node, ch chan<- *Node) (err error) {
	udp, err := n.udpAddr()
	if err != nil {
		go pingnil(ch)
		return
	}

	now := time.Now()
	hash, err := a.write(message{
		c:  codePing,
		id: a.node.ID,
		body: &ping{
			from: &a.node.EndPoint,
			to:   &n.EndPoint,
			net:  a.node.Net,
			ext:  a.node.Ext,
			time: now,
		},
	}, udp)

	if err != nil {
		go pingnil(ch)
		return
	}

	a.pool.add(&request{
		expectFrom: udp.String(),
		expectID:   n.ID,
		expectCode: codePong,
		handler: &pingRequest{
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
		id: a.node.ID,
		body: &pong{
			from: &a.node.EndPoint,
			to:   &n.EndPoint,
			net:  a.node.Net,
			ext:  a.node.Ext,
			echo: echo,
			time: time.Now(),
		},
	}, udp)

	return
}

func findnodenil(ch chan<- []*vnode.EndPoint) {
	ch <- nil
}
func (a *agent) findNode(target vnode.NodeID, count int, n *Node, ch chan<- []*vnode.EndPoint) (err error) {
	udp, err := n.udpAddr()
	if err != nil {
		go findnodenil(ch)
		return
	}

	_, err = a.write(message{
		c:  codeFindnode,
		id: a.node.ID,
		body: &findnode{
			target: target,
			count:  count,
			time:   time.Now(),
		},
	}, udp)

	if err != nil {
		go findnodenil(ch)
		return
	}

	a.pool.add(&request{
		expectID:   n.ID,
		expectFrom: udp.String(),
		expectCode: codeNeighbors,
		handler: &findNodeRequest{
			count: count,
			ch:    ch,
		},
		expiration: time.Now().Add(expiration),
	})

	return
}

func (a *agent) sendNodes(eps []*vnode.EndPoint, to *net.UDPAddr) (err error) {
	n := &neighbors{
		endpoints: nil,
		last:      false,
		time:      time.Now(),
	}

	msg := message{
		c:    codeNeighbors,
		id:   a.node.ID,
		body: n,
	}

	ept := splitEndPoints(eps)
	for i, epl := range ept {
		n.endpoints = epl
		n.last = i == len(ept)-1

		_, err = a.write(msg, to)
		if err != nil {
			return
		}
	}

	return
}

func splitEndPoints(eps []*vnode.EndPoint) (ept [][]*vnode.EndPoint) {
	var sent, bytes int
	for i, ep := range eps {
		bytes += ep.Length()

		if bytes+300 > maxPayloadLength {
			ept = append(ept, eps[sent:i])
			bytes = 0
			sent = i
		}
	}

	if sent != len(eps) {
		ept = append(ept, eps[sent:])
	}

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

		p, err := unPacket(buf[:n])
		if err != nil {
			continue
		}
		//fmt.Printf("%s receive packet %d from %s\n", a.self.Address(), p.c, addr.String())
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

	for pkt := range a.queue {
		a.handler(pkt)
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
