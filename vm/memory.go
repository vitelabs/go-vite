package vm

import (
	"encoding/hex"
	"math/big"
	"strconv"
)

type memory struct {
	store       []byte
	lastGasCost uint64
}

func newMemory() *memory {
	return &memory{}
}

// resize resizes the memory to size
func (m *memory) resize(size uint64) {
	if uint64(m.len()) < size {
		m.store = append(m.store, make([]byte, size-uint64(m.len()))...)
	}
}

// len returns the length of the backing slice
func (m *memory) len() int {
	return len(m.store)
}

// get returns offset + size as a new slice
func (m *memory) get(offset, size int64) (cpy []byte) {
	if size == 0 {
		return nil
	}

	if len(m.store) > int(offset) {
		cpy = make([]byte, size)
		copy(cpy, m.store[offset:offset+size])

		return cpy
	}

	return nil
}

// getPtr returns the offset + size
func (m *memory) getPtr(offset, size int64) []byte {
	if size == 0 {
		return nil
	}

	if len(m.store) > int(offset) {
		return m.store[offset : offset+size]
	}

	return nil
}

// set sets offset + size to amount
func (m *memory) set(offset, size uint64, value []byte) {
	// It's possible the offset is greater than 0 and size equals 0. This is because
	// the calcMemSize (common.go) could potentially return 0 when size is zero (NO-OP)
	if size > 0 {
		// length of store may never be less than offset + size.
		// The store should be resized PRIOR to setting the memory
		if offset+size > uint64(len(m.store)) {
			panic("invalid memory: store empty")
		}
		copy(m.store[offset:offset+size], value)
	}
}

// set32 sets the 32 bytes starting at offset to the amount of val, left-padded with zeroes to
// 32 bytes.
func (m *memory) set32(offset uint64, val *big.Int) {
	// length of store may never be less than offset + size.
	// The store should be resized PRIOR to setting the memory
	if offset+32 > uint64(len(m.store)) {
		panic("invalid memory: store empty")
	}
	// Zero the memory area
	copy(m.store[offset:offset+32], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
	// Fill in relevant bits
	ReadBits(val, m.store[offset:offset+32])
}

func (m *memory) print() string {
	var result string
	if len(m.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.store); i += 32 {
			if i+32 < len(m.store) {
				result += strconv.Itoa(addr) + "=>" + hex.EncodeToString(m.store[i:i+32]) + ", "
			} else {
				result += strconv.Itoa(addr) + "=>" + hex.EncodeToString(m.store[i:i+32])
			}
			addr++
		}
	}
	return result
}
