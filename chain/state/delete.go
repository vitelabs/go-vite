package chain_state

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

func (sDB *StateDB) RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk, newUnconfirmedBlocks []*ledger.AccountBlock) error {
	sDB.disableCache()
	defer sDB.enableCache()

	batch := sDB.store.NewBatch()

	latestHeight := sDB.chain.GetLatestSnapshotBlock().Height
	newUnconfirmedLog, hasRedo, err := sDB.redo.QueryLog(latestHeight + 1)

	if err != nil {
		return errors.New(fmt.Sprintf("1. sDB.redo.QueryLog failed, height is %d. Error: %s", latestHeight+1, err.Error()))
	}

	snapshotHeight := latestHeight

	hasBuiltInContract := true

	if hasRedo {
		rollbackKeySet := make(map[types.Address]map[string]struct{})
		rollbackTokenSet := make(map[types.Address]map[types.TokenTypeId]struct{})

		for index, seg := range deletedSnapshotSegments {
			snapshotHeight += 1

			for _, accountBlock := range seg.AccountBlocks {
				if !hasBuiltInContract && types.IsBuiltinContractAddr(accountBlock.AccountAddress) {
					hasBuiltInContract = true
				}
			}
			// query currentSnapshotLog
			var currentSnapshotLog map[types.Address][]LogItem
			if index <= 0 {
				currentSnapshotLog = newUnconfirmedLog
			} else {
				currentSnapshotLog, _, err = sDB.redo.QueryLog(snapshotHeight)
				if err != nil {
					return errors.New(fmt.Sprintf("2. sDB.redo.QueryLog failed, height is %d. Error: %s", snapshotHeight, err.Error()))
				}
			}

			// rollback snapshot data
			if err := sDB.rollbackByRedo(batch, seg.SnapshotBlock, currentSnapshotLog, rollbackKeySet, rollbackTokenSet); err != nil {
				return err
			}
		}

		// recover latest index
		if err := sDB.recoverLatestIndexToSnapshot(batch, latestHeight, rollbackKeySet, rollbackTokenSet); err != nil {
			return err
		}

	} else {
		addrMap := make(map[types.Address]struct{}) // for recover to latest snapshot

		var oldUnconfirmedLog map[types.Address][]LogItem

		for index, seg := range deletedSnapshotSegments {
			snapshotHeight += 1

			// get old unconfirmed Log
			if index == len(deletedSnapshotSegments)-1 && seg.SnapshotBlock == nil {
				if oldUnconfirmedLog, _, err = sDB.redo.QueryLog(latestHeight + uint64(len(deletedSnapshotSegments))); err != nil {
					return errors.New(fmt.Sprintf("3. sDB.redo.QueryLog failed, height is %d. Error: %s", latestHeight+uint64(len(deletedSnapshotSegments)), err.Error()))
				}
			}

			for _, accountBlock := range seg.AccountBlocks {
				addrMap[accountBlock.AccountAddress] = struct{}{}

				if !hasBuiltInContract && types.IsBuiltinContractAddr(accountBlock.AccountAddress) {
					hasBuiltInContract = true
				}

				// rollback code, contract meta, log hash, call depth
				sDB.rollbackAccountBlock(batch, accountBlock)
			}

		}

		// recover latest and history index
		if err := sDB.recoverToSnapshot(batch, latestHeight, oldUnconfirmedLog, addrMap); err != nil {
			return err
		}

	}

	// commit
	sDB.store.RollbackSnapshot(batch)

	// rollback redo
	sDB.redo.Rollback(deletedSnapshotSegments)

	// recover redo
	sDB.redo.SetCurrentSnapshot(latestHeight+1, newUnconfirmedLog)

	// recover cache
	if hasBuiltInContract {
		if err := sDB.initSnapshotValueCache(); err != nil {
			return err
		}
	}

	// recover unconfirmed when has newUnconfirmedBlocks
	for _, block := range newUnconfirmedBlocks {
		redoLogList := newUnconfirmedLog[block.AccountAddress]

		startHeight := redoLogList[0].Height
		targetIndex := block.Height - startHeight

		sDB.WriteByRedo(block.Hash, block.AccountAddress, redoLogList[targetIndex])
	}

	// compact snapshot value
	//sDB.compactHistoryStorage()
	return nil
}

