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

// Ticket represent limited resources to control concurrency
type Ticket interface {
	Take()
	Return()
	Remainder() int
	Total() int
}

type ticket struct {
	total int
	ch    chan struct{}
}

// Take retrieve a resource from pool, if pool is empty,
// then current goroutine will be blocked
func (t *ticket) Take() {
	<-t.ch
}

// Return a resource to pool, if there are some goroutine is blocked since resource shortage,
// then a goroutine will be notified
func (t *ticket) Return() {
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
