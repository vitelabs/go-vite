package chain_index

import (
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
	for iter.Next() && index < endIndex {

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

func (iDB *IndexDB) insertOnRoad(batch interfaces.Batch, sendBlockHash types.Hash, toAddr types.Address) error {

	value := sendBlockHash.Bytes()
	reverseKey := chain_utils.CreateOnRoadReverseKey(value)
	if ok, err := iDB.store.Has(reverseKey); err != nil {
		return err
	} else if ok {
		return nil
	}

	onRoadId := atomic.AddUint64(&iDB.latestOnRoadId, 1)
	key := chain_utils.CreateOnRoadKey(&toAddr, onRoadId)

	batch.Put(key, value)
	batch.Put(reverseKey, key)

	// FOR DEBUG
	//fmt.Printf("insert on road %s %d %d\n", sendBlockHash, key, reverseKey)
	return nil

}

func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, sendBlockHash types.Hash) error {
	reverseKey := chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes())
	value, err := iDB.store.Get(reverseKey)
	if err != nil {
		return err
	}
	if len(value) <= 0 {
		return nil
	}

	batch.Delete(reverseKey)
	batch.Delete(value)

	// FOR DEBUG
	//fmt.Printf("delete on road %s %d %d\n", sendBlockHash, value, reverseKey)

	return nil
}
