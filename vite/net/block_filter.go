package net

import (
	"sync"

	"github.com/seiflotfy/cuckoofilter"
)

type blockFilter interface {
	has(b []byte) bool
	record(b []byte)
	lookAndRecord(b []byte) bool
}

type defBlockFilter struct {
	rw   sync.RWMutex
	pool *cuckoofilter.CuckooFilter
}

func newBlockFilter(cap uint) blockFilter {
	return &defBlockFilter{
		pool: cuckoofilter.NewCuckooFilter(cap),
	}
}

func (d *defBlockFilter) has(b []byte) bool {
	d.rw.RLock()
	defer d.rw.RUnlock()

	return d.pool.Lookup(b)
}

func (d *defBlockFilter) record(b []byte) {
	d.rw.Lock()
	defer d.rw.Unlock()

	d.pool.Insert(b)
}

func (d *defBlockFilter) lookAndRecord(b []byte) bool {
	d.rw.Lock()
	defer d.rw.Unlock()

	ok := d.pool.Lookup(b)
	if ok {
		return ok
	}

	d.pool.Insert(b)
	return false
}
