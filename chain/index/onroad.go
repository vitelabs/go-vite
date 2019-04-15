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

//var logger = log15.New("onroad", "test")

func (iDB *IndexDB) insertOnRoad(batch interfaces.Batch, sendBlockHash types.Hash, toAddr types.Address) error {
	value := sendBlockHash.Bytes()
	reverseKey := chain_utils.CreateOnRoadReverseKey(value)
	ok, err := iDB.store.Has(reverseKey)
	if err != nil {
		return err
	}
	var key []byte
	if ok {
		key, err = iDB.store.Get(reverseKey)
		if err != nil {
			return err
		}
	} else {
		if len(key) <= 0 {
			// new key
			onRoadId := atomic.AddUint64(&iDB.latestOnRoadId, 1)
			key = chain_utils.CreateOnRoadKey(&toAddr, onRoadId)
		}

		batch.Put(reverseKey, key)
	}

	batch.Put(key, value)

	// FOR DEBUG
	//logger.Debug(fmt.Sprintf("insert on road %s %d %d\n", sendBlockHash, key, reverseKey))
	return nil

}

func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, sendBlockHash types.Hash) error {
	onRoadKey, err := iDB.store.Get(chain_utils.CreateOnRoadReverseKey(sendBlockHash.Bytes()))
	if err != nil {
		return err
	}
	if len(onRoadKey) <= 0 {
		return nil
	}

	batch.Delete(onRoadKey)

	// FOR DEBUG
	//logger.Debug(fmt.Sprintf("delete on road %s %d %d\n", sendBlockHash, value, reverseKey))

	return nil
}