func (sDB *StateDB) RollbackAccountBlocks(accountBlocks []*ledger.AccountBlock) error {
	batch := sDB.store.NewBatch()

	latestHeight := sDB.chain.GetLatestSnapshotBlock().Height

	unconfirmedLog, hasRedo, err := sDB.redo.QueryLog(latestHeight + 1)
	if err != nil {
		return err
	}
	if !hasRedo {
		return errors.New("hasRedo is false")
	}

	rollbackKeySet := make(map[types.Address]map[string]struct{})
	rollbackTokenSet := make(map[types.Address]map[types.TokenTypeId]struct{})

	rollbackLogMap := make(map[types.Address][]LogItem)
	addrMap := make(map[types.Address]struct{}) // for recover to latest snapshot

	for _, accountBlock := range accountBlocks {
		if _, ok := rollbackLogMap[accountBlock.AccountAddress]; ok {
			continue
		}
		addrMap[accountBlock.AccountAddress] = struct{}{}

		deletedLogList := deleteRedoLog(unconfirmedLog, accountBlock.AccountAddress, accountBlock.Height)
		// set deleted
		rollbackLogMap[accountBlock.AccountAddress] = deletedLogList

	}

	// delete latest index
	if err := sDB.rollbackByRedo(batch, nil, rollbackLogMap, rollbackKeySet, rollbackTokenSet); err != nil {
		return err
	}

	// recover some latest
	sDB.recoverLatestIndexByRedo(batch, addrMap, unconfirmedLog, rollbackKeySet, rollbackTokenSet)

	// recover other latest
	if err := sDB.recoverLatestIndexToSnapshot(batch, latestHeight, rollbackKeySet, rollbackTokenSet); err != nil {
		return err
	}

	// set redo log
	sDB.redo.SetCurrentSnapshot(latestHeight+1, unconfirmedLog)

	// write
	sDB.store.RollbackAccountBlocks(batch, accountBlocks)
	return nil
}

func (sDB *StateDB) rollbackByRedo(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, redoLogMap map[types.Address][]LogItem,
	rollbackKeySet map[types.Address]map[string]struct{}, rollbackTokenSet map[types.Address]map[types.TokenTypeId]struct{}) error {
	// rollback history storage key value and record rollback keys
	var resetKeyTemplate []byte
	var resetBalanceTemplate []byte
	isHistory := true
	if snapshotBlock == nil {
		isHistory = false

		resetKeyTemplate = chain_utils.CreateStorageValueKey(&types.Address{}, []byte{})
		resetBalanceTemplate = chain_utils.CreateBalanceKey(types.Address{}, types.TokenTypeId{})

	} else {
		resetKeyTemplate = chain_utils.CreateHistoryStorageValueKey(&types.Address{}, []byte{}, snapshotBlock.Height)
		resetBalanceTemplate = chain_utils.CreateHistoryBalanceKey(types.Address{}, types.TokenTypeId{}, snapshotBlock.Height)
	}

	for addr, redoLogList := range redoLogMap {
		// copy addr
		copy(resetKeyTemplate[1:1+types.AddressSize], addr.Bytes())
		copy(resetBalanceTemplate[1:1+types.AddressSize], addr.Bytes())

		// init keySet
		keySet, _ := rollbackKeySet[addr]

		if keySet == nil {
			keySet = make(map[string]struct{})
		}

		// init tokenSet
		tokenSet, _ := rollbackTokenSet[addr]
		if tokenSet == nil {
			tokenSet = make(map[types.TokenTypeId]struct{})
		}

		for _, redoLog := range redoLogList {
			// delete snapshot storage
			for _, kv := range redoLog.Storage {

				keySet[string(kv[0])] = struct{}{}

				// copy key
				copy(resetKeyTemplate[1+types.AddressSize:], common.RightPadBytes(kv[0], 32))
				resetKeyTemplate[1+types.AddressSize+types.HashSize] = byte(len(kv[0]))

				if isHistory {
					sDB.deleteHistoryKey(batch, resetKeyTemplate)
				} else {
					batch.Delete(resetKeyTemplate)
				}
			}

			// delete snapshot balance
			for tokenTypeId := range redoLog.BalanceMap {

				tokenSet[tokenTypeId] = struct{}{}

				copy(resetBalanceTemplate[1+types.AddressSize:], tokenTypeId.Bytes())

				if isHistory {
					batch.Delete(resetBalanceTemplate)
				} else {
					sDB.deleteBalance(batch, resetBalanceTemplate)
				}
			}

			// delete contract meta
			for contractAddr := range redoLog.ContractMeta {
				sDB.deleteContractMeta(batch, chain_utils.CreateContractMetaKey(contractAddr))
			}

			// delete code
			if len(redoLog.Code) > 0 {
				batch.Delete(chain_utils.CreateCodeKey(addr))
			}

			// delete vm log list
			for logHash := range redoLog.VmLogList {

				batch.Delete(chain_utils.CreateVmLogListKey(&logHash))

			}

			// delete call depth
			for sendBlockHash := range redoLog.CallDepth {

				batch.Delete(chain_utils.CreateCallDepthKey(sendBlockHash))

			}
		}

		if len(keySet) > 0 {
			rollbackKeySet[addr] = keySet
		}
		if len(tokenSet) > 0 {
			rollbackTokenSet[addr] = tokenSet
		}

	}

	return nil
}

