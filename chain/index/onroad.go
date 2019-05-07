package chain_index

import (
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) Load(addrList []types.Address) (map[types.Address]map[types.Address][]ledger.HashHeight, error) {
	onRoadData := make(map[types.Address]map[types.Address][]ledger.HashHeight, len(addrList))
	for _, addr := range addrList {
		onRoadListMap := make(map[types.Address][]ledger.HashHeight)
		onRoadData[addr] = onRoadListMap

		iter := iDB.store.NewIterator(util.BytesPrefix(append([]byte{chain_utils.OnRoadKeyPrefix}, addr.Bytes()...)))
		for iter.Next() {
			key := iter.Key()
			blockHashBytes := key[len(key)-types.HashSize:]

			blockHash, err := types.BytesToHash(blockHashBytes)
			if err != nil {
				return nil, err
			}

			fromAddr, height, err := iDB.GetAddrHeightByHash(&blockHash)

			if err != nil {
				return nil, err
			}

			onRoadListMap[*fromAddr] = append(onRoadListMap[*fromAddr], ledger.HashHeight{
				Hash:   blockHash,
				Height: height,
			})
		}

		err := iter.Error()
		iter.Release()

		if err != nil {
			return nil, err
		}

	}
	return onRoadData, nil
}

func (iDB *IndexDB) LoadAllHash() (map[types.Address][]types.Hash, error) {
	onRoadListMap := make(map[types.Address][]types.Hash)

	iter := iDB.store.NewIterator(util.BytesPrefix([]byte{chain_utils.OnRoadKeyPrefix}))
	for iter.Next() {
		key := iter.Key()
		addrBytes := key[1 : len(key)-types.HashSize]
		addr, err := types.BytesToAddress(addrBytes)
		if err != nil {
			return nil, err
		}
		blockHashBytes := key[len(key)-types.HashSize:]
		blockHash, err := types.BytesToHash(blockHashBytes)
		if err != nil {
			return nil, err
		}
		_, ok := onRoadListMap[addr]
		if !ok {
			onRoadListMap[addr] = make([]types.Hash, 0)
		}
		onRoadListMap[addr] = append(onRoadListMap[addr], blockHash)
	}

	err := iter.Error()
	iter.Release()
	if err != nil {
		return nil, err
	}

	return onRoadListMap, nil
}

func (iDB *IndexDB) GetOnRoadHashList(addr types.Address, pageNum, pageSize int) ([]types.Hash, error) {
	index := 0
	hashList := make([]types.Hash, 0, pageSize)
	iter := iDB.store.NewIterator(util.BytesPrefix(append([]byte{chain_utils.OnRoadKeyPrefix}, addr.Bytes()...)))
	for iter.Next() {
		if index >= pageSize*pageNum {
			if index >= pageSize*(pageNum+1) {
				break
			}

			key := iter.Key()
			blockHashBytes := key[len(key)-types.HashSize:]
			blockHash, err := types.BytesToHash(blockHashBytes)
			if err != nil {
				return nil, err
			}
			hashList = append(hashList, blockHash)
			index++
		}
	}

	err := iter.Error()
	iter.Release()

	if err != nil {
		return nil, err
	}
	return hashList, nil
}

func (iDB *IndexDB) insertOnRoad(batch interfaces.Batch, toAddr types.Address, blockHash types.Hash) {
	batch.Put(chain_utils.CreateOnRoadKey(toAddr, blockHash), []byte{})

}

func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, toAddr types.Address, blockHash types.Hash) {
	batch.Delete(chain_utils.CreateOnRoadKey(toAddr, blockHash))
}
