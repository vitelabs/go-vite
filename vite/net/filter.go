package net

import (
	"sync"
	"time"

	"github.com/seiflotfy/cuckoofilter"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/monitor"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
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

const maxMark = 5

var timeThreshold = 5 * time.Second

var logFilter = log15.New("module", "net/filter")

// use to filter redundant fetch

type Filter interface {
	hold(hash types.Hash) bool
	done(hash types.Hash)
	has(hash types.Hash) bool
}

type record struct {
	addAt  time.Time
	doneAt time.Time
	mark   int
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
	records map[types.Hash]*record
	lock    sync.RWMutex
	log     log15.Logger
	term    chan struct{}
	wg      sync.WaitGroup
}

func newFilter() *filter {
	return &filter{
		records: make(map[types.Hash]*record, 10000),
		log:     logFilter,
	}
}

func (f *filter) start() {
	f.term = make(chan struct{})

	f.wg.Add(1)
	common.Go(f.loop)
}

func (f *filter) stop() {
	if f.term == nil {
		return
	}

	select {
	case <-f.term:
	default:
		close(f.term)
	}
}

func (f *filter) loop() {
	defer f.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-f.term:
			return
		case now := <-ticker.C:
			f.lock.Lock()

			for hash, r := range f.records {
				if r._done && now.Sub(r.doneAt) > timeThreshold {
					delete(f.records, hash)
				}
			}

			f.lock.Unlock()

			f.log.Info("filter clean")
		}
	}
}

// will suppress fetch
func (f *filter) hold(hash types.Hash) bool {
	defer monitor.LogTime("net/filter", "hold", time.Now())

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

		r.inc()
	} else {
		f.records[hash] = &record{addAt: time.Now()}
		return false
	}

	return true
}

func (f *filter) done(hash types.Hash) {
	defer monitor.LogTime("net/filter", "done", time.Now())

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
	defer monitor.LogTime("net/filter", "has", time.Now())

	f.lock.RLock()
	defer f.lock.RUnlock()

	r, ok := f.records[hash]
	return ok && r._done
}

func (f *filter) fail(hash types.Hash) {
	defer monitor.LogTime("net/filter", "fail", time.Now())

	f.lock.RLock()
	defer f.lock.RUnlock()

	if r, ok := f.records[hash]; ok {
		if r._done {
			return
		}

		delete(f.records, hash)
	}
}