// only recover latest index
func (sDB *StateDB) recoverLatestIndexToSnapshot(batch *leveldb.Batch, latestSnapshotHeight uint64, keySetMap map[types.Address]map[string]struct{}, tokenSetMap map[types.Address]map[types.TokenTypeId]struct{}) error {

	// recover kv latest index
	for addr, keySet := range keySetMap {
		storage := NewStorageDatabase(sDB, latestSnapshotHeight, addr)

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

	// recover balance latest index

	iter := sDB.store.NewIterator(util.BytesPrefix([]byte{chain_utils.BalanceHistoryKeyPrefix}))
	defer iter.Release()

	balanceTemplateKey := chain_utils.CreateBalanceKey(types.Address{}, types.TokenTypeId{})

	seekKey := chain_utils.CreateHistoryBalanceKey(types.Address{}, types.TokenTypeId{}, latestSnapshotHeight+1)

	for addr, tokenSet := range tokenSetMap {
		// copy addr
		addrBytes := addr.Bytes()
		copy(seekKey[1:1+types.AddressSize], addrBytes)
		copy(balanceTemplateKey[1:1+types.AddressSize], addrBytes)

		for tokenId := range tokenSet {
			// copy tokenIdBytes
			tokenIdBytes := tokenId.Bytes()

			copy(seekKey[1+types.AddressSize:], tokenIdBytes)
			copy(balanceTemplateKey[1+types.AddressSize:], tokenIdBytes)

			iter.Seek(seekKey)

			if iter.Prev() && bytes.HasPrefix(iter.Key(), seekKey[:len(seekKey)-8]) {
				// set latest

				sDB.writeBalance(batch, balanceTemplateKey, iter.Value())
			} else {
				sDB.deleteBalance(batch, balanceTemplateKey)

			}
		}

	}

	return nil
}

func (sDB *StateDB) recoverLatestIndexByRedo(batch *leveldb.Batch, addrMap map[types.Address]struct{}, redoLogMap map[types.Address][]LogItem, rollbackKeySet map[types.Address]map[string]struct{}, rollbackTokenSet map[types.Address]map[types.TokenTypeId]struct{}) {
	for addr := range addrMap {
		redoLogList := redoLogMap[addr]
		for _, redoLog := range redoLogList {
			for _, kv := range redoLog.Storage {
				batch.Put(chain_utils.CreateStorageValueKey(&addr, kv[0]), kv[1])
				if _, ok := rollbackKeySet[addr]; ok {
					delete(rollbackKeySet[addr], string(kv[0]))
				}
			}

			for tokenId, balance := range redoLog.BalanceMap {
				sDB.writeBalance(batch, chain_utils.CreateBalanceKey(addr, tokenId), balance.Bytes())

				if _, ok := rollbackTokenSet[addr]; ok {
					delete(rollbackTokenSet[addr], tokenId)
				}
			}
		}
	}

}

func (sDB *StateDB) rollbackAccountBlock(batch *leveldb.Batch, accountBlock *ledger.AccountBlock) {
	// delete code
	if accountBlock.Height <= 1 {
		batch.Delete(chain_utils.CreateCodeKey(accountBlock.AccountAddress))
	}

	// delete contract meta
	if accountBlock.BlockType == ledger.BlockTypeSendCreate {
		sDB.deleteContractMeta(batch, chain_utils.CreateContractMetaKey(accountBlock.ToAddress))
	}

	// delete log hash
	if accountBlock.LogHash != nil {
		batch.Delete(chain_utils.CreateVmLogListKey(accountBlock.LogHash))
	}

	// delete call depth && contract meta
	for _, sendBlock := range accountBlock.SendBlockList {
		batch.Delete(chain_utils.CreateCallDepthKey(sendBlock.Hash))

		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			sDB.deleteContractMeta(batch, chain_utils.CreateContractMetaKey(sendBlock.ToAddress))
		}
	}

}

