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
	return iDB.store.HasPrefix(chain_utils.CreateOnRoadPrefixKey(address))
}

func (iDB *IndexDB) GetOnRoadBlocksHashList(address *types.Address, pageNum, countPerPage int) ([]types.Hash, error) {
	key := chain_utils.CreateOnRoadPrefixKey(address)

	iter := iDB.store.NewIterator(util.BytesPrefix(key))
	defer iter.Release()

	hashList := make([]types.Hash, 0, countPerPage)

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

			hashList = append(hashList, result)
		}
		index++
	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

	return hashList, nil
}

func (iDB *IndexDB) insertOnRoad(batch interfaces.Batch, sendBlockHash types.Hash, toAddr types.Address) {
	onRoadId := atomic.AddUint64(&iDB.latestOnRoadId, 1)
	key := chain_utils.CreateOnRoadKey(&toAddr, onRoadId)
	value := sendBlockHash.Bytes()

	reverseKey := chain_utils.CreateOnRoadReverseKey(value)

	batch.Put(key, value)
	batch.Put(reverseKey, key)

}

func (iDB *IndexDB) receiveOnRoad(batch interfaces.Batch, sendBlockHash *types.Hash) error {

	reverseKey := chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes())
	key, err := iDB.store.Get(reverseKey)
	if err != nil {
		return err
	}

	if len(key) <= 0 {
		return errors.New(fmt.Sprintf("onRoad block is not existed,  sendBlockHash is %s",
			sendBlockHash))
	}

	batch.Delete(reverseKey)
	batch.Delete(key)

	return nil
}

func (iDB *IndexDB) deleteOnRoad(batch *leveldb.Batch, sendBlockHash types.Hash) error {
	reverseKey := chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes())
	value, err := iDB.store.Get(reverseKey)
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
