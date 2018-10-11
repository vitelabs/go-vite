package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"sync"
	"time"
)

const maxMark = 5

var timeThreshold = 5 * time.Second

// use to filter redundant fetch

type Filter interface {
	hold(hash types.Hash) bool
	done(hash types.Hash)
	has(hash types.Hash) bool
}

type record struct {
	addAt  time.Time
	doneAt time.Time
	mark   int32 // atomic, how many times has fetched
	_done  bool
}

func (r *record) inc() {
	r.mark += 1
}

func (r *record) reset() {
	r.mark = 0
	r._done = false
	r.addAt = time.Now()
}

func (r *record) done() {
	r.doneAt = time.Now()
	r._done = true
}

type filter struct {
	chain   *skeleton
	records map[types.Hash]*record
	lock    sync.RWMutex
	log     log15.Logger
}

func newFilter() *filter {
	return &filter{
		records: make(map[types.Hash]*record, 10000),
		log:     log15.New("module", "net/filter"),
	}
}

// will suppress fetch
func (f *filter) hold(hash types.Hash) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	if r, ok := f.records[hash]; ok {
		if r._done {
			if r.mark >= maxMark && time.Now().Sub(r.doneAt) >= timeThreshold {
				r.reset()
				return false
			}
		} else {
			if r.mark >= maxMark*2 && time.Now().Sub(r.addAt) >= timeThreshold*2 {
				r.reset()
				return false
			}
		}

		f.records[hash].inc()
	} else {
		f.records[hash] = &record{addAt: time.Now()}
		return false
	}

	return true
}

func (f *filter) done(hash types.Hash) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if r, ok := f.records[hash]; ok {
		r.done()
	} else {
		f.records[hash] = &record{addAt: time.Now()}
		f.records[hash].done()
	}
}

func (f *filter) has(hash types.Hash) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()

	_, ok := f.records[hash]
	return ok
}
