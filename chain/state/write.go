package chain_state

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (sDB *StateDB) Write(block *vm_db.VmAccountBlock) error {
	batch := sDB.store.NewBatch()

	vmDb := block.VmDb
	accountBlock := block.AccountBlock

	// write unsaved storage
	unsavedStorage := vmDb.GetUnsavedStorage()
	unsavedBalanceMap := vmDb.GetUnsavedBalanceMap()

	historyKvList := make([][2][]byte, 0, len(unsavedStorage)+len(unsavedBalanceMap))

	for _, kv := range unsavedStorage {

		// set latest
		storageKey := chain_utils.CreateStorageValueKey(&accountBlock.AccountAddress, kv[0])

		batch.Put(storageKey, kv[1])

		// set history
		historyKvList = append(historyKvList, [2][]byte{chain_utils.CreateHistoryStorageValueKey(&accountBlock.AccountAddress, kv[0], 0), kv[1]})

		// FOR DEBUG
		//fmt.Println("write storage key", block.AccountBlock.AccountAddress, block.AccountBlock.Height, sDB.chain.GetLatestSnapshotBlock().Hash, sDB.chain.GetLatestSnapshotBlock().Height, kv[0], kv[1])
	}

	// write kv redo log
	kvListBytes, err := rlp.EncodeToBytes(unsavedStorage)
	if err != nil {
		return err
	}
	sDB.storageRedo.AddLog(accountBlock.Hash, kvListBytes)

	// write unsaved balance
	for tokenTypeId, balance := range unsavedBalanceMap {

		// set latest
		balanceKey := chain_utils.CreateBalanceKey(accountBlock.AccountAddress, tokenTypeId)

		balanceBytes := balance.Bytes()

		batch.Put(balanceKey, balanceBytes)

		// set history
		historyKvList = append(historyKvList, [2][]byte{chain_utils.CreateHistoryBalanceKey(accountBlock.AccountAddress, tokenTypeId, 0), balanceBytes})

		// FOR DEBUG
		//fmt.Println("write balance", block.AccountBlock.AccountAddress, block.AccountBlock.Height, sDB.chain.GetLatestSnapshotBlock().Hash, sDB.chain.GetLatestSnapshotBlock().Height, balance)
	}

	// write history
	sDB.historyKvMu.Lock()
	sDB.historyKv[accountBlock.Hash] = historyKvList
	sDB.historyKvMu.Unlock()

	// write unsaved code
	unsavedCode := vmDb.GetUnsavedContractCode()
	if unsavedCode != nil {
		codeKey := chain_utils.CreateCodeKey(accountBlock.AccountAddress)

		batch.Put(codeKey, unsavedCode)

	}

	// write unsaved contract meta
	unsavedContractMeta := vmDb.GetUnsavedContractMeta()
	if len(unsavedContractMeta) > 0 {
		for addr, meta := range unsavedContractMeta {
			contractKey := chain_utils.CreateContractMetaKey(addr)
			gidContractKey := chain_utils.CreateGidContractKey(meta.Gid, &addr)

			batch.Put(contractKey, meta.Serialize())
			batch.Put(gidContractKey, nil)
		}

	}

	// write vm log
	if accountBlock.LogHash != nil {
		vmLogListKey := chain_utils.CreateVmLogListKey(accountBlock.LogHash)

		bytes, err := vmDb.GetLogList().Serialize()
		if err != nil {
			return err
		}
		batch.Put(vmLogListKey, bytes)
	}

	// write call depth
	if accountBlock.IsReceiveBlock() && len(accountBlock.SendBlockList) > 0 {
		callDepth, err := vmDb.GetCallDepth(&accountBlock.FromBlockHash)
		if err != nil {
			return err
		}

		callDepth += 1
		callDepthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(callDepthBytes, callDepth)

		for _, sendBlock := range accountBlock.SendBlockList {
			batch.Put(chain_utils.CreateCallDepthKey(&sendBlock.Hash), callDepthBytes)
		}
	}

	sDB.store.WriteAccountBlock(batch, block.AccountBlock)

	return nil
}

// TODO redo
func (sDB *StateDB) InsertSnapshotBlocks(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) {
	// write history
	batch := new(leveldb.Batch)

	height := snapshotBlock.Height
	heightBytes := chain_utils.Uint64ToBytes(height)

	sDB.historyKvMu.Lock()
	for _, confirmedBlock := range confirmedBlocks {
		kvList := sDB.historyKv[confirmedBlock.Hash]
		for _, kv := range kvList {
			key := kv[0]
			copy(key[len(key)-8:], heightBytes)

			batch.Put(key, kv[1])
		}
		delete(sDB.historyKv, confirmedBlock.Hash)
	}
	sDB.historyKvMu.Unlock()

	// write snapshot
	sDB.store.WriteSnapshot(batch, confirmedBlocks)

	// next snapshot
	sDB.storageRedo.NextSnapshot(height+1, confirmedBlocks)

}