func (sDB *StateDB) recoverToSnapshot(batch *leveldb.Batch, snapshotHeight uint64, unconfirmedLog map[types.Address][]LogItem, addrMap map[types.Address]struct{}) error {
	keySetMap, tokenSetMap, err := parseRedoLog(unconfirmedLog)
	if err != nil {
		return err
	}

	for addr := range addrMap {
		// recover key value
		if err := sDB.recoverStorageToSnapshot(batch, snapshotHeight, addr, keySetMap[addr]); err != nil {
			return err
		}

		// recover balance
		if err := sDB.recoverBalanceToSnapshot(batch, snapshotHeight, addr, tokenSetMap[addr]); err != nil {
			return err
		}

	}
	return nil
}

func (sDB *StateDB) recoverStorageToSnapshot(batch *leveldb.Batch, height uint64, addr types.Address, keySet map[string][]byte) error {
	iter := sDB.store.NewIterator(util.BytesPrefix(append([]byte{chain_utils.StorageHistoryKeyPrefix}, addr.Bytes()...)))
	defer iter.Release()

	storageTemplateKey := chain_utils.CreateStorageValueKey(&addr, []byte{})

	seekTemplateKey := chain_utils.CreateHistoryStorageValueKey(&addr, []byte{}, height+1)
	iterOk := iter.Next()

	const keyStartIndex = 1 + types.AddressSize
	const keyEndIndex = 1 + types.AddressSize + types.HashSize + 1

	for iterOk {
		// copy key
		storageKeyBytes := iter.Key()[keyStartIndex:keyEndIndex]
		copy(seekTemplateKey[keyStartIndex:keyEndIndex], storageKeyBytes)
		copy(storageTemplateKey[keyStartIndex:keyEndIndex], storageKeyBytes)

		// real key
		realKey := string(storageKeyBytes[:storageKeyBytes[len(storageKeyBytes)-1]])

		// delete real key
		delete(keySet, realKey)

		// seek
		iter.Seek(seekTemplateKey)

		if iter.Prev() && bytes.Equal(seekTemplateKey[keyStartIndex:keyEndIndex], iter.Key()[keyStartIndex:keyEndIndex]) {
			batch.Put(storageTemplateKey, iter.Value())
		} else {
			batch.Delete(storageTemplateKey)
		}

		for {
			iterOk = iter.Next()
			if !iterOk {
				break
			}

			key := iter.Key()
			if bytes.Equal(seekTemplateKey[keyStartIndex:keyEndIndex], key[keyStartIndex:keyEndIndex]) {
				// delete history
				sDB.deleteHistoryKey(batch, key)
			} else {
				break
			}

		}

	}
	if err := iter.Error(); err != nil {
		return err
	}

	// delete unconfirmed balance
	for keyStr := range keySet {
		key := []byte(keyStr)
		copy(storageTemplateKey[keyStartIndex:], common.RightPadBytes(key, types.HashSize))
		storageTemplateKey[1+types.AddressSize+types.HashSize] = byte(len(key))
		batch.Delete(storageTemplateKey)
	}

	return nil
}

