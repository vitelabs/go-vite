package net

import (
	"sync"

	"github.com/jerry-vite/cuckoofilter"
)

type blockFilter interface {
	has(b []byte) bool
	record(b []byte)
	lookAndRecord(b []byte) bool
}

type defBlockFilter struct {
	rw   sync.RWMutex
	pool *cuckoo.Filter
}

func newBlockFilter(cap uint) blockFilter {
	return &defBlockFilter{
		pool: cuckoo.NewFilter(cap),
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

	d.recordLocked(b)
}

func (d *defBlockFilter) recordLocked(b []byte) {
	success := d.pool.Insert(b)
	if !success {
		d.pool.Reset()
		d.pool.Insert(b)
	}
}

func (d *defBlockFilter) lookAndRecord(b []byte) bool {
	d.rw.Lock()
	defer d.rw.Unlock()

	ok := d.pool.Lookup(b)
	if ok {
		return ok
	}

	d.recordLocked(b)
	return false
}
