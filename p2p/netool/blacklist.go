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

package netool

import (
	"encoding/hex"
	"time"
)

// Strategy receive first ban time and ban count, return true means banned
type Strategy func(t time.Time, count int) bool

type BlackList interface {
	Ban([]byte)
	UnBan([]byte)
	Banned([]byte) bool
}

type record struct {
	t time.Time
	c int
}

type blackList struct {
	records  map[string]*record
	strategy Strategy
}

func NewBlackList(strategy Strategy) BlackList {
	return &blackList{
		records:  make(map[string]*record),
		strategy: strategy,
	}
}

func (b *blackList) Ban(buf []byte) {
	id := hex.EncodeToString(buf)
	if r, ok := b.records[id]; ok {
		r.t = time.Now()
		r.c++
	} else {
		b.records[id] = &record{
			t: time.Now(),
			c: 1,
		}
	}
}

func (b *blackList) UnBan(buf []byte) {
	id := hex.EncodeToString(buf)
	delete(b.records, id)
}

func (b *blackList) Banned(buf []byte) bool {
	id := hex.EncodeToString(buf)

	if r, ok := b.records[id]; ok {
		if b.strategy(r.t, r.c) {
			return true
		} else {
			delete(b.records, id)
		}
	}

	return false
}
