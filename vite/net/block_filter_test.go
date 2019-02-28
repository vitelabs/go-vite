package net

import (
	"crypto/rand"
	"io"
	mrand "math/rand"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestFilter_record(t *testing.T) {
	filter := newBlockFilter(1000)

	count := mrand.Intn(100000)
	m := make(map[types.Hash]struct{}, count)

	for i := 0; i < count; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}

		m[hash] = struct{}{}

		filter.record(hash[:])

		if !filter.has(hash[:]) {
			t.Fail()
		}
	}
}

func TestFilter_has(t *testing.T) {
	filter := newBlockFilter(1000)

	count := mrand.Intn(100000)
	m := make(map[types.Hash]struct{}, count)

	for i := 0; i < count; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}

		m[hash] = struct{}{}

		filter.record(hash[:])
	}

	count = mrand.Intn(100000)
	var failed int
	for i := 0; i < count; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}
		if _, ok := m[hash]; ok {
			continue
		}
		if filter.has(hash[:]) {
			failed++
		}
	}

	t.Logf("failed: %d, count: %d\n", failed, count)
}

func TestFilter_LookAndRecord(t *testing.T) {
	filter := newBlockFilter(1000)

	count := mrand.Intn(100000)
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

		exist := filter.lookAndRecord(hash[:])
		if exist {
			failed++
		}

		exist = filter.lookAndRecord(hash[:])
		if !exist {
			failed++
		}
	}

	t.Logf("failed %d, count: %d", failed, count)
}
