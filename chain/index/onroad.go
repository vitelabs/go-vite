package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
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

// TEST
var onRoadMu sync.RWMutex

func (iDB *IndexDB) InitOnRoad() error {
	iDB.onRoadData = make(map[types.Address]map[types.Hash]struct{})
	data, err := iDB.chain.LoadOnRoad(types.DELEGATE_GID)
	if err != nil {
		return err
	}
	for toAddr, fromAddrSet := range data {
		hashSet := make(map[types.Hash]struct{})
		iDB.onRoadData[toAddr] = hashSet
		for _, hashHeightList := range fromAddrSet {
			for _, hashHeight := range hashHeightList {
				hashSet[hashHeight.Hash] = struct{}{}
			}

		}
	}
	return nil
}

func (iDB *IndexDB) insertOnRoad(batch interfaces.Batch, toAddr types.Address, block *ledger.AccountBlock) {
	batch.Put(chain_utils.CreateOnRoadKey(toAddr, block.Hash), []byte{})

	// fixme TEST
	isContract := false
	if block.BlockType == ledger.BlockTypeSendCreate {
		isContract = true
	} else {
		var err error
		isContract, err = iDB.chain.IsContractAccount(toAddr)
		if err != nil {
			panic(err)
		}
	}

	if !isContract {
		return
	}

	onRoadMu.Lock()
	defer onRoadMu.Unlock()

	fromBlockHashSet, ok := iDB.onRoadData[toAddr]
	if !ok {
		fromBlockHashSet = make(map[types.Hash]struct{})
		iDB.onRoadData[toAddr] = fromBlockHashSet
	}
	fromBlockHashSet[block.Hash] = struct{}{}
}

func (iDB *IndexDB) deleteOnRoad(batch interfaces.Batch, toAddr types.Address, blockHash types.Hash) {
	batch.Delete(chain_utils.CreateOnRoadKey(toAddr, blockHash))

	// fixme TEST
	onRoadMu.Lock()
	defer onRoadMu.Unlock()

	fromBlockHashSet, ok := iDB.onRoadData[toAddr]
	if !ok {
		return
	}
	delete(fromBlockHashSet, blockHash)

	if len(fromBlockHashSet) <= 0 {
		delete(iDB.onRoadData, toAddr)
	}
}

// fixme TEST
func (iDB *IndexDB) HasOnRoad(addr types.Address) (bool, error) {
	onRoadMu.RLock()
	defer onRoadMu.RUnlock()

	_, ok := iDB.onRoadData[addr]
	return ok, nil
}

// fixme TEST
func (iDB *IndexDB) GetOnRoad(addr types.Address, pageNum, num int) ([]types.Hash, error) {
	onRoadMu.RLock()
	defer onRoadMu.RUnlock()

	fromBlockHashSet, ok := iDB.onRoadData[addr]
	if !ok {
		return nil, nil
	}
	hashList := make([]types.Hash, 0, num)
	for fromBlockHash := range fromBlockHashSet {
		hashList = append(hashList, fromBlockHash)
		if len(hashList) >= num {
			break
		}
	}
	return hashList, nil
}
