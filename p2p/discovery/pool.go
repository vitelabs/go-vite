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
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package discovery

//import (
//	"sync"
//	"time"
//
//	"github.com/vitelabs/go-vite/common"
//	"github.com/vitelabs/go-vite/common/types"
//	"github.com/vitelabs/go-vite/p2p/list"
//)
//
//// after send query. wating for reply.
//type waitIsDone func(Message, error, *wait) bool
//
//type wait struct {
//	expectFrom NodeID
//	expectCode packetCode
//	sourceHash types.Hash
//	handle     waitIsDone
//	expiration time.Time
//}
//
//type wtPool struct {
//	list *list.List
//	add  chan *wait
//	rec  chan *packet
//	term chan struct{}
//	wg   sync.WaitGroup
//}
//
//func newWtPool() *wtPool {
//	return &wtPool{
//		list: list.New(),
//		add:  make(chan *wait),
//		rec:  make(chan *packet),
//	}
//}
//
//func (p *wtPool) start() {
//	p.term = make(chan struct{})
//
//	p.wg.Add(1)
//	common.Go(p.loop)
//}
//
//func (p *wtPool) stop() {
//	if p.term == nil {
//		return
//	}
//
//	select {
//	case <-p.term:
//	default:
//		close(p.term)
//		p.wg.Wait()
//	}
//}
//
//func (p *wtPool) loop() {
//	defer p.wg.Done()
//
//	checkTicker := time.NewTicker(expiration)
//	defer checkTicker.Stop()
//
//	for {
//		select {
//		case <-p.term:
//			p.list.Traverse(func(_, e *list.Element) {
//				wt, _ := e.Value.(*wait)
//				wt.handle(nil, errStopped, wt)
//			})
//			return
//		case w := <-p.add:
//			p.list.Append(w)
//		case r := <-p.rec:
//			p.handle(r)
//		case <-checkTicker.C:
//			p.clean()
//		}
//	}
//}
//
//func (p *wtPool) handle(rs *packet) {
//	p.list.Traverse(func(prev, current *list.Element) {
//		wt, _ := current.Value.(*wait)
//		if wt.expectFrom == rs.fromID && wt.expectCode == rs.code {
//			if wt.handle(rs.msg, nil, wt) {
//				// remove current wait from list
//				p.list.Remove(prev, current)
//			}
//		}
//	})
//}
//
//func (p *wtPool) clean() {
//	now := time.Now()
//	p.list.Traverse(func(prev, current *list.Element) {
//		wt, _ := current.Value.(*wait)
//		if wt.expiration.Before(now) {
//			wt.handle(nil, errWaitOvertime, wt)
//			// remove current wait from list
//			p.list.Remove(prev, current)
//		}
//	})
//}
