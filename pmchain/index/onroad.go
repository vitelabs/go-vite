package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/pmchain/utils"
)

func (iDB *IndexDB) HasOnRoadBlocks(address *types.Address) (bool, error) {
	accountId, err := iDB.GetAccountId(address)
	if err != nil {
		return false, err
	}

	key := chain_utils.CreateOnRoadPrefixKey(accountId)

	return iDB.hasValueByPrefix(key)
}

func (iDB *IndexDB) GetOnRoadBlocksHashList(address *types.Address, pageNum, countPerPage int) ([]*types.Hash, error) {
	accountId, err := iDB.GetAccountId(address)
	if err != nil {
		return nil, err
	}
	key := chain_utils.CreateOnRoadPrefixKey(accountId)
	iter := iDB.NewIterator(util.BytesPrefix(key))
	defer iter.Release()

	hashList := make([]*types.Hash, 0, countPerPage)

	startIndex := pageNum * countPerPage
	endIndex := (pageNum + 1) * countPerPage

	index := 0
	for iter.Next() {
		if index > endIndex {
			break
		}

		if index >= startIndex {
			result := chain_utils.DeserializeHashList(iter.Value())

			lackLen := countPerPage - len(hashList)

			hashList = append(hashList, result[:lackLen]...)

			index += len(result)
		} else {
			index++
		}

		if len(hashList) >= countPerPage {
			break
		}
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return hashList, nil
}

func (iDB *IndexDB) newOnRoadId(blockHash *types.Hash) uint64 {
	return iDB.chain.GetLatestSnapshotBlock().Height
}

func (iDB *IndexDB) insertOnRoad(blockHash *types.Hash, toAccountId uint64) error {
	onRoadId := iDB.newOnRoadId(blockHash)

	key := chain_utils.CreateOnRoadKey(toAccountId, onRoadId)
	value := blockHash.Bytes()

	reverseKey := chain_utils.CreateOnRoadReverseKey(value)

	if err := iDB.memDb.Append(blockHash, key, value); err != nil {
		return err
	}
	iDB.memDb.Put(blockHash, reverseKey, key)
	return nil
}

func (iDB *IndexDB) receiveOnRoad(blockHash *types.Hash, sendBlockHash *types.Hash) error {
	reverseKey := chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes())
	value, err := iDB.getValue(reverseKey)
	if err != nil {
		return err
	}

	iDB.memDb.Delete(blockHash, reverseKey)

	iDB.memDb.Delete(blockHash, value)

	return nil
}

func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, sendBlockHash *types.Hash) error {
	reverseKey := chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes())
	value, err := iDB.getValue(reverseKey)
	if err != nil {
		return err
	}
	if len(value) <= 0 {
		return nil
	}

	batch.Delete(reverseKey)
	batch.Delete(value)

	return nil
}
