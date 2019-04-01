package chain_state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

func (sDB *StateDB) NewStorageIterator(addr *types.Address, prefix []byte) interfaces.StorageIterator {
	slice := util.BytesPrefix(chain_utils.CreateStorageValueKeyPrefix(addr, prefix))
	return newStateStorageIterator(sDB.store.NewIterator(slice))
}

func (sDB *StateDB) NewSnapshotStorageIteratorByHeight(snapshotHeight uint64, addr *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	if snapshotHeight == sDB.chain.GetLatestSnapshotBlock().Height {
		return newStateStorageIterator(sDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateStorageValueKeyPrefix(addr, prefix)))), nil
	}

	storeIterator := sDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateHistoryStorageValueKeyPrefix(addr, prefix)))
	iterator := newStateStorageIterator(newSnapshotStorageIterator(storeIterator, snapshotHeight))

	return iterator, nil
}

func (sDB *StateDB) NewSnapshotStorageIterator(snapshotHash types.Hash, addr types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	if snapshotHash == sDB.chain.GetLatestSnapshotBlock().Hash {
		return newStateStorageIterator(sDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateStorageValueKeyPrefix(&addr, prefix)))), nil
	}

	height, err := sDB.chain.GetSnapshotHeightByHash(snapshotHash)
	if err != nil {
		sErr := errors.New(fmt.Sprintf("sDB.chain.GetSnapshotHeightByHash failed, hash is %s. Error: %s", snapshotHash, err))
		return nil, sErr
	}

	if height <= 0 {
		return nil, nil
	}

	storeIterator := sDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateHistoryStorageValueKeyPrefix(&addr, prefix)))

	iterator := newStateStorageIterator(newSnapshotStorageIterator(storeIterator, height))
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

func (iterator *stateStorageIterator) Seek(key []byte) bool {
	return iterator.iter.Seek(key)
}

func (iterator *stateStorageIterator) Next() bool {
	return iterator.iter.Next()
}

func (iterator *stateStorageIterator) Key() []byte {
	key := iterator.iter.Key()

	if len(key) <= 0 {
		return nil
	}

	keySize := key[1+types.AddressSize+types.HashSize]
	return key[1+types.AddressSize : 1+types.AddressSize+keySize]
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
	iter   interfaces.StorageIterator
	iterOk bool

	snapshotHeight uint64

	lastKey []byte
}

func newSnapshotStorageIterator(iter interfaces.StorageIterator, height uint64) interfaces.StorageIterator {
	sIterator := &snapshotStorageIterator{
		iter:           iter,
		snapshotHeight: height,
		iterOk:         true,
	}

	return sIterator
}

func (sIterator *snapshotStorageIterator) Last() bool {
	iter := sIterator.iter
	sIterator.iterOk = iter.Last()

	if sIterator.iterOk {
		sIterator.lastKey = iter.Key()
	}
	return sIterator.iterOk
}

func (sIterator *snapshotStorageIterator) Prev() bool {
	return sIterator.step(false)
}

func (sIterator *snapshotStorageIterator) Next() bool {
	return sIterator.step(true)
}
func (sIterator *snapshotStorageIterator) Seek(key []byte) bool {
	return sIterator.Seek(key)
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

		key := make([]byte, len(iter.Key()))
		// important
		copy(key, iter.Key())

		sIterator.lastKey = key

		sIterator.iterOk = iter.Seek(append(key[:len(key)-8], chain_utils.Uint64ToBytes(sIterator.snapshotHeight)...))

		if sIterator.iterOk {
			seekKey := iter.Key()

			if bytes.Equal(seekKey[:len(seekKey)-8], key[:len(key)-8]) &&
				binary.BigEndian.Uint64(seekKey[len(seekKey)-8:]) <= sIterator.snapshotHeight {
				break
			}

			if iter.Prev() {
				prevKey := iter.Key()

				if bytes.Equal(prevKey[:len(prevKey)-8], key[:len(key)-8]) {
					break
				}

			}
		} else if isNext {
			if sIterator.iterOk = iter.Last(); sIterator.iterOk {
				lastKey := iter.Key()
				if bytes.Equal(lastKey[:len(lastKey)-8], key[:len(key)-8]) {
					break
				}
			}
		}
	}

	return sIterator.iterOk
}
