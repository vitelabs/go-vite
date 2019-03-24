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

package discovery

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/p2p/list"
)

var errStopped = errors.New("discovery server has stopped")
var errWaitOvertime = errors.New("wait for response timeout")

type waitHandler interface {
	handle(msg Message, err error) bool
}

type wait struct {
	expectFrom NodeID
	expectCode packetCode
	sourceHash types.Hash
	handler    waitHandler
	expiration time.Time
}

type packet struct {
	fromID NodeID
	from   *net.UDPAddr
	code   packetCode
	hash   types.Hash
	msg    Message
}

type pool interface {
	start()
	stop()
	add(wt *wait) bool
	rec(res *packet) bool
	size() int
}

type wtPool struct {
	list    list.List
	mu      sync.Mutex
	running int32
	term    chan struct{}
	wg      sync.WaitGroup
}

func (p *wtPool) start() {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})

		p.wg.Add(1)
		go p.loop()
	}
}

func (p *wtPool) stop() {
	atomic.StoreInt32(&p.running, 0)
	if p.term == nil {
		return
	}

	select {
	case <-p.term:
	default:
		close(p.term)
		p.wg.Wait()
	}
}

func newWtPool() *wtPool {
	return &wtPool{
		list: list.New(),
	}
}

func (p *wtPool) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.list.Size()
}

func (p *wtPool) add(wt *wait) bool {
	if atomic.LoadInt32(&p.running) == 0 {
		return false
	}

	p.mu.Lock()
	p.list.Append(wt)
	p.mu.Unlock()

	return true
}

func (p *wtPool) rec(res *packet) bool {
	want := false

	p.mu.Lock()
	defer p.mu.Unlock()

	p.list.Filter(func(v interface{}) bool {
		wt := v.(*wait)
		if wt.expectFrom == res.fromID && wt.expectCode == res.code {
			want = true
			return wt.handler.handle(res.msg, nil)
		}
		return false
	})

	return want
}

func (p *wtPool) loop() {
	defer p.wg.Done()

	checkTicker := time.NewTicker(expiration)
	defer checkTicker.Stop()

Loop:
	for {
		select {
		case <-p.term:
			break Loop
		case now := <-checkTicker.C:
			p.clean(now)
		}
	}

	p.mu.Lock()
	p.list.Traverse(func(value interface{}) bool {
		wt := value.(*wait)
		wt.handler.handle(nil, errStopped)
		return true
	})
	p.mu.Unlock()

	return
}

func (p *wtPool) clean(now time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.list.Filter(func(value interface{}) bool {
		wt := value.(*wait)
		if wt.expiration.Before(now) {
			wt.handler.handle(nil, errWaitOvertime)
			return true
		}
		return false
	})
}

type pingWait struct {
	sourceHash types.Hash
	done       chan<- bool
}

func (p *pingWait) handle(msg Message, err error) bool {
	if err != nil {
		if p.done != nil {
			p.done <- false
		}
		return true
	}

	if msg, ok := msg.(*Pong); ok {
		if msg.Ping == p.sourceHash {
			if p.done != nil {
				p.done <- true
			}
			return true
		}
	}

	return false
}

type findnodeWait struct {
	need uint32
	rec  []*Node
	ch   chan<- []*Node
}

func (f *findnodeWait) handle(msg Message, err error) bool {
	if err != nil {
		if f.ch != nil {
			f.ch <- f.rec
		}
		return true
	}

	if msg, ok := msg.(*Neighbors); ok {
		f.rec = append(f.rec, msg.Nodes...)

		if len(msg.Nodes) < maxNeighborsOneTrip || len(f.rec) >= int(f.need) {
			if f.ch != nil {
				f.ch <- f.rec
			}
			return true
		}
	}

	return false
}
