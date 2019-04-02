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
	"bytes"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/vnode"
	"github.com/vitelabs/go-vite/tools/list"
)

var errStopped = errors.New("discovery server has stopped")
var errWaitOvertime = errors.New("wait for response timeout")

type waitHandler interface {
	handle(pkt *packet, err error) bool
}

type wait struct {
	expectAddr string
	expectID   vnode.NodeID
	expectCode code
	hash       []byte
	handler    waitHandler
	expiration time.Time
}

type packet struct {
	message
	from *net.UDPAddr
	hash types.Hash
}

type pool interface {
	start()
	stop()
	add(wt *wait) bool
	rec(res *packet) bool
}

type wtPool struct {
	pending map[string]list.List
	mu      sync.Mutex

	running int32
	term    chan struct{}
	wg      sync.WaitGroup
}

func (p *wtPool) start() {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})
		p.pending = make(map[string]list.List)

		p.wg.Add(1)
		go p.loop()
	}
}

func (p *wtPool) stop() {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		close(p.term)
		p.wg.Wait()
	}
}

func (p *wtPool) add(wt *wait) bool {
	if atomic.LoadInt32(&p.running) == 0 {
		return false
	}

	p.mu.Lock()
	l, ok := p.pending[wt.expectAddr]
	if !ok {
		l = list.New()
		p.pending[wt.expectAddr] = l
	}
	l.Append(wt)
	p.mu.Unlock()

	return true
}

func (p *wtPool) rec(res *packet) bool {
	want := false

	p.mu.Lock()
	defer p.mu.Unlock()

	addr := res.from.String()
	if l, ok := p.pending[addr]; ok {
		l.Filter(func(v interface{}) bool {
			wt := v.(*wait)
			if wt.expectCode == res.c {
				want = true
				return wt.handler.handle(res, nil)
			}
			return false
		})
	}

	return want
}

func (p *wtPool) loop() {
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

func (p *wtPool) clean(now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for addr, l := range p.pending {
		l.Filter(func(value interface{}) bool {
			wt := value.(*wait)
			if wt.expiration.Before(now) {
				wt.handler.handle(nil, errWaitOvertime)
				return true
			}
			return false
		})

		if l.Size() == 0 {
			delete(p.pending, addr)
		}
	}
}

func (p *wtPool) release() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, l := range p.pending {
		l.Filter(func(value interface{}) bool {
			wt := value.(*wait)
			wt.handler.handle(nil, errWaitOvertime)
			return true
		})
	}

	p.pending = nil
}

type pingWait struct {
	hash []byte
	done chan<- *Node
}

func (p *pingWait) handle(msg *packet, err error) bool {
	if err != nil {
		if p.done != nil {
			p.done <- nil
		}
		return true
	}

	bd := msg.body
	if png, ok := bd.(*pong); ok {
		if bytes.Equal(png.echo, p.hash) {
			e, addr := extractEndPoint(msg.from, png.from)

			if p.done != nil {
				p.done <- &Node{
					Node: vnode.Node{
						ID:       msg.id,
						EndPoint: *e,
						Net:      png.net,
						Ext:      png.ext,
					},
					checkAt:  time.Now(),
					checking: false,
					finding:  false,
					addr:     addr,
				}
			}

			return true
		}
	}

	return false
}

type findnodeWait struct {
	count uint32
	rec   []vnode.EndPoint
	ch    chan<- []vnode.EndPoint
}

func (f *findnodeWait) handle(msg *packet, err error) bool {
	if err != nil {
		f.ch <- f.rec
		return true
	}

	bd := msg.body
	if n, ok := bd.(*neighbors); ok {
		f.rec = append(f.rec, n.endpoints...)

		if n.last {
			f.ch <- f.rec
			return true
		}
	}

	return false
}
