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
	"encoding/binary"
	"hash"
	"math"
)

const fillRatio = 0.5

var fillRatioX float64 = 0

// OptimalM calculates the optimal Bloom filter size, m, based on the number of
// items and the desired rate of false positives.
func OptimalM(n uint, fpRate float64) uint {
	if fillRatioX == 0 {
		fillRatioX = math.Log(fillRatio) * math.Log(1-fillRatio)
	}

	base := math.Abs(math.Log(fpRate)) / fillRatioX
	r := float64(n) * base
	return uint(math.Ceil(r))
}

// OptimalK calculates the optimal number of hash functions to use for a Bloom
// filter based on the desired rate of false positives.
func OptimalK(fpRate float64) uint {
	return uint(math.Ceil(math.Log2(1 / fpRate)))
}

// hashInternal returns the upper and lower base hash values from which the k
// hashes are derived.
func hashInternal(data []byte, hash hash.Hash64) (uint32, uint32) {
	hash.Write(data)
	sum := hash.Sum(nil)
	hash.Reset()
	return binary.BigEndian.Uint32(sum[4:8]), binary.BigEndian.Uint32(sum[0:4])
}
