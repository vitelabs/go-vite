package block

import (
	"github.com/seiflotfy/cuckoofilter"
	"sync"
)

// use to block specified net.IP or NodeID
type CuckooSet struct {
	lock   sync.RWMutex
	filter *cuckoofilter.CuckooFilter
}

func (s *CuckooSet) Has(v []byte) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.filter.Lookup(v)
}

func (s *CuckooSet) Add(v []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.filter.InsertUnique(v)
}
func (s *CuckooSet) Del(v []byte) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.filter.Delete(v)
}

func (s *CuckooSet) Count() uint {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.filter.Count()
}

func NewCuckooSet(cap uint) *CuckooSet {
	return &CuckooSet{
		filter: cuckoofilter.NewCuckooFilter(cap),
	}
}
