package dbutils

import (
	"bytes"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/comparer"
	"github.com/vitelabs/go-vite/interfaces"
)

type MergedIterator struct {
	cmp         comparer.BasicComparer
	deletedKeys map[string]struct{}

	iters      []interfaces.StorageIterator
	iterIsLast []byte

	index int

	keys [][]byte

	prevKey []byte

	err error
}

func NewMergedIterator(iters []interfaces.StorageIterator, deletedKeys map[string]struct{}) interfaces.StorageIterator {
	return &MergedIterator{
		cmp:         comparer.DefaultComparer,
		deletedKeys: deletedKeys,
		iters:       iters,
		iterIsLast:  make([]byte, len(iters)),
		keys:        make([][]byte, len(iters)),
	}
}

func (mi *MergedIterator) reset() {
	mi.iterIsLast = make([]byte, len(mi.iters))
	mi.index = 0

	mi.keys = make([][]byte, len(mi.iters))
	mi.prevKey = nil
}

func (mi *MergedIterator) Last() bool {
	if mi.err != nil {
		return false
	}
	mi.reset()

	maxKeyIndex := -1
	var maxKey []byte
	for i := 0; i < len(mi.iters); i++ {
		iter := mi.iters[i]
		if !iter.Last() {
			continue
		}

		key := iter.Key()
		if mi.cmp.Compare(maxKey, key) < 0 || len(maxKey) <= 0 {
			maxKey = key
			maxKeyIndex = i
		}
	}
	if maxKeyIndex < 0 {
		return false
	}

	mi.index = maxKeyIndex
	mi.prevKey = maxKey

	return true

}
func (mi *MergedIterator) Next() bool {
	if mi.err != nil {
		return false
	}
	mi.keys[mi.index] = nil

	minKeyIndex := -1
	var minKey []byte

	for i := 0; i < len(mi.iters); i++ {
		iter := mi.iters[i]

		if mi.iterIsLast[i] > 0 {
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

					mi.iterIsLast[i] = 1
					break
				}
				key = iter.Key()
			}

			if mi.deletedKeys != nil {
				if _, ok := mi.deletedKeys[string(key)]; ok {
					key = nil
					mi.keys[i] = nil
					continue
				}
			}

			if bytes.Equal(key, mi.prevKey) {
				key = nil
				mi.keys[i] = nil
				continue
			}
			mi.keys[i] = key
			break
		}

		if key != nil && (mi.cmp.Compare(key, minKey) < 0 || len(minKey) <= 0) {
			minKey = key
			minKeyIndex = i

		}
	}

	if minKeyIndex < 0 {
		return false
	}

	mi.index = minKeyIndex
	mi.prevKey = minKey

	return true
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
