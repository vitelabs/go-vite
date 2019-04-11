package chain_state

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

func (sDB *StateDB) RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk) error {

	batch := sDB.store.NewBatch()

	if err := sDB.rollback(batch, deletedSnapshotSegments, false, false); err != nil {
		return err
	}
	sDB.store.RollbackSnapshot(batch)
	return nil
}

func (sDB *StateDB) RollbackAccountBlocks(accountBlocks []*ledger.AccountBlock, deleteAllUnconfirmed bool) error {
	batch := sDB.store.NewBatch()

	if err := sDB.rollback(batch, []*ledger.SnapshotChunk{{
		AccountBlocks: accountBlocks,
	}}, true, deleteAllUnconfirmed); err != nil {
		return err
	}
	sDB.store.RollbackAccountBlocks(batch, accountBlocks)
	return nil

}

func (sDB *StateDB) rollback(batch *leveldb.Batch, deletedSnapshotSegments []*ledger.SnapshotChunk, onlyDeleteAbs bool, deleteAllUnconfirmed bool) error {

	blocksCache := make(map[types.Hash]*ledger.AccountBlock)

	balanceCache := make(map[types.Address]map[types.TokenTypeId]*big.Int)

	rollbackStorageKeySet := make(map[types.Address]map[string]struct{})

	latestHeight := sDB.chain.GetLatestSnapshotBlock().Height

	snapshotHeight := latestHeight

	hasRedo := true

	addrMap := make(map[types.Address]struct{}) // for recover to latest snapshot

	for index, seg := range deletedSnapshotSegments {
		snapshotHeight += 1

		if index > 0 {
			sDB.storageRedo.Rollback(snapshotHeight)
		}

		if len(seg.AccountBlocks) <= 0 {
			continue
		}

		var err error
		var kvLogMap map[types.Hash][]byte
		kvLogMap, hasRedo, err = sDB.storageRedo.QueryLog(snapshotHeight)

		if err != nil {
			return err
		}

		deleteKeys := make(map[string]struct{})

		for _, accountBlock := range seg.AccountBlocks {

			// set blocks cache
			blocksCache[accountBlock.Hash] = accountBlock
			for _, sendBlock := range accountBlock.SendBlockList {
				blocksCache[sendBlock.Hash] = sendBlock
			}

			// set addr cache
			addr := accountBlock.AccountAddress
			addrMap[addr] = struct{}{}

			// record rollback keys
			if hasRedo && len(kvLogMap) > 0 {
				kvList, err := getKvList(kvLogMap, accountBlock.Hash)
				if err != nil {
					return err
				}

				if len(kvList) > 0 {
					keySet, _ := rollbackStorageKeySet[addr]
					if keySet == nil {
						keySet = make(map[string]struct{})
					}

					for _, kv := range kvList {
						keySet[string(kv[0])] = struct{}{}
						deleteKeys[string(chain_utils.CreateHistoryStorageValueKey(&addr, kv[0], snapshotHeight))] = struct{}{}
					}

					rollbackStorageKeySet[addr] = keySet
				}
			}

			// remove log
			if onlyDeleteAbs {
				sDB.storageRedo.RemoveLog(accountBlock.Hash)
			}
			// record balance
			// query info
			tokenIds, err := sDB.calculateBalance(accountBlock, balanceCache, blocksCache)
			if err != nil {
				return err
			}
			for tokenId := range tokenIds {
				deleteKeys[string(chain_utils.CreateHistoryBalanceKey(addr, tokenId, snapshotHeight))] = struct{}{}
			}

			// delete code
			if accountBlock.Height <= 1 {
				batch.Delete(chain_utils.CreateCodeKey(accountBlock.AccountAddress))
			}

			// delete contract meta
			if accountBlock.BlockType == ledger.BlockTypeSendCreate {
				batch.Delete(chain_utils.CreateContractMetaKey(accountBlock.AccountAddress))
			}

			// delete log hash
			if accountBlock.LogHash != nil {
				batch.Delete(chain_utils.CreateVmLogListKey(accountBlock.LogHash))
			}

			// delete call depth
			if accountBlock.IsReceiveBlock() {
				for _, sendBlock := range accountBlock.SendBlockList {
					batch.Delete(chain_utils.CreateCallDepthKey(&sendBlock.Hash))
				}
			}
		}

		// delete history storage key & history balance
		for key := range deleteKeys {
			batch.Delete([]byte(key))
		}
	}

	// reset balance index
	for addr, balanceMap := range balanceCache {
		for tokenTypeId, balance := range balanceMap {
			balanceBytes := balance.Bytes()

			batch.Put(chain_utils.CreateBalanceKey(addr, tokenTypeId), balanceBytes)

			if onlyDeleteAbs {
				batch.Put(chain_utils.CreateHistoryBalanceKey(addr, tokenTypeId, latestHeight+1), balanceBytes)
			}
		}
	}

	// reset storage index
	if hasRedo {
		if err := sDB.recoverStorage(batch, rollbackStorageKeySet, onlyDeleteAbs); err != nil {
			return err
		}
	} else {
		if err := sDB.recoverStorageToLatestSnapshot(batch, addrMap, onlyDeleteAbs, deleteAllUnconfirmed); err != nil {
			return err
		}
	}

	// reset redo log
	if !onlyDeleteAbs {
		sDB.storageRedo.ResetSnapshot(latestHeight + 1)
	}

	return nil
}

