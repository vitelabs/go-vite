package net

import (
	"crypto/rand"
	"io"
	mrand "math/rand"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestFilter_record(t *testing.T) {
	f := newBlockFilter(1000)

	count := mrand.Intn(100000)
	m := make(map[types.Hash]struct{}, count)

	for i := 0; i < count; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}

		m[hash] = struct{}{}

		f.record(hash[:])

		if !f.has(hash[:]) {
			t.Fail()
		}
	}
}

func TestFilter_has(t *testing.T) {
	f := newBlockFilter(1000)

	count := mrand.Intn(100000)
	m := make(map[types.Hash]struct{}, count)

	for i := 0; i < count; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}

		m[hash] = struct{}{}

		f.record(hash[:])
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
		if f.has(hash[:]) {
			failed++
		}
	}

	t.Logf("failed: %d, count: %d\n", failed, count)
}

func TestFilter_LookAndRecord(t *testing.T) {
	f := newBlockFilter(1000)

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

		exist := f.lookAndRecord(hash[:])
		if exist {
			failed++
		}

		exist = f.lookAndRecord(hash[:])
		if !exist {
			failed++
		}
	}

	t.Logf("failed %d, count: %d", failed, count)
}

func BenchmarkBlockFilter_has(b *testing.B) {
	// 60ns
	f := newBlockFilter(filterCap)

	var hash types.Hash
	_, _ = rand.Read(hash[:])

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			f.has(hash[:])
		}
	})
}

func BenchmarkBlockFilter_record(b *testing.B) {
	// 500ns
	f := newBlockFilter(filterCap)

	b.RunParallel(func(pb *testing.PB) {
		var hash types.Hash

		for pb.Next() {
			_, _ = rand.Read(hash[:])
			f.record(hash[:])
		}
	})
}

func BenchmarkBlockFilter_lookAndRecord(b *testing.B) {
	// 700ns
	f := newBlockFilter(filterCap)

	b.RunParallel(func(pb *testing.PB) {
		var hash types.Hash
		for pb.Next() {
			_, _ = rand.Read(hash[:])
			f.lookAndRecord(hash[:])
		}
	})
}

// 1500ns
func BenchmarkCrypto(b *testing.B) {
	var hash types.Hash
	for i := 0; i < b.N; i++ {
		_, _ = rand.Read(hash[:])
	}
}
