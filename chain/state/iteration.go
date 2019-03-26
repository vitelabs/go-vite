package chain_state

import (
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/dbutils"
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

	return newSnapshotStorageIterator(sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateHistoryStorageValueKeyPrefix(addr, prefix)), nil), height), err
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
	iter           iterator.Iterator
	iterOk         bool
	hasSeek        bool
	snapshotHeight uint64

	currentKey []byte
}

func newSnapshotStorageIterator(iter iterator.Iterator, height uint64) interfaces.StorageIterator {
	iterator := &snapshotStorageIterator{
		iter:           iter,
		snapshotHeight: height,
	}
	iterator.iterOk = iter.Last()
	return iterator
}

func (iterator *snapshotStorageIterator) Last() bool {
	iterator.iterOk = iterator.iter.First()
	iterator.currentKey = nil
	iterator.hasSeek = false
	return iterator.iterOk
}

func (iterator *snapshotStorageIterator) Next() bool {
	iter := iterator.iter
	iterator.currentKey = nil
	if iterator.hasSeek {
		iterator.iterOk = iter.Prev()
		iterator.hasSeek = false
	}

	for iterator.iterOk {
		key := iter.Key()
		height := binary.BigEndian.Uint64(iter.Key()[len(key)-8:])
		if height <= iterator.snapshotHeight {
			iterator.currentKey = key

			iter.Seek(append(key[:len(key)-8], make([]byte, 8)...))
			iterator.hasSeek = true
			break
		}
		iterator.iterOk = iter.Prev()
	}

	return iterator.iterOk
}

func (iterator *snapshotStorageIterator) Key() []byte {
	return iterator.currentKey
}
func (iterator *snapshotStorageIterator) Value() []byte {
	return iterator.iter.Value()
}
func (iterator *snapshotStorageIterator) Error() error {
	err := iterator.iter.Error()
	if err != leveldb.ErrNotFound {
		return err
	}
	return nil
}

func (iterator *snapshotStorageIterator) Release() {
	iterator.iter.Release()
}
