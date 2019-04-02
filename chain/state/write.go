package chain_state

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (sDB *StateDB) Write(block *vm_db.VmAccountBlock) error {
	batch := sDB.store.NewBatch()
	vmDb := block.VmDb
	accountBlock := block.AccountBlock

	latestSnapshotBlock := sDB.chain.GetLatestSnapshotBlock()
	nextSnapshotHeight := uint64(1)
	if latestSnapshotBlock != nil {
		nextSnapshotHeight = latestSnapshotBlock.Height + 1
	}

	// write unsaved storage
	unsavedStorage := vmDb.GetUnsavedStorage()
	for _, kv := range unsavedStorage {

		storageKey := chain_utils.CreateStorageValueKey(&accountBlock.AccountAddress, kv[0])

		historyStorageKey := chain_utils.CreateHistoryStorageValueKey(&accountBlock.AccountAddress, kv[0], nextSnapshotHeight)

		batch.Put(storageKey, kv[1])

		batch.Put(historyStorageKey, kv[1])
	}

	kvListBytes, err := rlp.EncodeToBytes(unsavedStorage)
	if err != nil {
		return err
	}
	sDB.storageRedo.AddLog(accountBlock.Hash, kvListBytes)

	// write unsaved balance
	unsavedBalanceMap := vmDb.GetUnsavedBalanceMap()
	for tokenTypeId, balance := range unsavedBalanceMap {

		balanceKey := chain_utils.CreateBalanceKey(accountBlock.AccountAddress, tokenTypeId)

		balanceStorageKey := chain_utils.CreateHistoryBalanceKey(accountBlock.AccountAddress, tokenTypeId, nextSnapshotHeight)

		balanceBytes := balance.Bytes()

		batch.Put(balanceKey, balanceBytes)

		batch.Put(balanceStorageKey, balanceBytes)

	}

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
	callDepth := vmDb.GetUnsavedCallDepth()
	if accountBlock.IsReceiveBlock() && callDepth > 0 {
		callDepthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(callDepthBytes, callDepth)

		for _, sendBlock := range accountBlock.SendBlockList {
			batch.Put(chain_utils.CreateCallDepthKey(&sendBlock.Hash), callDepthBytes)
		}
	}

	sDB.store.Write(batch)

	return nil
}

func (sDB *StateDB) InsertSnapshotBlock(invalidAccountBlocks []*ledger.AccountBlock) error {
	if len(invalidAccountBlocks) <= 0 {
		return nil
	}

	return sDB.Rollback([]*ledger.SnapshotChunk{{
		AccountBlocks: invalidAccountBlocks,
	}})
}
