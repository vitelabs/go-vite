package net

import (
	"github.com/seiflotfy/cuckoofilter"
	"sync"
)

type cuckooSet struct {
	lock   sync.RWMutex
	filter *cuckoofilter.CuckooFilter
}

func (s *cuckooSet) Has(v interface{}) bool {
	bytes, ok := v.([]byte)
	if !ok {
		return false
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.filter.Lookup(bytes)
}

func (s *cuckooSet) Add(v interface{}) {
	bytes, ok := v.([]byte)
	if !ok {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.filter.InsertUnique(bytes)
}
func (s *cuckooSet) Del(v interface{}) {
	bytes, ok := v.([]byte)
	if !ok {
		return
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.filter.Delete(bytes)
}
func (s *cuckooSet) Count() uint {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.filter.Count()
}

func NewCuckooSet(cap uint) *cuckooSet {
	return &cuckooSet{
		filter: cuckoofilter.NewCuckooFilter(cap),
	}
}