func (sDB *StateDB) recoverStorageToLatestSnapshot(batch *leveldb.Batch, addrList map[types.Address]struct{}, onlyDeleteAbs bool, deleteAllUnconfirmed bool) error {

	height := sDB.chain.GetLatestSnapshotBlock().Height + 1
	if onlyDeleteAbs && deleteAllUnconfirmed {
		height -= 1
	}

	for addr := range addrList {

		storage := NewStorageDatabase(sDB, helper.MaxUint64, addr)

		iter, err := storage.NewStorageIterator(nil)
		if err != nil {
			return err
		}

		for iter.Next() {
			key := iter.Key()

			// FOR DEBUG
			//fmt.Println()
			//fmt.Printf("%d. %s. %s: %v", height, addr, key, iter.Value())
			//fmt.Println()

			startKey := chain_utils.CreateHistoryStorageValueKey(&addr, key, 0)
			endKey := chain_utils.CreateHistoryStorageValueKey(&addr, key, helper.MaxUint64)

			iter2 := sDB.store.NewIterator(&util.Range{Start: startKey, Limit: endKey})
			iterOk := iter2.Last()

			hasResetLatestIndex := false
			for iterOk {
				iter2Key := iter2.Key()

				currentHeight := binary.BigEndian.Uint64(iter2Key[len(iter2Key)-8:])

				if currentHeight <= height {
					hasResetLatestIndex = true
					value := iter.Value()
					if len(value) <= 0 {
						//FOR DEBUG
						//fmt.Println()
						//fmt.Printf("%d. %s. delete latest: %s", height, addr, key)
						//fmt.Println()

						batch.Delete(chain_utils.CreateStorageValueKey(&addr, key))
					} else {

						// FOR DEBUG
						//fmt.Println()
						//fmt.Printf("%d. %s. set latest: %s, %+v", height, addr, key, value)
						//fmt.Println()

						batch.Put(chain_utils.CreateStorageValueKey(&addr, key), value)
					}
					break
				} else {
					// delete history key
					batch.Delete(iter2Key)
				}

				iterOk = iter2.Prev()

			}

			if err := iter2.Error(); err != nil {
				iter.Release()
				iter2.Release()
				return err
			}

			if !hasResetLatestIndex {
				// FOR DEBUG
				//fmt.Println()
				//fmt.Printf("%d. %s. delete latest: %s", height, addr, key)
				//fmt.Println()

				batch.Delete(chain_utils.CreateStorageValueKey(&addr, key))
			}

			iter2.Release()
		}
		if err := iter.Error(); err != nil {
			iter.Release()
			return err
		}
		iter.Release()
	}
	return nil
}

