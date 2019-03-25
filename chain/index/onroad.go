package chain_index

import (
	"errors"
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"sync/atomic"
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
	key := chain_utils.CreateOnRoadPrefixKey(address)

	iter := iDB.NewIterator(util.BytesPrefix(key))
	defer iter.Release()

	hashList := make([]*types.Hash, 0, countPerPage)

	startIndex := pageNum * countPerPage
	endIndex := (pageNum + 1) * countPerPage

	index := 0
	for iter.Next() && len(hashList) < countPerPage {
		if index > endIndex {
			break
		}

		if index >= startIndex {
			result, err := types.BytesToHash(iter.Value())
			if err != nil {
				return nil, err
			}

			hashList = append(hashList, &result)
		}
		index++
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return hashList, nil
}

func (iDB *IndexDB) insertOnRoad(sendBlockHash *types.Hash, addr *types.Address) {
	onRoadId := atomic.AddUint64(&iDB.latestOnRoadId, 1)
	key := chain_utils.CreateOnRoadKey(addr, onRoadId)
	value := sendBlockHash.Bytes()

	reverseKey := chain_utils.CreateOnRoadReverseKey(value)

	iDB.memDb.Put(sendBlockHash, key, value)
	iDB.memDb.Put(sendBlockHash, reverseKey, key)

}

func (iDB *IndexDB) receiveOnRoad(receiveBlockHash *types.Hash, sendBlockHash *types.Hash) error {
	reverseKey := chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes())
	key, err := iDB.getValue(reverseKey)
	if err != nil {
		return err
	}
	if len(key) <= 0 {
		return errors.New(fmt.Sprintf("onRoad block is not existed, receiveBlockHash is %s, sendBlockHash is %s",
			receiveBlockHash, sendBlockHash))
	}

	iDB.memDb.Delete(receiveBlockHash, reverseKey)

	iDB.memDb.Delete(receiveBlockHash, key)

	return nil
}

func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, sendBlockHash *types.Hash) error {
	reverseKey := chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes())
	value, err := iDB.getValue(reverseKey)
	if err != nil {
		return err
	}
	if len(value) <= 0 {
		return errors.New(fmt.Sprintf("onRoad block is not exsited, block hash is %s", sendBlockHash))
	}

	batch.Delete(reverseKey)
	batch.Delete(value)

	return nil
}
