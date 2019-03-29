package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
)

func (iDB *IndexDB) DeleteTo(endLocation *chain_file_manager.Location) error {

	iter := iDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateAccountIdPrefixKey()))
	defer iter.Release()

	for iter.Next() {
		value := iter.Value()

		addr, err := types.BytesToAddress(value)
		if err != nil {
			return err
		}

		heightIter := iDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateAccountBlockHeightPrefixKey(&addr)))

		iterOk := heightIter.Last()
		for iterOk {
			valueBytes := heightIter.Value()

			location := chain_utils.DeserializeLocation(valueBytes[types.HashSize:])
			if location.Compare(endLocation) < 0 {
				break
			}

			height := chain_utils.BytesToUint64(heightIter.Key()[1+types.AddressSize:])

			hash, err := types.BytesToHash(valueBytes[:types.HashSize])
			if err != nil {
				heightIter.Release()
				return err
			}

			// rollback
			iDB.deleteAccountBlock(addr, hash, height)

			iterOk = heightIter.Prev()
		}
		if err := heightIter.Error(); err != nil && err != leveldb.ErrNotFound {
			heightIter.Release()
			return err
		}

		heightIter.Release()
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}

	iter2 := iDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateAccountIdPrefixKey()))
	defer iter2.Release()

	for iter2.Next() {
		value := iter2.Value()

		location := chain_utils.DeserializeLocation(value[types.HashSize:])
		if location.Compare(endLocation) < 0 {
			break
		}

		height := chain_utils.BytesToUint64(iter.Key()[1:])
		hash, err := types.BytesToHash(value[:types.HashSize])
		if err != nil {
			return err
		}

		// rollback
		iDB.deleteSnapshotBlock(hash, height)
	}

	if err := iter2.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}

	return nil
}

func (iDB *IndexDB) deleteSnapshotBlock(hash types.Hash, height uint64) {
	iDB.deleteSnapshotBlockHeight(height)
	iDB.deleteSnapshotBlockHash(hash)
}

func (iDB *IndexDB) deleteAccountBlock(addr types.Address, hash types.Hash, height uint64) {
	iDB.deleteAccountBlockHash(hash)

	iDB.deleteAccountBlockHeight(addr, height)

	iDB.deleteReceive(hash)

	iDB.deleteOnRoad(hash)

	iDB.deleteConfirmHeight(addr, height)
}
