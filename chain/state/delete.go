package chain_state

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

// TODO
func (sDB *StateDB) Rollback(deletedSnapshotSegments []*ledger.SnapshotChunk) error {
	if len(deletedSnapshotSegments) <= 0 {
		return nil
	}

	onlyDeleteAbs := false
	if deletedSnapshotSegments[0].SnapshotBlock == nil {
		onlyDeleteAbs = true
	}

	batch := sDB.store.NewBatch()

	blocksCache := make(map[types.Hash]*ledger.AccountBlock)
	balanceCache := make(map[types.Address]map[types.TokenTypeId]*big.Int)

	latestHeight := sDB.chain.GetLatestSnapshotBlock().Height

	addrList := make(map[types.Address]struct{})

	rollbackStorageKeySet := make(map[types.Address]map[string]struct{})

	snapshotHeight := latestHeight
	for _, seg := range deletedSnapshotSegments {
		snapshotHeight += 1

		if len(seg.AccountBlocks) <= 0 {
			continue
		}

		if !onlyDeleteAbs {
			sDB.storageRedo.Rollback(snapshotHeight)
		}

		var err error
		kvLogMap, err := sDB.storageRedo.QueryLog(snapshotHeight)
		if err != nil {
			return err
		}

		deleteKeys := make(map[string]struct{})

		for _, accountBlock := range seg.AccountBlocks {
			if onlyDeleteAbs {
				sDB.storageRedo.RemoveLog(accountBlock.Hash)
			}
			blocksCache[accountBlock.Hash] = accountBlock
			for _, sendBlock := range accountBlock.SendBlockList {
				blocksCache[sendBlock.Hash] = sendBlock
			}

			addr := accountBlock.AccountAddress
			addrList[addr] = struct{}{}

			// record rollback keys
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
	if err := sDB.recoverStorage(batch, rollbackStorageKeySet, onlyDeleteAbs); err != nil {
		return err
	}

	sDB.store.Write(batch)
	return nil
}

func (sDB *StateDB) recoverStorage(batch *leveldb.Batch, rollbackStorageKeySet map[types.Address]map[string]struct{}, onlyDeleteAbs bool) error {
	latestHeight := sDB.chain.GetLatestSnapshotBlock().Height
	if onlyDeleteAbs {
		kvLogMap, err := sDB.storageRedo.QueryLog(latestHeight + 1)
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
					batch.Delete(chain_utils.CreateStorageValueKey(&addr, key))
				} else {
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
		if err := rlp.DecodeBytes(kvLog, kvList); err != nil {
			return nil, err
		}
		return kvList, nil
	}
	return nil, nil
}

//func (sDB *StateDB) RecoverUnconfirmed(accountBlocks []*ledger.AccountBlock) error {
//	batch := sDB.store.NewBatch()
//
//	for _, accountBlock := range accountBlocks {
//		// rollback storage redo key
//		batch.Delete(chain_utils.CreateStorageRedoKey(accountBlock.Hash))
//
//		// rollback balance
//		addr := accountBlock.AccountAddress
//		tokenId := accountBlock.TokenId
//
//		var sendBlock *ledger.AccountBlock
//
//		if accountBlock.IsReceiveBlock() {
//			sendBlock, err := sDB.chain.GetAccountBlockByHash(accountBlock.FromBlockHash)
//			if err != nil {
//				return err
//			}
//			tokenId = sendBlock.TokenId
//		}
//		balance, err := getBalance(addr, tokenId)
//		if err != nil {
//			return err
//		}
//		if accountBlock.IsReceiveBlock() {
//			balance.Add(balance, sendBlock.Amount)
//		} else {
//			balance.Sub(balance, accountBlock.Amount)
//
//		}
//		allBalanceMap[addr][tokenId] = balance
//
//		// delete history balance
//		if snapshotBlock != nil {
//			deleteKey[string(chain_utils.CreateHistoryBalanceKey(addr, tokenId, snapshotBlock.Height))] = struct{}{}
//		}
//
//		// delete code
//		if accountBlock.Height <= 1 {
//			batch.Delete(chain_utils.CreateCodeKey(accountBlock.AccountAddress))
//		}
//
//		// delete contract meta
//		if accountBlock.BlockType == ledger.BlockTypeSendCreate {
//			batch.Delete(chain_utils.CreateContractMetaKey(accountBlock.AccountAddress))
//		}
//
//		// delete log hash
//		if accountBlock.LogHash != nil {
//			batch.Delete(chain_utils.CreateVmLogListKey(accountBlock.LogHash))
//		}
//
//		// delete call depth
//		if accountBlock.IsReceiveBlock() {
//			for _, sendBlock := range accountBlock.SendBlockList {
//				batch.Delete(chain_utils.CreateCallDepthKey(&sendBlock.Hash))
//			}
//		}
//	}
//	sDB.store.Write(batch)
//
//}
