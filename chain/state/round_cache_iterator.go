package chain_state

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/iterator"
)

type RoundCacheIterator struct {
	iter iterator.Iterator
}

func NewRoundCacheIterator(iter iterator.Iterator) *RoundCacheIterator {
	return &RoundCacheIterator{
		iter: iter,
	}
}

func (rcIter *RoundCacheIterator) Last() bool {
	return rcIter.iter.Last()
}

func (rcIter *RoundCacheIterator) Prev() bool {
	return rcIter.iter.Prev()
}

func (rcIter *RoundCacheIterator) Seek(key []byte) bool {
	return rcIter.iter.Seek(key)
}

func (rcIter *RoundCacheIterator) Next() bool {
	return rcIter.iter.Next()
}

func (rcIter *RoundCacheIterator) Key() []byte {
	return rcIter.iter.Key()[1:]
}

func (rcIter *RoundCacheIterator) Value() []byte {
	return rcIter.iter.Value()
}

func (rcIter *RoundCacheIterator) Error() error {
	return rcIter.iter.Error()
}

func (rcIter *RoundCacheIterator) Release() {
	rcIter.iter.Release()
}
