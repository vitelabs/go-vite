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

package bloom

import (
	"crypto/rand"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	_ "net/http/pprof"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestFilter_record(t *testing.T) {
	count := mrand.Intn(100000)
	f := New(uint(count), 0.01)
	m := make(map[types.Hash]struct{}, count)

	for i := 0; i < count; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}

		if _, ok := m[hash]; ok {
			continue
		}

		m[hash] = struct{}{}

		f.Add(hash[:])
	}

	failed := 0
	for hash := range m {
		if !f.Test(hash[:]) {
			failed++
		}
	}

	fmt.Printf("failed %d, total: %d\n", failed, count)
}

func TestFilter_LookAndRecord(t *testing.T) {
	count := mrand.Intn(100000)
	f := New(uint(count), 0.01)
	count *= 10

	m := make(map[types.Hash]struct{}, count)

	var failed int
	for i := 0; i < count; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}
		if _, ok := m[hash]; ok {
			continue
		}
		m[hash] = struct{}{}

		exist := f.TestAndAdd(hash[:])
		if exist {
			failed++
		}

		exist = f.TestAndAdd(hash[:])
		if !exist {
			failed++
		}
	}

	t.Logf("failed %d, count: %d", failed, count)
}

func TestBlockFilter(t *testing.T) {
	var m = make(map[int]*Filter)

	go func() {
		http.ListenAndServe("localhost:8081", nil)
	}()

	for i := 0; i < 100; i++ {
		m[i] = New(100000, 0.0001)
	}

	ch := make(chan struct{})
	ch <- struct{}{}
}
