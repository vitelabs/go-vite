package chain_state

import (
	"bytes"
	"encoding/binary"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

func (sDB *StateDB) RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk, newUnconfirmedBlocks []*ledger.AccountBlock) error {

	batch := sDB.store.NewBatch()

	latestHeight := sDB.chain.GetLatestSnapshotBlock().Height
	newUnconfirmedLog, hasRedo, err := sDB.redo.QueryLog(latestHeight + 1)

	if err != nil {
		return err
	}

	snapshotHeight := latestHeight

	if hasRedo {
		rollbackKeySet := make(map[types.Address]map[string]struct{})
		rollbackTokenSet := make(map[types.Address]map[types.TokenTypeId]struct{})

		for index, seg := range deletedSnapshotSegments {
			snapshotHeight += 1

			var currentSnapshotLog map[types.Address][]LogItem
			if index <= 0 {
				currentSnapshotLog = newUnconfirmedLog
			} else {
				currentSnapshotLog, _, err = sDB.redo.QueryLog(snapshotHeight)
				if err != nil {
					return err
				}
			}
			// rollback redo
			sDB.redo.Rollback(snapshotHeight)

			// rollback snapshot data
			if err := sDB.rollbackByRedo(batch, seg.SnapshotBlock, currentSnapshotLog, rollbackKeySet, rollbackTokenSet); err != nil {
				return err
			}
		}

		// recover latest index
		if err := sDB.recoverLatestIndex(batch, latestHeight, rollbackKeySet, rollbackTokenSet); err != nil {
			return err
		}

		// set new unconfirmed redo snapshot
		sDB.redo.SetSnapshot(latestHeight+1, newUnconfirmedLog)

	} else {
		addrMap := make(map[types.Address]struct{}) // for recover to latest snapshot

		var oldUnconfirmedLog map[types.Address][]LogItem

		for index, seg := range deletedSnapshotSegments {
			snapshotHeight += 1

			// get old unconfirmed Log
			if index == len(deletedSnapshotSegments)-1 && seg.SnapshotBlock == nil {
				if oldUnconfirmedLog, _, err = sDB.redo.QueryLog(latestHeight + uint64(len(deletedSnapshotSegments))); err != nil {
					return err
				}

			}

			// rollback redo
			sDB.redo.Rollback(snapshotHeight)

			for _, accountBlock := range seg.AccountBlocks {
				addrMap[accountBlock.AccountAddress] = struct{}{}

				// rollback code, contract meta, log hash, call depth
				sDB.rollbackAccountBlock(batch, accountBlock)
			}

		}

		// recover latest and history index
		if err := sDB.recoverToHeight(batch, latestHeight, oldUnconfirmedLog, addrMap); err != nil {
			return err
		}

		// set redo snapshot
		sDB.redo.SetSnapshot(latestHeight+1, make(map[types.Address][]LogItem))
	}

	// commit
	sDB.store.RollbackSnapshot(batch)

	// recover unconfirmed
	if len(newUnconfirmedBlocks) <= 0 {
		return nil
	}

	for _, block := range newUnconfirmedBlocks {
		redoLogList := newUnconfirmedLog[block.AccountAddress]

		startHeight := redoLogList[0].Height
		targetIndex := block.Height - startHeight

		sDB.WriteByRedo(block.Hash, block.AccountAddress, redoLogList[targetIndex])

	}

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

	// recover latest
	sDB.recoverLatestIndexByRedo(batch, addrMap, unconfirmedLog, rollbackKeySet, rollbackTokenSet)

	// recover other latest
	if err := sDB.recoverLatestIndex(batch, latestHeight, rollbackKeySet, rollbackTokenSet); err != nil {
		return err
	}

	// set redo log
	sDB.redo.SetSnapshot(latestHeight+1, unconfirmedLog)

	// write
	sDB.store.RollbackAccountBlocks(batch, accountBlocks)
	return nil
}

