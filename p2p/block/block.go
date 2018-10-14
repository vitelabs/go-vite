package block

import (
	"github.com/seiflotfy/cuckoofilter"
)

// use to block specified net.IP or NodeID
type Set struct {
	filter *cuckoofilter.CuckooFilter
}

func (s *Set) Has(v []byte) bool {
	return s.filter.Lookup(v)
}

func (s *Set) Add(v []byte) {
	s.filter.InsertUnique(v)
}
func (s *Set) Del(v []byte) {
	s.filter.Delete(v)
}

func (s *Set) Count() uint {
	return s.filter.Count()
}

func New(cap uint) *Set {
	return &Set{
		filter: cuckoofilter.NewCuckooFilter(cap),
	}
}
