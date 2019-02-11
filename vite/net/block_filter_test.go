package net

import (
	"crypto/rand"
	"io"
	"testing"

	"github.com/vitelabs/go-vite/common/types"
)

func TestFilter_Has(t *testing.T) {
	const cap = 1000
	filter := newBlockFilter(cap)

	m := make(map[types.Hash]struct{})

	for i := 0; i < cap*10; i++ {
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

	for i := 0; i < cap*10; i++ {
		var hash types.Hash
		if _, err := io.ReadFull(rand.Reader, hash[:]); err != nil {
			continue
		}
		if _, ok := m[hash]; ok {
			continue
		}
		if filter.has(hash[:]) {
			t.Log("should not has")
			t.Fail()
		}
	}
}

func TestFilter_LookAndRecord(t *testing.T) {
	const cap = 1000
	filter := newBlockFilter(cap)

	m := make(map[types.Hash]struct{})

	for i := 0; i < cap*10; i++ {
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
			t.Log("should not exist")
			t.Fail()
		}

		exist = filter.lookAndRecord(hash[:])
		if !exist {
			t.Log("should exist")
			t.Fail()
		}
	}
}