// TODO no map
func (sDB *StateDB) rollbackByRedo(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock, redoLogMap map[types.Address][]LogItem,
	rollbackKeySet map[types.Address]map[string]struct{}, rollbackTokenSet map[types.Address]map[types.TokenTypeId]struct{}) error {
	// rollback history storage key value and record rollback keys
	var resetKeyTemplate []byte
	var resetBalanceTemplate []byte
	if snapshotBlock == nil {
		resetKeyTemplate = make([]byte, 1+types.AddressSize+types.HashSize+1)
		resetKeyTemplate[0] = chain_utils.StorageKeyPrefix

		resetBalanceTemplate = make([]byte, 1+types.AddressSize+types.TokenTypeIdSize+8)
		resetBalanceTemplate[0] = chain_utils.BalanceKeyPrefix

	} else {
		resetKeyTemplate = make([]byte, 1+types.AddressSize+types.HashSize+9)
		resetKeyTemplate[0] = chain_utils.StorageHistoryKeyPrefix
		binary.BigEndian.PutUint64(resetKeyTemplate[len(resetKeyTemplate)-8:], snapshotBlock.Height)

		resetBalanceTemplate = make([]byte, 1+types.AddressSize+types.TokenTypeIdSize+8)
		resetBalanceTemplate[0] = chain_utils.BalanceHistoryKeyPrefix
		binary.BigEndian.PutUint64(resetBalanceTemplate[len(resetBalanceTemplate)-8:], snapshotBlock.Height)

	}

	for addr, redoLogList := range redoLogMap {
		copy(resetKeyTemplate[1:1+types.AddressSize], addr.Bytes())
		copy(resetBalanceTemplate[1:1+types.AddressSize], addr.Bytes())

		keySet, _ := rollbackKeySet[addr]

		if keySet == nil {
			keySet = make(map[string]struct{})
		}

		tokenSet, _ := rollbackTokenSet[addr]
		if tokenSet == nil {
			tokenSet = make(map[types.TokenTypeId]struct{})
		}

		for _, redoLog := range redoLogList {
			// delete snapshot storage
			for _, kv := range redoLog.Storage {

				keySet[string(kv[0])] = struct{}{}

				copy(resetKeyTemplate[1+types.AddressSize:], common.RightPadBytes(kv[0], 32))
				resetKeyTemplate[1+types.AddressSize+types.HashSize] = byte(len(kv[0]))

				batch.Delete(resetKeyTemplate)

			}

			// delete snapshot balance
			for tokenTypeId := range redoLog.BalanceMap {

				tokenSet[tokenTypeId] = struct{}{}

				copy(resetBalanceTemplate[1+types.AddressSize:], tokenTypeId.Bytes())

				batch.Delete(resetBalanceTemplate)

			}

			// delete contract meta
			for contractAddr := range redoLog.ContractMeta {

				batch.Delete(chain_utils.CreateContractMetaKey(contractAddr))

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
func (sDB *StateDB) recoverLatestIndex(batch *leveldb.Batch, latestSnapshotHeight uint64, keySetMap map[types.Address]map[string]struct{}, tokenSetMap map[types.Address]map[types.TokenTypeId]struct{}) error {

	// recover kv latest index
	for addr, keySet := range keySetMap {
		storage := NewStorageDatabase(sDB, latestSnapshotHeight, addr)

		for keyStr := range keySet {
			key := []byte(keyStr)

			value, err := storage.GetValue(key)
			if err != nil {
				return err
			}

			//fmt.Println("recoverKvAndBalance query value", addr, key, value)
			if len(value) <= 0 {
				//fmt.Println("recoverKvAndBalance delete key", addr, key)
				batch.Delete(chain_utils.CreateStorageValueKey(&addr, key))
			} else {
				//fmt.Println("recoverKvAndBalance Put key", addr, key, value)
				batch.Put(chain_utils.CreateStorageValueKey(&addr, key), value)
			}

		}
	}

	// recover balance latest index
	prefix := chain_utils.BalanceHistoryKeyPrefix

	iter := sDB.store.NewIterator(util.BytesPrefix([]byte{prefix}))
	defer iter.Release()

	seekKey := make([]byte, 1+types.AddressSize+types.TokenTypeIdSize+8)
	seekKey[0] = prefix
	binary.BigEndian.PutUint64(seekKey[1+types.AddressSize+types.TokenTypeIdSize:], latestSnapshotHeight+1)

	balanceTemplateKey := make([]byte, 1+types.AddressSize+types.TokenTypeIdSize)
	balanceTemplateKey[0] = chain_utils.BalanceKeyPrefix

	for addr, tokenSet := range tokenSetMap {

		addrBytes := addr.Bytes()
		copy(seekKey[1:1+types.AddressSize], addrBytes)
		copy(balanceTemplateKey[1:1+types.AddressSize], addrBytes)

		for tokenId := range tokenSet {
			tokenIdBytes := tokenId.Bytes()

			copy(seekKey[1+types.AddressSize:], tokenIdBytes)
			copy(balanceTemplateKey[1+types.AddressSize:], tokenIdBytes)

			iter.Seek(seekKey)

			if iter.Prev() && bytes.HasPrefix(iter.Key(), seekKey[:len(seekKey)-8]) {
				// set latest
				batch.Put(balanceTemplateKey, iter.Value())
			} else {
				batch.Delete(balanceTemplateKey)
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
				batch.Put(chain_utils.CreateBalanceKey(addr, tokenId), balance.Bytes())
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
		batch.Delete(chain_utils.CreateContractMetaKey(accountBlock.ToAddress))
	}

	// delete log hash
	if accountBlock.LogHash != nil {
		batch.Delete(chain_utils.CreateVmLogListKey(accountBlock.LogHash))
	}

	// delete call depth && contract meta
	for _, sendBlock := range accountBlock.SendBlockList {
		batch.Delete(chain_utils.CreateCallDepthKey(sendBlock.Hash))

		if sendBlock.BlockType == ledger.BlockTypeSendCreate {
			batch.Delete(chain_utils.CreateContractMetaKey(sendBlock.ToAddress))
		}
	}

}

func (sDB *StateDB) recoverToHeight(batch *leveldb.Batch, snapshotHeight uint64, unconfirmedLog map[types.Address][]LogItem, addrMap map[types.Address]struct{}) error {
	keySetMap, tokenSetMap, err := parseRedoLog(unconfirmedLog)
	if err != nil {
		return err
	}

	for addr := range addrMap {
		// recover key value
		if err := sDB.recoverStorageToHeight(batch, snapshotHeight, addr, keySetMap[addr]); err != nil {
			return err
		}

		// recover balance
		if err := sDB.recoverBalanceToHeight(batch, snapshotHeight, addr, tokenSetMap[addr]); err != nil {
			return err
		}

	}
	return nil
}

func (sDB *StateDB) recoverStorageToHeight(batch *leveldb.Batch, height uint64, addr types.Address, keySet map[string][]byte) error {

	prefixKey := append([]byte{chain_utils.StorageHistoryKeyPrefix}, addr.Bytes()...)

	iter := sDB.store.NewIterator(util.BytesPrefix(prefixKey))
	defer iter.Release()

	storageTemplateKey := make([]byte, 1+types.AddressSize+types.HashSize+1)

	storageTemplateKey[0] = chain_utils.StorageKeyPrefix
	copy(storageTemplateKey[1:], addr.Bytes())

	seekTemplateKey := make([]byte, 1+types.AddressSize+types.HashSize+9)
	seekTemplateKey[0] = chain_utils.StorageHistoryKeyPrefix
	copy(seekTemplateKey[1:], addr.Bytes())
	binary.BigEndian.PutUint64(seekTemplateKey[len(seekTemplateKey)-8:], height+1)

	iterOk := iter.Next()
	for iterOk {
		// copy key
		storageKeyBytes := iter.Key()[1+types.AddressSize : 1+types.AddressSize+types.HashSize+1]
		copy(seekTemplateKey[1+types.AddressSize:], storageKeyBytes)

		copy(storageTemplateKey[1+types.AddressSize:], storageKeyBytes)

		delete(keySet, string(storageKeyBytes[:storageKeyBytes[len(storageKeyBytes)-1]]))

		iter.Seek(seekTemplateKey)

		if iter.Prev() && bytes.Equal(seekTemplateKey[1+types.AddressSize:1+types.AddressSize+types.HashSize+1], iter.Key()[1+types.AddressSize:1+types.AddressSize+types.HashSize+1]) {
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
			if bytes.Equal(seekTemplateKey[1+types.AddressSize:1+types.AddressSize+types.HashSize+1], key[1+types.AddressSize:1+types.AddressSize+types.HashSize+1]) {

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
	for key := range keySet {
		//fmt.Printf("recoverToLatestBalance  %d. %s. delete %s\n", height, addr, tokenId)

		copy(storageTemplateKey[1+types.AddressSize:], key)
		storageTemplateKey[1+types.AddressSize+types.HashSize] = byte(len(key))
		batch.Delete(storageTemplateKey)
	}

	return nil
}

func (sDB *StateDB) recoverBalanceToHeight(batch *leveldb.Batch, height uint64, addr types.Address, tokenSet map[types.TokenTypeId]*big.Int) error {

	prefixKey := append([]byte{chain_utils.BalanceHistoryKeyPrefix}, addr.Bytes()...)

	iter := sDB.store.NewIterator(util.BytesPrefix(prefixKey))
	defer iter.Release()

	balanceTemplateKey := make([]byte, 1+types.AddressSize+types.TokenTypeIdSize)
	balanceTemplateKey[0] = chain_utils.BalanceKeyPrefix
	copy(balanceTemplateKey[1:], addr.Bytes())

	seekTemplateKey := make([]byte, 1+types.AddressSize+types.TokenTypeIdSize+8)
	seekTemplateKey[0] = chain_utils.BalanceHistoryKeyPrefix
	copy(seekTemplateKey[1:], addr.Bytes())
	binary.BigEndian.PutUint64(seekTemplateKey[len(seekTemplateKey)-8:], height+1)

	iterOk := iter.Next()
	for iterOk {
		tokenIdBytes := iter.Key()[1+types.AddressSize : 1+types.AddressSize+types.TokenTypeIdSize]
		copy(seekTemplateKey[1+types.AddressSize:], tokenIdBytes)

		copy(balanceTemplateKey[1+types.AddressSize:], tokenIdBytes)
		tokenId, err := types.BytesToTokenTypeId(tokenIdBytes)
		if err != nil {
			return err
		}
		delete(tokenSet, tokenId)
		iter.Seek(seekTemplateKey)

		if iter.Prev() && bytes.Equal(seekTemplateKey[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize], iter.Key()[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize]) {

			//fmt.Printf("recoverToLatestBalance %d. %s. %s: %s\n", height, addr, tokenId, new(big.Int).SetBytes(iter.Value()))

			batch.Put(balanceTemplateKey, iter.Value())
		} else {
			batch.Delete(balanceTemplateKey)
		}
		for {
			iterOk = iter.Next()
			if !iterOk {
				break
			}
			key := iter.Key()
			if bytes.Equal(seekTemplateKey[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize], key[1+types.AddressSize:1+types.AddressSize+types.TokenTypeIdSize]) {

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

		copy(balanceTemplateKey[1+types.AddressSize:], tokenId.Bytes())
		batch.Delete(balanceTemplateKey)
	}

	return nil
}
