package chain_state

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

func (sDB *StateDB) RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk, newUnconfirmedBlocks []*ledger.AccountBlock) error {
	sDB.disableCache()
	defer sDB.enableCache()
	if err := sDB.rollbackRoundCache(deletedSnapshotSegments); err != nil {
		return err
	}

	batch := sDB.store.NewBatch()

	latestSnapshotBlock := sDB.chain.GetLatestSnapshotBlock()
	newUnconfirmedLog, hasRedo, err := sDB.redo.QueryLog(latestSnapshotBlock.Height + 1)

	if err != nil {
		return errors.New(fmt.Sprintf("1. sDB.redo.QueryLog failed, height is %d. Error: %s", latestSnapshotBlock.Height+1, err.Error()))
	}

	snapshotHeight := latestSnapshotBlock.Height

	hasBuiltInContract := true

	if hasRedo {
		rollbackKeySet := make(map[types.Address]map[string]struct{})
		rollbackTokenSet := make(map[types.Address]map[types.TokenTypeId]struct{})

		for index, seg := range deletedSnapshotSegments {
			snapshotHeight += 1

			for _, accountBlock := range seg.AccountBlocks {
				if !hasBuiltInContract && sDB.shouldCacheContractData(accountBlock.AccountAddress) {
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
		if err := sDB.recoverLatestIndexToSnapshot(batch, ledger.HashHeight{
			Height: latestSnapshotBlock.Height,
			Hash:   latestSnapshotBlock.Hash,
		}, rollbackKeySet, rollbackTokenSet); err != nil {
			return err
		}

	} else {
		addrMap := make(map[types.Address]struct{}) // for recover to latest snapshot

		var oldUnconfirmedLog map[types.Address][]LogItem

		for index, seg := range deletedSnapshotSegments {
			snapshotHeight += 1

			// get old unconfirmed Log
			if index == len(deletedSnapshotSegments)-1 && seg.SnapshotBlock == nil {
				if oldUnconfirmedLog, _, err = sDB.redo.QueryLog(latestSnapshotBlock.Height + uint64(len(deletedSnapshotSegments))); err != nil {
					return errors.New(fmt.Sprintf("3. sDB.redo.QueryLog failed, height is %d. Error: %s", latestSnapshotBlock.Height+uint64(len(deletedSnapshotSegments)), err.Error()))
				}
			}

			for _, accountBlock := range seg.AccountBlocks {
				addrMap[accountBlock.AccountAddress] = struct{}{}

				if !hasBuiltInContract && sDB.shouldCacheContractData(accountBlock.AccountAddress) {
					hasBuiltInContract = true
				}

				// rollback code, contract meta, log hash, call depth
				sDB.rollbackAccountBlock(batch, accountBlock)
			}

		}

		// recover latest and history index
		if err := sDB.recoverToSnapshot(batch, latestSnapshotBlock.Height, oldUnconfirmedLog, addrMap); err != nil {
			return err
		}

	}

	// commit
	sDB.store.RollbackSnapshot(batch)

	// rollback redo
	sDB.redo.Rollback(deletedSnapshotSegments)

	// recover redo
	sDB.redo.SetCurrentSnapshot(latestSnapshotBlock.Height+1, newUnconfirmedLog)

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

	latestSnapshotBlock := sDB.chain.GetLatestSnapshotBlock()

	unconfirmedLog, hasRedo, err := sDB.redo.QueryLog(latestSnapshotBlock.Height + 1)
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
	if err := sDB.recoverLatestIndexToSnapshot(batch, ledger.HashHeight{
		Height: latestSnapshotBlock.Height,
		Hash:   latestSnapshotBlock.Hash,
	}, rollbackKeySet, rollbackTokenSet); err != nil {
		return err
	}

	// set redo log
	sDB.redo.SetCurrentSnapshot(latestSnapshotBlock.Height+1, unconfirmedLog)

	// write
	sDB.store.RollbackAccountBlocks(batch, accountBlocks)
	return nil
}

func (sDB *StateDB) rollbackByRedo(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, redoLogMap map[types.Address][]LogItem,
	rollbackKeySet map[types.Address]map[string]struct{}, rollbackTokenSet map[types.Address]map[types.TokenTypeId]struct{}) error {
	// rollback history storage key value and record rollback keys
	var resetKeyTemplate chain_utils.DBKeyStorage
	var resetBalanceTemplate chain_utils.DBKeyBalance
	isHistory := true
	if snapshotBlock == nil {
		isHistory = false
		key := chain_utils.CreateStorageValueKey(&types.Address{}, []byte{})
		resetKeyTemplate = &key
		balanceKey := chain_utils.CreateBalanceKey(types.Address{}, types.TokenTypeId{})
		resetBalanceTemplate = &balanceKey

	} else {
		key := chain_utils.CreateHistoryStorageValueKey(&types.Address{}, []byte{}, snapshotBlock.Height)
		resetKeyTemplate = &key
		balanceKey := chain_utils.CreateHistoryBalanceKey(types.Address{}, types.TokenTypeId{}, snapshotBlock.Height)
		resetBalanceTemplate = &balanceKey
	}

	for addr, redoLogList := range redoLogMap {
		// copy addr
		//copy(resetKeyTemplate[1:1+types.AddressSize], addr.Bytes())
		//copy(resetBalanceTemplate[1:1+types.AddressSize], addr.Bytes())
		resetKeyTemplate.AddressRefill(addr)
		resetBalanceTemplate.AddressRefill(addr)

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
				//copy(resetKeyTemplate[1+types.AddressSize:], common.RightPadBytes(kv[0], 32))
				//resetKeyTemplate[1+types.AddressSize+types.HashSize] = byte(len(kv[0]))
				resetKeyTemplate.KeyRefill(chain_utils.StorageRealKey{}.Construct(kv[0]))

				if isHistory {
					sDB.deleteHistoryKey(batch, resetKeyTemplate.Bytes())
				} else {
					batch.Delete(resetKeyTemplate.Bytes())
				}
			}

			// delete snapshot balance
			for tokenTypeId := range redoLog.BalanceMap {

				tokenSet[tokenTypeId] = struct{}{}

				//copy(resetBalanceTemplate[1+types.AddressSize:], tokenTypeId.Bytes())
				resetBalanceTemplate.TokenIdRefill(tokenTypeId)

				if isHistory {
					batch.Delete(resetBalanceTemplate.Bytes())
				} else {
					sDB.deleteBalance(batch, resetBalanceTemplate.Bytes())
				}
			}

			// delete contract meta
			for contractAddr := range redoLog.ContractMeta {
				sDB.deleteContractMeta(batch, chain_utils.CreateContractMetaKey(contractAddr).Bytes())
			}

			// delete code
			if len(redoLog.Code) > 0 {
				batch.Delete(chain_utils.CreateCodeKey(addr).Bytes())
			}

			// delete vm log list
			for logHash := range redoLog.VmLogList {

				batch.Delete(chain_utils.CreateVmLogListKey(&logHash).Bytes())

			}

			// delete call depth
			for sendBlockHash := range redoLog.CallDepth {

				batch.Delete(chain_utils.CreateCallDepthKey(sendBlockHash).Bytes())

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
func (sDB *StateDB) recoverLatestIndexToSnapshot(batch *leveldb.Batch, hashHeight ledger.HashHeight, keySetMap map[types.Address]map[string]struct{}, tokenSetMap map[types.Address]map[types.TokenTypeId]struct{}) error {

	// recover kv latest index
	for addr, keySet := range keySetMap {
		storage := NewStorageDatabase(sDB, hashHeight, addr)

		for keyStr := range keySet {

			key := []byte(keyStr)

			value, err := storage.GetValue(key)
			if err != nil {
				return err
			}

			if len(value) <= 0 {

				batch.Delete(chain_utils.CreateStorageValueKey(&addr, key).Bytes())
			} else {

				batch.Put(chain_utils.CreateStorageValueKey(&addr, key).Bytes(), value)
			}

		}
	}

	// recover balance latest index

	iter := sDB.store.NewIterator(util.BytesPrefix([]byte{chain_utils.BalanceHistoryKeyPrefix}))
	defer iter.Release()

	balanceTemplateKey := chain_utils.CreateBalanceKey(types.Address{}, types.TokenTypeId{})

	seekKey := chain_utils.CreateHistoryBalanceKey(types.Address{}, types.TokenTypeId{}, hashHeight.Height+1)

	for addr, tokenSet := range tokenSetMap {
		// copy addr
		//addrBytes := addr.Bytes()
		//copy(seekKey[1:1+types.AddressSize], addrBytes)
		//copy(balanceTemplateKey[1:1+types.AddressSize], addrBytes)
		seekKey.AddressRefill(addr)
		balanceTemplateKey.AddressRefill(addr)

		for tokenId := range tokenSet {
			// copy tokenIdBytes
			//tokenIdBytes := tokenId.Bytes()
			//copy(seekKey[1+types.AddressSize:], tokenIdBytes)
			//copy(balanceTemplateKey[1+types.AddressSize:], tokenIdBytes)
			seekKey.TokenIdRefill(tokenId)
			balanceTemplateKey.TokenIdRefill(tokenId)

			iter.Seek(seekKey.Bytes())

			if iter.Prev() && bytes.HasPrefix(iter.Key(), seekKey[:len(seekKey)-8]) {
				// set latest

				sDB.writeBalance(batch, balanceTemplateKey.Bytes(), iter.Value())
			} else {
				sDB.deleteBalance(batch, balanceTemplateKey.Bytes())

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
				if len(kv[1]) <= 0 {
					batch.Delete(chain_utils.CreateStorageValueKey(&addr, kv[0]).Bytes())
				} else {
					batch.Put(chain_utils.CreateStorageValueKey(&addr, kv[0]).Bytes(), kv[1])
				}
				if _, ok := rollbackKeySet[addr]; ok {
					delete(rollbackKeySet[addr], string(kv[0]))
				}
			}

			for tokenId, balance := range redoLog.BalanceMap {
				sDB.writeBalance(batch, chain_utils.CreateBalanceKey(addr, tokenId).Bytes(), balance.Bytes())

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
		batch.Delete(chain_utils.CreateCodeKey(accountBlock.AccountAddress).Bytes())
	}

	// delete contract meta
	if accountBlock.BlockType == ledger.BlockTypeSendCreate {
		sDB.deleteContractMeta(batch, chain_utils.CreateContractMetaKey(accountBlock.ToAddress).Bytes())
	}

	// delete log hash
	if accountBlock.LogHash != nil {
		batch.Delete(chain_utils.CreateVmLogListKey(accountBlock.LogHash).Bytes())
	}

	// delete call depth && contract meta
	for _, sendBlock := range accountBlock.SendBlockList {
		batch.Delete(chain_utils.CreateCallDepthKey(sendBlock.Hash).Bytes())

		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			sDB.deleteContractMeta(batch, chain_utils.CreateContractMetaKey(sendBlock.ToAddress).Bytes())
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
		storageKeyBytes := chain_utils.StorageRealKey{}.ConstructFix(iter.Key()[keyStartIndex:keyEndIndex])
		//copy(seekTemplateKey[keyStartIndex:keyEndIndex], storageKeyBytes)
		//copy(storageTemplateKey[keyStartIndex:keyEndIndex], storageKeyBytes)
		seekTemplateKey.KeyRefill(storageKeyBytes)
		storageTemplateKey.KeyRefill(storageKeyBytes)

		// real key
		//realKey := string(storageKeyBytes[:storageKeyBytes[len(storageKeyBytes)-1]])
		realKey := storageKeyBytes.String()

		// delete real key
		delete(keySet, realKey)

		// seek
		iter.Seek(seekTemplateKey.Bytes())

		prevOk := iter.Prev()
		prevKey := chain_utils.StorageHistoryKey{}.Construct(iter.Key())
		if prevKey == nil {
			sDB.log.Error("prevKey is nil.")
			break
		}
		if prevKey.ExtraAddress() != addr {
			sDB.log.Error("address is different.")
			break
		}

		if prevOk && bytes.Equal(seekTemplateKey.ExtraKeyAndLen(), prevKey.ExtraKeyAndLen()) {
			if len(iter.Value()) > 0 {
				batch.Put(storageTemplateKey.Bytes(), iter.Value())
			} else {
				batch.Delete(storageTemplateKey.Bytes())
			}
		} else {
			batch.Delete(storageTemplateKey.Bytes())
		}

		for {
			iterOk = iter.Next()
			if !iterOk {
				break
			}

			key := chain_utils.StorageHistoryKey{}.Construct(iter.Key())
			if key == nil || key.ExtraAddress() != addr {
				sDB.log.Error("iter key is nill or addr not match")
				break
			}

			if bytes.Equal(seekTemplateKey.ExtraKeyAndLen(), key.ExtraKeyAndLen()) {
				// delete history
				sDB.deleteHistoryKey(batch, key.Bytes())
			} else {
				break
			}

		}

	}
	if err := iter.Error(); err != nil {
		return err
	}

	// delete unconfirmed key
	for keyStr := range keySet {
		key := []byte(keyStr)
		copy(storageTemplateKey[keyStartIndex:], common.RightPadBytes(key, types.HashSize))
		storageTemplateKey[1+types.AddressSize+types.HashSize] = byte(len(key))
		batch.Delete(storageTemplateKey.Bytes())
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
		iterKey := chain_utils.BalanceHistoryKey{}.Construct(iter.Key())
		tokenId := iterKey.ExtraTokenId()
		tokenIdBytes := tokenId.Bytes()

		// copy tokenIdBytes
		//copy(seekTemplateKey[tokenIdStartIndex:tokenIdEndIndex], tokenIdBytes)
		//copy(balanceTemplateKey[tokenIdStartIndex:tokenIdEndIndex], tokenIdBytes)
		seekTemplateKey.TokenIdRefill(tokenId)
		balanceTemplateKey.TokenIdRefill(tokenId)

		// parse to tokenId
		tokenId, err := types.BytesToTokenTypeId(tokenIdBytes)
		if err != nil {
			return err
		}
		// delete token set
		delete(tokenSet, tokenId)
		// seek
		iter.Seek(seekTemplateKey.Bytes())

		prevOk := iter.Prev()
		prevKey := chain_utils.BalanceHistoryKey{}.Construct(iter.Key())

		if prevOk && bytes.Equal(seekTemplateKey[tokenIdStartIndex:tokenIdEndIndex], iter.Key()[tokenIdStartIndex:tokenIdEndIndex]) {
			if seekTemplateKey.ExtraTokenId() != prevKey.ExtraTokenId() {
				panic("not equal")
			}
		}
		// recover balance in latest snapshot from history snapshot
		// if iter.Prev() && bytes.Equal(seekTemplateKey[tokenIdStartIndex:tokenIdEndIndex], iter.Key()[tokenIdStartIndex:tokenIdEndIndex]) {
		if prevOk && seekTemplateKey.ExtraTokenId() == prevKey.ExtraTokenId() {
			//fmt.Printf("recoverToLatestBalance %d, %d. %s. %s: %s\n", height, prevKey.ExtraHeight(), addr, tokenId, new(big.Int).SetBytes(iter.Value()))
			sDB.writeBalance(batch, balanceTemplateKey.Bytes(), iter.Value())
		} else {
			//fmt.Printf("deleteCurrentBalance %d. %s. %s: %s\n", height, addr, tokenId, new(big.Int).SetBytes(iter.Value()))
			sDB.deleteBalance(batch, balanceTemplateKey.Bytes())
		}
		for { // delete all balance store [current height+1, max]
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
		//copy(balanceTemplateKey[tokenIdStartIndex:tokenIdEndIndex], tokenId.Bytes())
		balanceTemplateKey.TokenIdRefill(tokenId)
		sDB.deleteBalance(batch, balanceTemplateKey.Bytes())
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
	if sDB.shouldCacheContractData(addr) {
		sDB.cache.Delete(snapshotValuePrefix + string(addrBytes) + string(sDB.parseStorageKey(key)))
	}
}

func (sDB *StateDB) rollbackRoundCache(deletedSnapshotSegments []*ledger.SnapshotChunk) error {
	// delete round cache
	deletedSnapshotBlocks := make([]*ledger.SnapshotBlock, 0, len(deletedSnapshotSegments))
	for _, chunk := range deletedSnapshotSegments {
		if chunk.SnapshotBlock != nil {
			deletedSnapshotBlocks = append(deletedSnapshotBlocks, chunk.SnapshotBlock)
		}
	}
	return sDB.roundCache.DeleteSnapshotBlocks(deletedSnapshotBlocks)
}