func (sDB *StateDB) recoverStorage(batch *leveldb.Batch, rollbackStorageKeySet map[types.Address]map[string]struct{}, onlyDeleteAbs bool) error {
	latestHeight := sDB.chain.GetLatestSnapshotBlock().Height
	if onlyDeleteAbs {
		kvLogMap, _, err := sDB.storageRedo.QueryLog(latestHeight + 1)
		if err != nil {
			return err
		}

		for addr, keySet := range rollbackStorageKeySet {
			storage := NewStorageDatabase(sDB, latestHeight, addr)

			unconfirmedKvMap := make(map[string][]byte)
			unconfirmedBlocks := sDB.chain.GetUnconfirmedBlocks(addr)
			for _, unconfirmedBlock := range unconfirmedBlocks {
				kvList, err := getKvList(kvLogMap, unconfirmedBlock.Hash)
				if err != nil {
					return err
				}

				for _, kv := range kvList {
					unconfirmedKvMap[string(kv[0])] = kv[1]
				}

			}

			for keyStr := range keySet {
				key := []byte(keyStr)
				value, ok := unconfirmedKvMap[keyStr]
				if !ok {
					var err error
					value, err = storage.GetValue(key)
					if err != nil {
						return err
					}
				} else {

					batch.Put(chain_utils.CreateHistoryStorageValueKey(&addr, key, latestHeight+1), value)

				}

				if len(value) <= 0 {
					batch.Delete(chain_utils.CreateStorageValueKey(&addr, key))
				} else {
					batch.Put(chain_utils.CreateStorageValueKey(&addr, key), value)
				}

			}
		}
	} else {
		for addr, keySet := range rollbackStorageKeySet {
			storage := NewStorageDatabase(sDB, latestHeight+1, addr)
			for keyStr := range keySet {
				key := []byte(keyStr)

				value, err := storage.GetValue(key)
				if err != nil {
					return err
				}

				if len(value) <= 0 {
					//FOR DEBUG
					//fmt.Println()
					//fmt.Printf("%d. %s. delete latest: %s", latestHeight+1, addr, key)
					//fmt.Println()

					batch.Delete(chain_utils.CreateStorageValueKey(&addr, key))
				} else {

					// FOR DEBUG
					//fmt.Println()
					//fmt.Printf("%d. %s. set latest: %s, %+v", latestHeight+1, addr, key, value)
					//fmt.Println()
					batch.Put(chain_utils.CreateStorageValueKey(&addr, key), value)
				}

			}
		}
	}

	return nil
}

func (sDB *StateDB) calculateBalance(accountBlock *ledger.AccountBlock, balanceCache map[types.Address]map[types.TokenTypeId]*big.Int, blocksCache map[types.Hash]*ledger.AccountBlock) (map[types.TokenTypeId]struct{}, error) {

	tokenIdMap := make(map[types.TokenTypeId]struct{})

	tokenId := accountBlock.TokenId
	amount := accountBlock.Amount

	if accountBlock.IsReceiveBlock() {
		sendBlock := blocksCache[accountBlock.FromBlockHash]
		if sendBlock == nil {
			var err error
			sendBlock, err = sDB.chain.GetAccountBlockByHash(accountBlock.FromBlockHash)
			if err != nil {
				return nil, err
			}
			if sendBlock == nil {
				return nil, errors.New(fmt.Sprintf("sendBlock is nil, receiveBlock is %+v\n", accountBlock))
			}
		}

		tokenId = sendBlock.TokenId
		amount = sendBlock.Amount
	}

	balanceMap, ok := balanceCache[accountBlock.AccountAddress]
	if !ok {
		balanceMap = make(map[types.TokenTypeId]*big.Int)
		balanceCache[accountBlock.AccountAddress] = balanceMap
	}

	balance, ok := balanceMap[tokenId]
	if !ok {
		var err error
		balance, err = sDB.GetBalance(accountBlock.AccountAddress, tokenId)
		if err != nil {
			return nil, err
		}
		balanceMap[tokenId] = balance
	}

	if accountBlock.IsReceiveBlock() {
		balance.Sub(balance, amount)
	} else {
		balance.Add(balance, accountBlock.Amount)
	}

	if accountBlock.Fee != nil {
		balance.Add(balance, accountBlock.Fee)
	}

	balanceCache[accountBlock.AccountAddress][tokenId] = balance

	tokenIdMap[tokenId] = struct{}{}
	for _, sendBlock := range accountBlock.SendBlockList {
		if tempMap, err := sDB.calculateBalance(sendBlock, balanceCache, blocksCache); err != nil {
			return nil, err
		} else {
			for tokenId := range tempMap {
				tokenIdMap[tokenId] = struct{}{}
			}
		}
	}

	return tokenIdMap, nil
}

func getKvList(kvLogMap map[types.Hash][]byte, blockHash types.Hash) ([][2][]byte, error) {
	if kvLog, ok := kvLogMap[blockHash]; ok {
		var kvList [][2][]byte
		if err := rlp.DecodeBytes(kvLog, &kvList); err != nil {
			return nil, err
		}
		return kvList, nil
	}
	return nil, nil
}
