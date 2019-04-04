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
 * You should have received chain copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package discovery

import (
	"bytes"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/p2p/vnode"
	"github.com/vitelabs/go-vite/tools/list"
)

var errStopped = errors.New("discovery server has stopped")
var errWaitOvertime = errors.New("wait for response timeout")

// request expect a response, eg. ping, findNode
type request struct {
	expectFrom string
	expectID   vnode.NodeID
	expectCode code
	handler    interface {
		// handle return true, then this wait will be removed from pool
		handle(pkt *packet, err error) bool
	}
	expiration time.Time // response timeout
}

// requestPool is the interface can hold pending request
type requestPool interface {
	start()
	stop()
	// add return true mean operation success, return false if pool is not running
	add(req *request) bool
	// rec return true mean the response is expected, else return false
	rec(pkt *packet) bool
	// size is the count of pending request
	size() int
}

type requestPoolImpl struct {
	// every address hash a request list
	pending map[string]list.List
	mu      sync.Mutex

	running int32
	term    chan struct{}
	wg      sync.WaitGroup
}

func newRequestPool() requestPool {
	return &requestPoolImpl{
		pending: make(map[string]list.List),
	}
}

func (p *requestPoolImpl) size() (n int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, l := range p.pending {
		n += l.Size()
	}

	return
}

func (p *requestPoolImpl) start() {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})
		p.pending = make(map[string]list.List)

		p.wg.Add(1)
		go p.loop()
	}
}

func (p *requestPoolImpl) stop() {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		close(p.term)
		p.wg.Wait()
	}
}

func (p *requestPoolImpl) add(req *request) bool {
	if atomic.LoadInt32(&p.running) == 0 {
		return false
	}

	p.mu.Lock()
	l, ok := p.pending[req.expectFrom]
	if !ok {
		l = list.New()
		p.pending[req.expectFrom] = l
	}
	l.Append(req)
	p.mu.Unlock()

	return true
}

func (p *requestPoolImpl) rec(pkt *packet) bool {
	want := false

	p.mu.Lock()
	defer p.mu.Unlock()

	addr := pkt.from.String()
	if l, ok := p.pending[addr]; ok {
		l.Filter(func(v interface{}) bool {
			wt := v.(*request)
			if wt.expectCode == pkt.c {
				want = true
				return wt.handler.handle(pkt, nil)
			}
			return false
		})
	}

	return want
}

func (p *requestPoolImpl) loop() {
	defer p.wg.Done()

	checkTicker := time.NewTicker(expiration / 2)
	defer checkTicker.Stop()

	var now time.Time
Loop:
	for {
		select {
		case <-p.term:
			break Loop
		case now = <-checkTicker.C:
			p.clean(now)
		}
	}

	p.release()

	return
}

func (p *requestPoolImpl) clean(now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, l := range p.pending {
		l.Filter(func(value interface{}) bool {
			wt := value.(*request)
			if wt.expiration.Before(now) {
				wt.handler.handle(nil, errWaitOvertime)
				return true
			}
			return false
		})

		// delete nil list
		if l.Size() == 0 {
			delete(p.pending, addr)
		}
	}
}

func (p *requestPoolImpl) release() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, l := range p.pending {
		l.Filter(func(value interface{}) bool {
			wt := value.(*request)
			wt.handler.handle(nil, errStopped)
			return true
		})
	}

	p.pending = nil
}

type pingRequest struct {
	hash []byte
	done chan<- *Node
}

func (p *pingRequest) handle(pkt *packet, err error) bool {
	if err != nil {
		go pingnil(p.done)
		return true
	}

	bd := pkt.body
	if png, ok := bd.(*pong); ok {
		if bytes.Equal(png.echo, p.hash) {
			go p.receivePong(pkt, png)

			return true
		}
	}

	return false
}

func (p *pingRequest) receivePong(pkt *packet, png *pong) {
	if p.done != nil {
		e, addr := extractEndPoint(pkt.from, png.from)
		p.done <- &Node{
			Node: vnode.Node{
				ID:       pkt.id,
				EndPoint: *e,
				Net:      png.net,
				Ext:      png.ext,
			},
			addr: addr,
		}
	}
}

type findNodeRequest struct {
	count uint32
	rec   []*vnode.EndPoint
	ch    chan<- []*vnode.EndPoint
}

func (f *findNodeRequest) handle(pkt *packet, err error) bool {
	if err != nil {
		go f.done()
		return true
	}

	bd := pkt.body
	if n, ok := bd.(*neighbors); ok {
		f.rec = append(f.rec, n.endpoints...)

		// the last packet maybe received first
		if len(f.rec) >= int(f.count) || n.last {
			go f.done()
			return true
		}
	}

	return false
}

func (f *findNodeRequest) done() {
	f.ch <- f.rec
}
