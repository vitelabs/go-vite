package chain_state

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/iterator"
)

type TransformIterator struct {
	iter            iterator.Iterator
	keyPrefixLength int
}

func NewTransformIterator(iter iterator.Iterator, keyPrefixLength int) *TransformIterator {
	return &TransformIterator{
		iter:            iter,
		keyPrefixLength: keyPrefixLength,
	}
}

func (rcIter *TransformIterator) Last() bool {
	return rcIter.iter.Last()
}

func (rcIter *TransformIterator) Prev() bool {
	return rcIter.iter.Prev()
}

func (rcIter *TransformIterator) Seek(key []byte) bool {
	return rcIter.iter.Seek(key)
}

func (rcIter *TransformIterator) Next() bool {
	return rcIter.iter.Next()
}

func (rcIter *TransformIterator) Key() []byte {
	return rcIter.iter.Key()[rcIter.keyPrefixLength:]
}

func (rcIter *TransformIterator) Value() []byte {
	return rcIter.iter.Value()
}

func (rcIter *TransformIterator) Error() error {
	return rcIter.iter.Error()
}

func (rcIter *TransformIterator) Release() {
	rcIter.iter.Release()
}
