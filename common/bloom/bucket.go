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

// Buckets is a fast, space-efficient array of buckets where each bucket can
// store up to a configured maximum value.
type Buckets struct {
	data       []byte
	bucketSize uint8
	max        uint8
	count      uint
	total      uint
}

// NewBuckets creates a new Buckets with the provided number of buckets where
// each bucket is the specified number of bits.
func NewBuckets(count uint, bucketSize uint8) *Buckets {
	return &Buckets{
		total:      count,
		count:      0,
		data:       make([]byte, (count*uint(bucketSize)+7)/8),
		bucketSize: bucketSize,
		max:        (1 << bucketSize) - 1,
	}
}

// MaxBucketValue returns the maximum value that can be stored in a bucket.
func (b *Buckets) MaxBucketValue() uint8 {
	return b.max
}

func (b *Buckets) FullRatio() float64 {
	return float64(b.count) / float64(b.total)
}

// Increment will increment the value in the specified bucket by the provided
// delta. A bucket can be decremented by providing a negative delta. The value
// is clamped to zero and the maximum bucket value. Returns itself to allow for
// chaining.
//func (b *Buckets) Increment(bucket uint, delta int32) *Buckets {
//	val := int32(b.getBits(bucket*uint(b.bucketSize), uint(b.bucketSize))) + delta
//	if val > int32(b.max) {
//		val = int32(b.max)
//	} else if val < 0 {
//		val = 0
//	}
//
//	b.setBits(uint32(bucket)*uint32(b.bucketSize), uint32(b.bucketSize), uint32(val))
//	return b
//}

// Set will set the bucket value. The value is clamped to zero and the maximum
// bucket value. Returns itself to allow for chaining.
func (b *Buckets) Set(bucket uint, value uint8) *Buckets {
	if value > b.max {
		value = b.max
	}

	b.setBits(uint32(bucket)*uint32(b.bucketSize), uint32(b.bucketSize), uint32(value))
	b.count++
	return b
}

// Get returns the value in the specified bucket.
func (b *Buckets) Get(bucket uint) uint32 {
	return b.getBits(bucket*uint(b.bucketSize), uint(b.bucketSize))
}

// Reset restores the Buckets to the original state. Returns itself to allow
// for chaining.
func (b *Buckets) Reset() *Buckets {
	for i := range b.data {
		b.data[i] = 0
	}
	b.count = 0
	return b
}

// getBits returns the bits at the specified offset and length.
func (b *Buckets) getBits(offset, length uint) uint32 {
	byteIndex := offset / 8
	byteOffset := offset % 8
	if byteOffset+length > 8 {
		rem := 8 - byteOffset
		return b.getBits(offset, rem) | (b.getBits(offset+rem, length-rem) << rem)
	}
	bitMask := uint32((1 << length) - 1)
	return (uint32(b.data[byteIndex]) & (bitMask << byteOffset)) >> byteOffset
}

// setBits sets bits at the specified offset and length.
func (b *Buckets) setBits(offset, length, bits uint32) {
	byteIndex := offset / 8
	byteOffset := offset % 8
	if byteOffset+length > 8 {
		rem := 8 - byteOffset
		b.setBits(offset, rem, bits)
		b.setBits(offset+rem, length-rem, bits>>rem)
		return
	}
	bitMask := uint32((1 << length) - 1)
	b.data[byteIndex] = byte(uint32(b.data[byteIndex]) & ^(bitMask << byteOffset))
	b.data[byteIndex] = byte(uint32(b.data[byteIndex]) | ((bits & bitMask) << byteOffset))
}
