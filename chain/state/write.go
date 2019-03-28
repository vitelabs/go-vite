package chain_state

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (sDB *StateDB) Write(block *vm_db.VmAccountBlock) error {
	vmDb := block.VmDb
	accountBlock := block.AccountBlock

	latestSnapshotBlock := sDB.chain.GetLatestSnapshotBlock()
	nextSnapshotHeight := latestSnapshotBlock.Height + 1

	// write unsaved storage
	unsavedStorage := vmDb.GetUnsavedStorage()
	unsavedBalanceMap := vmDb.GetUnsavedBalanceMap()
	unsavedCode := vmDb.GetUnsavedContractCode()
	unsavedContractMeta := vmDb.GetUnsavedContractMeta()

	undoLogSize := types.HashSize +
		len(unsavedStorage)*(types.AddressSize+34) + len(unsavedBalanceMap)*(1+types.AddressSize+types.TokenTypeIdSize)

	if unsavedCode != nil {
		undoLogSize += 1 + types.AddressSize
	}

	if len(unsavedContractMeta) > 0 {
		undoLogSize += len(unsavedContractMeta) * (1 + types.AddressSize + 1 + types.GidSize + types.AddressSize)
	}

	undoLog := make([]byte, 0, undoLogSize+4)
	undoLog = append(undoLog, accountBlock.Hash.Bytes()...)

	for _, kv := range unsavedStorage {
		storageKey := chain_utils.CreateStorageValueKey(&accountBlock.AccountAddress, kv[0])

		historyStorageKey := chain_utils.CreateHistoryStorageValueKey(&accountBlock.AccountAddress, kv[0], nextSnapshotHeight)

		sDB.pending.Put(&accountBlock.Hash, storageKey, kv[1])

		sDB.pending.Put(&accountBlock.Hash, historyStorageKey, kv[1])

		undoLog = append(undoLog, storageKey...)
	}

	// write unsaved balance
	for tokenTypeId, balance := range unsavedBalanceMap {

		balanceKey := chain_utils.CreateBalanceKey(&accountBlock.AccountAddress, &tokenTypeId)

		balanceStorageKey := chain_utils.CreateHistoryBalanceKey(&accountBlock.AccountAddress, &tokenTypeId, nextSnapshotHeight)

		balanceBytes := balance.Bytes()

		sDB.pending.Put(&accountBlock.Hash, balanceKey, balanceBytes)

		sDB.pending.Put(&accountBlock.Hash, balanceStorageKey, balanceBytes)

		undoLog = append(undoLog, balanceKey...)

	}

	// write unsaved code
	if unsavedCode != nil {
		codeKey := chain_utils.CreateCodeKey(&accountBlock.AccountAddress)

		sDB.pending.Put(&accountBlock.Hash, codeKey, unsavedCode)

		undoLog = append(undoLog, codeKey...)
	}

	// write unsaved contract meta
	if len(unsavedContractMeta) > 0 {
		for addr, meta := range unsavedContractMeta {
			contractKey := chain_utils.CreateContractMetaKey(&addr)
			gidContractKey := chain_utils.CreateGidContractKey(meta.Gid, &addr)

			sDB.pending.Put(&accountBlock.Hash, contractKey, meta.Serialize())
			sDB.pending.Put(&accountBlock.Hash, gidContractKey, nil)

			undoLog = append(undoLog, contractKey...)
			undoLog = append(undoLog, gidContractKey...)
		}

	}

	undoLog = append(undoLog, make([]byte, 4)...)
	binary.BigEndian.PutUint32(undoLog[undoLogSize:], uint32(undoLogSize))

	// write vm log
	if accountBlock.LogHash != nil {
		vmLogListKey := chain_utils.CreateVmLogListKey(accountBlock.LogHash)

		bytes, err := vmDb.GetLogList().Serialize()
		if err != nil {
			return err
		}
		sDB.pending.Put(&accountBlock.Hash, vmLogListKey, bytes)
	}

	// write call depth
	callDepth := vmDb.GetUnsavedCallDepth()
	if accountBlock.IsReceiveBlock() && callDepth > 0 {
		callDepthBytes := make([]byte, 16)
		binary.BigEndian.PutUint16(callDepthBytes, callDepth)

		for _, sendBlock := range accountBlock.SendBlockList {
			sDB.pending.Put(&accountBlock.Hash, chain_utils.CreateCallDepthKey(&sendBlock.Hash), callDepthBytes)
		}
	}

	sDB.undoLogger.InsertBlock(&accountBlock.Hash, undoLog)

	return nil
}

func (sDB *StateDB) Flush(snapshotBlock *ledger.SnapshotBlock, blocks []*ledger.AccountBlock,
	invalidAccountBlocks []*ledger.AccountBlock, location *chain_file_manager.Location) error {
	batch := new(leveldb.Batch)
	blockHashList := make([]*types.Hash, 0, len(blocks))
	for _, block := range blocks {
		blockHashList = append(blockHashList, &block.Hash)
	}

	location, err := sDB.undoLogger.Flush(&snapshotBlock.Hash, blockHashList)
	if err != nil {
		return err
	}

	sDB.updateUndoLocation(batch, location)
	sDB.pending.FlushList(batch, blockHashList)

	sDB.updateStateDbLocation(batch, location)

	if err := sDB.db.Write(batch, nil); err != nil {
		return err
	}

	for _, block := range invalidAccountBlocks {
		sDB.pending.DeleteByBlockHash(&block.Hash)
	}

	return nil
}

func (sDB *StateDB) updateUndoLocation(batch *leveldb.Batch, location *chain_file_manager.Location) {
	batch.Put(chain_utils.CreateUndoLocationKey(), chain_utils.SerializeLocation(location))

}

func (sDB *StateDB) updateStateDbLocation(batch *leveldb.Batch, latestLocation *chain_file_manager.Location) {
	batch.Put(chain_utils.CreateStateDbLocationKey(), chain_utils.SerializeLocation(latestLocation))

}
