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

package ticket

import (
	"errors"
	"sync/atomic"
)

var errClosed = errors.New("ticket has already closed")

// Ticket represent limited resources to control concurrency
type Ticket interface {
	// Take a ticket, if no tickets remain, then will be blocked.
	// if Ticket is closed, will return non-block.
	Take()
	// Return a ticket, if Ticket is closed, will return
	Return()
	// Remainder get the number of available tickets
	Remainder() int
	// Total is the total tickets
	Total() int
	// Close Ticket, Take() will get endless non-blocking tickets
	// return error, if Ticket has been closed
	Close() error
	// Reset will create new tickets
	Reset()
}

type ticket struct {
	total   int
	ch      chan struct{}
	_closed int32
}

// Take retrieve a resource from pool, if pool is empty,
// then current goroutine will be blocked
func (t *ticket) Take() {
	if t.closed() {
		return
	}

	<-t.ch
}

// Return a resource to pool, if there are some goroutine is blocked since resource shortage,
// then a goroutine will be notified
func (t *ticket) Return() {
	if t.closed() {
		return
	}

	t.ch <- struct{}{}
}

// Remainder return rest resources in pool
func (t *ticket) Remainder() int {
	return len(t.ch)
}

// Total represent count of all resources in pool
func (t *ticket) Total() int {
	return t.total
}

func (t *ticket) Close() error {
	if atomic.CompareAndSwapInt32(&t._closed, 0, 1) {
		close(t.ch)
		return nil
	}
	return errClosed
}

func (t *ticket) closed() bool {
	return atomic.LoadInt32(&t._closed) == 1
}

func (t *ticket) Reset() {
	t.ch = make(chan struct{}, t.total)
	for i := 0; i < t.total; i++ {
		t.ch <- struct{}{}
	}
	atomic.StoreInt32(&t._closed, 0)
}

// New will create a resource pool, total means count of resource in pool
func New(total int) Ticket {
	ch := make(chan struct{}, total)
	for i := 0; i < total; i++ {
		ch <- struct{}{}
	}

	return &ticket{
		total: total,
		ch:    ch,
	}
}