func (sDB *StateDB) recoverBalanceToSnapshot(batch *leveldb.Batch, height uint64, addr types.Address, tokenSet map[types.TokenTypeId]*big.Int) error {
	iter := sDB.store.NewIterator(util.BytesPrefix(append([]byte{chain_utils.BalanceHistoryKeyPrefix}, addr.Bytes()...)))
	defer iter.Release()

	balanceTemplateKey := chain_utils.CreateBalanceKey(addr, types.TokenTypeId{})

	seekTemplateKey := chain_utils.CreateHistoryBalanceKey(addr, types.TokenTypeId{}, height+1)

	const tokenIdStartIndex = 1 + types.AddressSize
	const tokenIdEndIndex = 1 + types.AddressSize + types.TokenTypeIdSize
	iterOk := iter.Next()

	for iterOk {
		// get tokenIdBytes
		tokenIdBytes := iter.Key()[tokenIdStartIndex:tokenIdEndIndex]

		// copy tokenIdBytes
		copy(seekTemplateKey[tokenIdStartIndex:tokenIdEndIndex], tokenIdBytes)
		copy(balanceTemplateKey[tokenIdStartIndex:tokenIdEndIndex], tokenIdBytes)

		// parse to tokenId
		tokenId, err := types.BytesToTokenTypeId(tokenIdBytes)
		if err != nil {
			return err
		}
		// delete token set
		delete(tokenSet, tokenId)
		// seek
		iter.Seek(seekTemplateKey)

		if iter.Prev() && bytes.Equal(seekTemplateKey[tokenIdStartIndex:tokenIdEndIndex], iter.Key()[tokenIdStartIndex:tokenIdEndIndex]) {
			//fmt.Printf("recoverToLatestBalance %d. %s. %s: %s\n", height, addr, tokenId, new(big.Int).SetBytes(iter.Value()))
			sDB.writeBalance(batch, balanceTemplateKey, iter.Value())
		} else {
			sDB.deleteBalance(batch, balanceTemplateKey)
		}
		for {
			iterOk = iter.Next()
			if !iterOk {
				break
			}
			key := iter.Key()
			if bytes.Equal(seekTemplateKey[tokenIdStartIndex:tokenIdEndIndex], key[tokenIdStartIndex:tokenIdEndIndex]) {
				// delete history
				batch.Delete(key)
			} else {
				break
			}

		}
	}

	if err := iter.Error(); err != nil {
		return err
	}

	// delete unconfirmed balance
	for tokenId := range tokenSet {
		//fmt.Printf("recoverToLatestBalance  %d. %s. delete %s\n", height, addr, tokenId)
		copy(balanceTemplateKey[tokenIdStartIndex:tokenIdEndIndex], tokenId.Bytes())
		sDB.deleteBalance(batch, balanceTemplateKey)
	}

	return nil
}

func (sDB *StateDB) compactHistoryStorage() {
	//startAddrBytes []byte, endAddrBytes []byte
	// compact storage history key prefix
	//for _, addr := range addrList {
	if err := sDB.store.CompactRange(*util.BytesPrefix([]byte{chain_utils.StorageHistoryKeyPrefix})); err != nil {
		sDB.log.Error("CompactRange error, key prefix is %d. Error: %s", chain_utils.StorageHistoryKeyPrefix, err)
	}
	//}

}

func (sDB *StateDB) deleteContractMeta(batch interfaces.Batch, key []byte) {
	batch.Delete(key)

	sDB.cache.Delete(contractAddrPrefix + string(key))
}

func (sDB *StateDB) deleteBalance(batch interfaces.Batch, key []byte) {
	batch.Delete(key)

	sDB.cache.Delete(balancePrefix + string(key))
}

func (sDB *StateDB) deleteHistoryKey(batch interfaces.Batch, key []byte) {
	batch.Delete(key)

	addrBytes := key[1 : 1+types.AddressSize]
	addr, err := types.BytesToAddress(addrBytes)
	if err != nil {
		panic(err)
	}
	if types.IsBuiltinContractAddr(addr) {
		sDB.cache.Delete(snapshotValuePrefix + string(addrBytes) + string(sDB.parseStorageKey(key)))
	}
}
