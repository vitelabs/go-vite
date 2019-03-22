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
	iter := iDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateAccountIdPrefixKey()))
	defer iter.Release()

	batch := new(leveldb.Batch)
	for iter.Next() {
		value := iter.Value()

		addr, err := types.BytesToAddress(value)
		if err != nil {
			return err
		}

		heightIter := iDB.store.NewIterator(util.BytesPrefix(chain_utils.CreateAccountBlockHeightPrefixKey(&addr)))
		iterOk := heightIter.Last()
		for iterOk {
			height := chain_utils.FixedBytesToUint64(iter.Key()[1+types.AddressSize:])
			locationBytes := heightIter.Value()
			location := chain_utils.DeserializeLocation(locationBytes)

			if location.Compare(endLocation) >= 0 {
				// rollback
				iDB.rollbackAccountBlock(batch, &addr, &ledger.HashHeight{
					Height: height,
				})
			} else {
				break
			}

			iterOk = heightIter.Prev()
		}
		if err := heightIter.Error(); err != nil && err != leveldb.ErrNotFound {
			heightIter.Release()
			return err
		}

		heightIter.Release()
		//iDB.getValue()
	}
	return nil
}

func (iDB *IndexDB) rollbackAccountBlock(batch interfaces.Batch, addr *types.Address, hashHeight *ledger.HashHeight) {
	iDB.deleteAccountBlockHash(batch, &hashHeight.Hash)

	iDB.deleteAccountBlockHeight(batch, addr, hashHeight.Height)

	iDB.deleteReceive(batch, &hashHeight.Hash)

	iDB.deleteOnRoad(batch, &hashHeight.Hash)

	iDB.deleteConfirmHeight(batch, &hashHeight.Hash)
}
