package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/utils"
)

func (iDB *IndexDB) Rollback(endLocation *chain_block.Location) error {
	batch := new(leveldb.Batch)
	// rollback account blocks

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

			height := chain_utils.FixedBytesToUint64(heightIter.Key()[1+types.AddressSize:])

			hash, err := types.BytesToHash(valueBytes[:types.HashSize])
			if err != nil {
				heightIter.Release()
				return err
			}

			// rollback
			iDB.rollbackAccountBlock(batch, &addr, &ledger.HashHeight{
				Hash:   hash,
				Height: height,
			})

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

		height := chain_utils.FixedBytesToUint64(iter.Key()[1:])
		hash, err := types.BytesToHash(value[:types.HashSize])
		if err != nil {
			return err
		}

		// rollback
		iDB.rollbackSnapshotBlock(batch, &ledger.HashHeight{
			Hash:   hash,
			Height: height,
		})
	}

	if err := iter2.Error(); err != nil && err != leveldb.ErrNotFound {
		return err
	}

	iDB.setIndexDbLatestLocation(batch, endLocation)
	// write index db
	if err := iDB.store.Write(batch); err != nil {
		return err
	}
	return nil
}

func (iDB *IndexDB) rollbackSnapshotBlock(batch interfaces.Batch, hashHeight *ledger.HashHeight) {
	iDB.deleteSnapshotBlockHeight(batch, hashHeight.Height)
	iDB.deleteSnapshotBlockHash(batch, &hashHeight.Hash)
}

func (iDB *IndexDB) rollbackAccountBlock(batch interfaces.Batch, addr *types.Address, hashHeight *ledger.HashHeight) {
	iDB.deleteAccountBlockHash(batch, &hashHeight.Hash)

	iDB.deleteAccountBlockHeight(batch, addr, hashHeight.Height)

	iDB.deleteReceive(batch, &hashHeight.Hash)

	iDB.deleteOnRoad(batch, &hashHeight.Hash)

	iDB.deleteConfirmHeight(batch, &hashHeight.Hash)
}
