package dbutils

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/vitelabs/go-vite/interfaces"
)

type MergedIterator struct {
	cmp         comparer.BasicComparer
	iters       []interfaces.StorageIterator
	finishIters []interfaces.StorageIterator

	deletedList map[string]struct{}
	index       int

	lastKey []byte

	keys [][]byte

	currentKey []byte

	err error
}

func NewMergedIterator(iters []interfaces.StorageIterator, deletedList map[string]struct{}) interfaces.StorageIterator {
	return &MergedIterator{
		cmp:         comparer.DefaultComparer,
		deletedList: deletedList,
		iters:       iters,
		finishIters: make([]interfaces.StorageIterator, 0, len(iters)),
		keys:        make([][]byte, len(iters)),
	}
}

func (mi *MergedIterator) Next() bool {
	mi.keys[mi.index] = nil

	minKeyIndex := -1
	var minKey []byte

	for i := 0; i < len(mi.iters); i++ {
		iter := mi.iters[i]

		if iter == nil {
			continue
		}

		key := mi.keys[i]
		for {
			if key == nil {
				if !iter.Next() {
					if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
						mi.err = err
						return false
					}
					mi.iters[i] = nil
					mi.finishIters = append(mi.finishIters, iter)
					break
				}
				key = iter.Key()
			}

			if mi.deletedList != nil {
				if _, ok := mi.deletedList[string(key)]; ok {
					key = nil
					mi.keys[i] = nil
					continue
				}
			}

			if bytes.Equal(key, mi.lastKey) {
				key = nil
				mi.keys[i] = nil
				continue
			}
			mi.keys[i] = key
			break
		}

		if key != nil && (mi.cmp.Compare(minKey, key) < 0 || len(minKey) <= 0) {
			minKey = key
			minKeyIndex = i

		}
	}

	if minKeyIndex < 0 {
		return false
	}

	mi.index = minKeyIndex
	mi.lastKey = minKey

	return true
}

func (mi *MergedIterator) Key() []byte {
	if mi.err != nil {
		return nil
	}
	return mi.keys[mi.index]
}

func (mi *MergedIterator) Value() []byte {
	if len(mi.currentKey) <= 0 || mi.err != nil {
		return nil
	}

	return mi.iters[mi.index].Value()
}

func (mi *MergedIterator) Error() error {
	return mi.err
}

func (mi *MergedIterator) Release() {
	for _, iter := range mi.iters {
		if iter != nil {
			iter.Release()
		}
	}
	for _, iter := range mi.finishIters {
		iter.Release()
	}
}
