package dbutils

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/vitelabs/go-vite/interfaces"
)

const (
	iterPointHead     = 0
	iterPointerMiddle = 1
	iterPointTail     = 2
)

type MergedIterator struct {
	cmp      comparer.BasicComparer
	isDelete func([]byte) bool

	iters []interfaces.StorageIterator

	iterStatus []byte

	index int

	keys [][]byte

	prevKey []byte

	err error

	directionToNext bool
}

func NewMergedIterator(iters []interfaces.StorageIterator, isDelete func([]byte) bool) interfaces.StorageIterator {
	return &MergedIterator{
		cmp:             comparer.DefaultComparer,
		isDelete:        isDelete,
		directionToNext: true,
		iters:           iters,
		iterStatus:      make([]byte, len(iters)),
		keys:            make([][]byte, len(iters)),
	}
}

func (mi *MergedIterator) reset() {
	mi.iterStatus = make([]byte, len(mi.iters))
	mi.index = -1

	mi.keys = make([][]byte, len(mi.iters))
	mi.prevKey = nil
}

func (mi *MergedIterator) Last() bool {
	mi.reset()

	mi.directionToNext = false

	for i := 0; i < len(mi.iters); i++ {
		iter := mi.iters[i]
		if iter.Last() {
			mi.iterStatus[i] = iterPointTail
			mi.keys[i] = iter.Key()
		} else {
			mi.iterStatus[i] = iterPointHead
		}

	}

	return mi.Prev()
}
func (mi *MergedIterator) Prev() bool {
	return mi.step(false)
}
func (mi *MergedIterator) Next() bool {
	return mi.step(true)
}

func (mi *MergedIterator) Key() []byte {
	if mi.err != nil {
		return nil
	}
	return mi.keys[mi.index]
}

func (mi *MergedIterator) Value() []byte {
	if len(mi.keys[mi.index]) <= 0 || mi.err != nil {
		return nil
	}

	return mi.iters[mi.index].Value()
}

func (mi *MergedIterator) Error() error {
	return mi.err
}

func (mi *MergedIterator) Release() {
	for _, iter := range mi.iters {
		iter.Release()
	}
}

func (mi *MergedIterator) step(toNext bool) bool {
	if mi.err != nil {
		return false
	}

	if (mi.directionToNext && !toNext) || (!mi.directionToNext && toNext) {
		mi.reset()
	}

	if mi.index >= 0 {
		mi.keys[mi.index] = nil
	}

	fitKeyIndex := -1
	var fitKey []byte

	for i := 0; i < len(mi.iters); i++ {
		iter := mi.iters[i]

		if (toNext && mi.iterStatus[i] == iterPointTail) ||
			(!toNext && mi.iterStatus[i] == iterPointHead) {
			continue
		}

		key := mi.keys[i]
		for {
			if key == nil {
				if (toNext && !iter.Next()) ||
					(!toNext && !iter.Prev()) {

					if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
						mi.err = err
						return false
					}

					if toNext {
						mi.iterStatus[i] = iterPointTail
					} else {
						mi.iterStatus[i] = iterPointHead
					}
					break
				}

				key = iter.Key()

				if mi.isDelete != nil && mi.isDelete(key) {
					key = nil
					continue
				}

			} else if bytes.Equal(key, mi.prevKey) {
				key = nil
				mi.keys[i] = nil
				continue
			}

			mi.keys[i] = key
			break
		}

		if key != nil {
			compareResult := mi.cmp.Compare(key, fitKey)
			if (toNext && compareResult < 0) || (!toNext && compareResult > 0) || len(fitKey) <= 0 {
				fitKey = key
				fitKeyIndex = i
			}
		}
	}

	if fitKeyIndex < 0 {
		return false
	}

	mi.index = fitKeyIndex
	mi.prevKey = fitKey

	return true
}
