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
	"hash"
	"hash/fnv"
	"sync"
)

// Filter implements a classic thread-safe Bloom filter. A Bloom filter has a non-zero
// probability of false positives and a zero probability of false negatives.
type Filter struct {
	buckets [2]*Buckets // filter data
	hash    hash.Hash64 // hash function (kernel for all k functions)
	m       uint        // filter size
	k       uint        // number of hash functions
	rw      sync.RWMutex
}

// New creates a new Bloom filter optimized to store n items with a
// specified target false-positive rate.
func New(n uint, fpRate float64) *Filter {
	m := OptimalM(n, fpRate)
	return &Filter{
		buckets: [2]*Buckets{
			NewBuckets(m, 1),
			NewBuckets(m, 1),
		},
		hash: fnv.New64(),
		m:    m,
		k:    OptimalK(fpRate),
	}
}

func (b *Filter) Test(data []byte) (exist bool) {
	b.rw.Lock()
	lower, upper := hashInternal(data, b.hash)
	exist = b.testHashUnlocked(lower, upper)
	b.rw.Unlock()
	return
}

//func (b *Filter) TestHash(lower, upper uint32) (exist bool) {
//	b.rw.RLock()
//	exist = b.testHashUnlocked(lower, upper)
//	b.rw.RUnlock()
//
//	return
//}

func (b *Filter) testHashUnlocked(lower, upper uint32) (exist bool) {
	// If any of the K bits are not set, then it's not a member.
	for _, bkt := range b.buckets {
		exist = true
		for i := uint(0); i < b.k; i++ {
			if bkt.Get((uint(lower)+uint(upper)*i)%b.m) == 0 {
				exist = false
				break
			}
		}

		if exist {
			return true
		}
	}

	exist = false
	return
}

// Add will add the data to the Bloom filter. It returns the filter to allow
// for chaining.
func (b *Filter) Add(data []byte) {
	b.rw.Lock()
	lower, upper := hashInternal(data, b.hash)
	b.addHashUnlocked(lower, upper)
	b.rw.Unlock()
	return
}

//func (b *Filter) AddHash(lower, upper uint32) {
//	b.rw.Lock()
//	b.addHashUnlocked(lower, upper)
//	b.rw.Unlock()
//	return
//}

func (b *Filter) addHashUnlocked(lower, upper uint32) {
	if b.buckets[0].FullRatio() > 0.8 {
		temp := b.buckets[0]
		b.buckets[0] = b.buckets[1]
		b.buckets[1] = temp

		b.buckets[0].Reset()
	}

	// Set the K bits.
	for i := uint(0); i < b.k; i++ {
		b.buckets[0].Set((uint(lower)+uint(upper)*i)%b.m, 1)
	}
}

// TestAndAdd is equivalent to calling Test followed by Add. It returns true if
// the data is a member, false if not.
func (b *Filter) TestAndAdd(data []byte) bool {
	b.rw.Lock()
	lower, upper := hashInternal(data, b.hash)
	if b.testHashUnlocked(lower, upper) {
		b.rw.Unlock()
		return true
	}
	b.addHashUnlocked(lower, upper)
	b.rw.Unlock()
	return false
}
