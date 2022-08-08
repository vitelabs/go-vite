package chain_index

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	chain_utils "github.com/vitelabs/go-vite/v2/ledger/chain/utils"
)

func (iDB *IndexDB) LoadRange(addrList []types.Address, loadFn interfaces.LoadOnroadFn) error {
	for _, addr := range addrList {
		iter := iDB.store.NewIterator(util.BytesPrefix(append([]byte{chain_utils.OnRoadKeyPrefix}, addr.Bytes()...)))
		for iter.Next() {
			key := iter.Key()
			blockHashBytes := key[len(key)-types.HashSize:]

			blockHash, err := types.BytesToHash(blockHashBytes)
			if err != nil {
				return err
			}

			fromAddr, height, err := iDB.GetAddrHeightByHash(&blockHash)

			if err != nil {
				return err
			}

			if fromAddr == nil {
				iDB.log.Error(fmt.Sprintf("block hash is %s, fromAddr is %s, height is %d", blockHash, fromAddr, height), "method", "Load")
				continue
			}
			err = loadFn(*fromAddr, addr, ledger.HashHeight{
				Height: height,
				Hash:   blockHash,
			})

			if err != nil {
				return err
			}
		}

		err := iter.Error()
		iter.Release()

		if err != nil {
			return err
		}

	}
	return nil
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
	defer iter.Release()

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
		}
		index++
	}

	err := iter.Error()
	if err != nil {
		return nil, err
	}
	return hashList, nil
}

func (iDB *IndexDB) insertOnRoad(batch interfaces.Batch, toAddr types.Address, blockHash types.Hash) {
	batch.Put(chain_utils.CreateOnRoadKey(toAddr, blockHash).Bytes(), []byte{})

}

func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, toAddr types.Address, blockHash types.Hash) {
	batch.Delete(chain_utils.CreateOnRoadKey(toAddr, blockHash).Bytes())
}
