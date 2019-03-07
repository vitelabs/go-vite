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

package network

import (
	"encoding/hex"
	"time"
)

type Record struct {
	T time.Time
	C int
}

type Policy func(t time.Time, count int) bool

type Block struct {
	records map[string]*Record
	policy  Policy
}

func New(policy Policy) *Block {
	return &Block{
		records: make(map[string]*Record),
		policy:  policy,
	}
}

func (b *Block) Block(buf []byte) {
	id := hex.EncodeToString(buf)
	if r, ok := b.records[id]; ok {
		r.T = time.Now()
		r.C++
	} else {
		b.records[id] = &Record{
			T: time.Now(),
			C: 1,
		}
	}
}

func (b *Block) UnBlock(buf []byte) {
	id := hex.EncodeToString(buf)
	delete(b.records, id)
}

func (b *Block) Blocked(buf []byte) bool {
	id := hex.EncodeToString(buf)

	if r, ok := b.records[id]; ok {
		return b.policy(r.T, r.C)
	}

	return false
}
