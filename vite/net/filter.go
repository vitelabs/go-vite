package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"sync/atomic"
	"time"
)

const maxMark = 3

var timeThreshold = 20 * time.Second

// use to filter redundant fetch

type Filter interface {
}

type record struct {
	addAt time.Time
	mark  int32 // atomic, how many times has fetched
	state reqState
}

func (r *record) inc() {
	atomic.AddInt32(&r.mark, 1)
}

type filter struct {
	chain   *skeleton
	records map[types.Hash]*record
}

func newFilter() *filter {
	return &filter{}
}

func (f *filter) pass(hash types.Hash) bool {
	return true
}
