package chain_state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func (sDB *StateDB) NewStorageIterator(addr *types.Address, prefix []byte) interfaces.StorageIterator {
	slice := util.BytesPrefix(chain_utils.CreateStorageValueKeyPrefix(addr, prefix))
	return newStateStorageIterator(dbutils.NewMergedIterator([]interfaces.StorageIterator{
		sDB.pending.NewIterator(slice),
		sDB.db.NewIterator(slice, nil),
	}, sDB.pending.IsDelete))
}

func (sDB *StateDB) NewSnapshotStorageIterator(snapshotHash *types.Hash, addr *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	if *snapshotHash == sDB.chain.GetLatestSnapshotBlock().Hash {
		return sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateStorageValueKeyPrefix(addr, prefix)), nil), nil
	}

	height, err := sDB.chain.GetSnapshotHeightByHash(*snapshotHash)
	if err != nil {
		sErr := errors.New(fmt.Sprintf("sDB.chain.GetSnapshotHeightByHash failed, hash is %s. Error: %s", snapshotHash, err))
		return nil, sErr
	}

	if height <= 0 {
		return nil, nil
	}

	iterator := newStateStorageIterator(newSnapshotStorageIterator(sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateHistoryStorageValueKeyPrefix(addr, prefix)), nil), height))
	return iterator, nil
}

type stateStorageIterator struct {
	iter interfaces.StorageIterator
}

func newStateStorageIterator(iter interfaces.StorageIterator) interfaces.StorageIterator {
	return &stateStorageIterator{
		iter: iter,
	}
}

// TODO
func (iterator *stateStorageIterator) Last() bool {
	return iterator.iter.Last()
}

func (iterator *stateStorageIterator) Prev() bool {
	return iterator.iter.Prev()
}

func (iterator *stateStorageIterator) Next() bool {
	return iterator.iter.Next()
}

func (iterator *stateStorageIterator) Key() []byte {
	key := iterator.iter.Key()

	if len(key) <= 0 {
		return nil
	}

	keySize := key[34]
	return key[1 : 1+keySize]
}

func (iterator *stateStorageIterator) Value() []byte {
	return iterator.iter.Value()
}
func (iterator *stateStorageIterator) Error() error {
	err := iterator.iter.Error()
	if err != leveldb.ErrNotFound {
		return err
	}
	return nil
}

func (iterator *stateStorageIterator) Release() {
	iterator.iter.Release()
}

type snapshotStorageIterator struct {
	iter   iterator.Iterator
	iterOk bool

	snapshotHeight uint64

	lastKey []byte
}

func newSnapshotStorageIterator(iter iterator.Iterator, height uint64) interfaces.StorageIterator {
	sIterator := &snapshotStorageIterator{
		iter:           iter,
		snapshotHeight: height,
		iterOk:         true,
	}

	//iterator.iterOk = iter.Last()
	return sIterator
}

func (sIterator *snapshotStorageIterator) Last() bool {
	sIterator.iterOk = sIterator.iter.Last()
	sIterator.lastKey = nil
	return sIterator.iterOk
}

func (sIterator *snapshotStorageIterator) Prev() bool {
	return sIterator.step(false)
}

func (sIterator *snapshotStorageIterator) Next() bool {
	return sIterator.step(true)
}

func (sIterator *snapshotStorageIterator) Key() []byte {
	return sIterator.iter.Key()
}
func (sIterator *snapshotStorageIterator) Value() []byte {
	return sIterator.iter.Value()
}
func (sIterator *snapshotStorageIterator) Error() error {
	return sIterator.iter.Error()
}

func (sIterator *snapshotStorageIterator) Release() {
	sIterator.iter.Release()
}

func (sIterator *snapshotStorageIterator) step(isNext bool) bool {
	iter := sIterator.iter

	for sIterator.iterOk {
		if len(sIterator.lastKey) > 0 {
			if isNext {
				binary.BigEndian.PutUint64(sIterator.lastKey[len(sIterator.lastKey)-8:], helper.MaxUint64)
				sIterator.iterOk = iter.Seek(sIterator.lastKey)
			} else {
				binary.BigEndian.PutUint64(sIterator.lastKey[len(sIterator.lastKey)-8:], 0)
				sIterator.iterOk = iter.Seek(sIterator.lastKey)
				if sIterator.iterOk {
					sIterator.iterOk = iter.Prev()
				}
			}

		} else {
			if isNext {
				sIterator.iterOk = iter.Next()
			} else {
				sIterator.iterOk = iter.Prev()
			}
		}

		if !sIterator.iterOk {
			break
		}

		key := iter.Key()
		sIterator.lastKey = key

		binary.BigEndian.PutUint64(key[len(key)-8:], sIterator.snapshotHeight)
		sIterator.iterOk = iter.Seek(key)

		if sIterator.iterOk {
			seekKey := iter.Key()
			height := binary.BigEndian.Uint64(seekKey[len(seekKey)-8:])
			if height <= sIterator.snapshotHeight {
				break
			}
			if iter.Prev() {
				prevKey := iter.Key()
				if bytes.Equal(prevKey[:len(prevKey)-8], key[:len(key)-8]) {
					break
				}
			}

		}
	}

	return sIterator.iterOk
}
