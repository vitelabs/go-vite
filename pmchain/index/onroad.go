package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
)

func (iDB *IndexDB) HasOnRoadBlocks(address *types.Address) (bool, error) {
	accountId, err := iDB.GetAccountId(address)
	if err != nil {
		return false, err
	}

	key := chain_dbutils.CreateOnRoadPrefixKey(accountId)

	return iDB.hasValueByPrefix(key)
}

// TODO
func (iDB *IndexDB) GetOnRoadBlocksHashList(address *types.Address, pageNum, countPerPage int) ([]*types.Hash, error) {

	return nil, nil
}

func (iDB *IndexDB) newOnRoadId(blockHash *types.Hash) uint64 {
	return iDB.chain.GetLatestSnapshotBlock().Height
}

func (iDB *IndexDB) insertOnRoad(blockHash *types.Hash, toAccountId, sendAccountId, sendHeight uint64) error {
	onRoadId := iDB.newOnRoadId(blockHash)

	key := chain_dbutils.CreateOnRoadKey(toAccountId, onRoadId)
	value := chain_dbutils.SerializeAccountIdHeight(sendAccountId, sendHeight)

	reverseKey := chain_dbutils.CreateOnRoadReverseKey(value)

	if err := iDB.memDb.Append(blockHash, key, value); err != nil {
		return err
	}
	iDB.memDb.Put(blockHash, reverseKey, key)
	return nil
}

// TODO
func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, sendAccountId uint64, sendHeight uint64) error {
	key := chain_dbutils.CreateOnRoadReverseKey(chain_dbutils.SerializeAccountIdHeight(sendAccountId, sendHeight))

	batch.Delete(key)

	return nil
}
