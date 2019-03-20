package vm_db

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/vitelabs/go-vite/interfaces"
)

func (db *vmDb) NewStorageIterator(prefix []byte) (interfaces.StorageIterator, error) {
	iter, err := db.chain.GetStateIterator(db.address, prefix)
	if err != nil {
		return nil, err
	}

	unsavedIter := db.unsaved.NewStorageIterator(prefix)
	return newStorageIterator([]interfaces.StorageIterator{
		unsavedIter,
		iter,
	}, map[string]struct{}{}), nil
}

type storageIterator struct {
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

func newStorageIterator(iters []interfaces.StorageIterator, deletedList map[string]struct{}) interfaces.StorageIterator {
	return &storageIterator{
		cmp:         comparer.DefaultComparer,
		deletedList: deletedList,
		iters:       iters,
		finishIters: make([]interfaces.StorageIterator, 0, len(iters)),
		keys:        make([][]byte, len(iters)),
	}
}

func (si *storageIterator) Next() bool {
	si.keys[si.index] = nil

	minKeyIndex := -1
	var minKey []byte

	for i := 0; i < len(si.iters); i++ {
		iter := si.iters[i]

		if iter == nil {
			continue
		}

		key := si.keys[i]
		for {
			if key == nil {
				if !iter.Next() {
					if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
						si.err = err
						return false
					}
					si.iters[i] = nil
					si.finishIters = append(si.finishIters, iter)
					break
				}
				key = iter.Key()
			}

			if _, ok := si.deletedList[string(key)]; ok {
				key = nil
				si.keys[i] = nil
				continue
			}

			if bytes.Equal(key, si.lastKey) {
				key = nil
				si.keys[i] = nil
				continue
			}
			si.keys[i] = key
			break
		}

		if key != nil && (si.cmp.Compare(minKey, key) < 0 || len(minKey) <= 0) {
			minKey = key
			minKeyIndex = i

		}
	}

	if minKeyIndex < 0 {
		return false
	}

	si.index = minKeyIndex
	si.lastKey = minKey

	return true
}

func (si *storageIterator) Key() []byte {
	if si.err != nil {
		return nil
	}
	return si.keys[si.index]
}

func (si *storageIterator) Value() []byte {
	if len(si.currentKey) <= 0 || si.err != nil {
		return nil
	}

	return si.iters[si.index].Value()
}

func (si *storageIterator) Error() error {
	return si.err
}

func (si *storageIterator) Release() {
	for _, iter := range si.iters {
		if iter != nil {
			iter.Release()
		}
	}
	for _, iter := range si.finishIters {
		iter.Release()
	}
}
