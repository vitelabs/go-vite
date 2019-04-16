package net

import (
	"sync"

	"github.com/tylertreat/BoomFilters"
)

const filterCap = 100000
const rt = 0.001

type blockFilter interface {
	has(b []byte) bool
	record(b []byte)
	lookAndRecord(b []byte) (hasExist bool)
}

type defBlockFilter struct {
	rw   sync.RWMutex
	pool *boom.CountingBloomFilter
	cp   uint
	th   uint
}

func newBlockFilter(cp uint) blockFilter {
	return &defBlockFilter{
		pool: boom.NewDefaultCountingBloomFilter(cp, rt),
		cp:   cp,
		th:   cp * 9 / 10,
	}
}

func (d *defBlockFilter) has(b []byte) bool {
	d.rw.RLock()
	defer d.rw.RUnlock()

	return d.pool.Test(b)
}

func (d *defBlockFilter) record(b []byte) {
	d.rw.Lock()
	defer d.rw.Unlock()

	d.recordLocked(b)
}

func (d *defBlockFilter) recordLocked(b []byte) {
	if d.pool.Count() > d.th {
		d.pool.Reset()
	}

	d.pool.Add(b)
}

func (d *defBlockFilter) lookAndRecord(b []byte) bool {
	d.rw.Lock()
	defer d.rw.Unlock()

	ok := d.pool.Test(b)
	if ok {
		return ok
	}

	d.recordLocked(b)
	return false
}
