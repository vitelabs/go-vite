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
	"fmt"
	"testing"
	"time"
)

func TestTicket_Remainder(t *testing.T) {
	const total = 5

	pool := New(5)

	const i = 3
	for j := 0; j < i; j++ {
		pool.Take()
	}

	if pool.Remainder() != total-i {
		t.Errorf("Remainder should be %d, but get %d", total-i, pool.Remainder())
	}

	pool.Return()
	if pool.Remainder() != total-i+1 {
		t.Errorf("Remainder should be %d, but get %d", total-i+1, pool.Remainder())
	}
}

func TestTicket_Take(t *testing.T) {
	const total = 5
	pool := New(total)
	ch := make(chan int, 1)
	go func() {
		for i := 0; i < total; i++ {
			pool.Take()
		}
		pool.Take()
		ch <- 1
	}()

	select {
	case <-ch:
		t.Error("Take should be blocked")
	case <-time.After(3 * time.Second):

	}
}

func TestTicket_Return(t *testing.T) {
	const total = 5
	pool := New(total)
	for i := 0; i < total; i++ {
		pool.Take()
	}
	for i := 0; i < total; i++ {
		pool.Return()
	}
	if pool.Remainder() != total {
		t.Errorf("should be %d resources in pool, but get %d", total, pool.Remainder())
	}
}

func TestTicket_Close(t *testing.T) {
	const total = 5
	pool := New(total)

	if err := pool.Close(); err != nil {
		fmt.Println(err)
		t.Fail()
	}

	if err := pool.Close(); err == nil {
		fmt.Println(err)
		t.Fail()
	}

	pool.Reset()
	for i := 0; i < total-2; i++ {
		pool.Take()
	}

	pool.Close()
	for i := 0; i < total; i++ {
		pool.Return()
	}

	if pool.Remainder() != 2 {
		t.Fail()
	}
}

func TestTicket_Reset(t *testing.T) {
	const total = 5
	pool := New(total)
	for i := 0; i < total; i++ {
		pool.Take()
	}

	if pool.Remainder() != 0 {
		t.Fail()
	}

	pool.Reset()
	if pool.Remainder() != total {
		t.Fail()
	}
}
