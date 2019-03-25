package chain_state

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/block"
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

	undoLogSize := types.HashSize +
		len(unsavedStorage)*(types.AddressSize+34) + len(unsavedBalanceMap)*(1+types.AddressSize+types.TokenTypeIdSize)

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

	binary.BigEndian.PutUint32(undoLog[undoLogSize:], uint32(undoLogSize))

	// write unsaved code
	unsavedCode := vmDb.GetUnsavedContractCode()
	if unsavedCode != nil {
		codeKey := chain_utils.CreateCodeKey(&accountBlock.AccountAddress)

		sDB.pending.Put(&accountBlock.Hash, codeKey, unsavedCode)
	}

	// write unsaved contract meta
	unsavedContractMeta := vmDb.GetUnsavedContractMeta()
	if unsavedContractMeta != nil {
		contractKey := chain_utils.CreateContractMetaKey(&accountBlock.AccountAddress)

		sDB.pending.Put(&accountBlock.Hash, contractKey, unsavedContractMeta.Serialize())
	}

	// write vm log
	if accountBlock.LogHash != nil {
		vmLogListKey := chain_utils.CreateVmLogListKey(accountBlock.LogHash)

		bytes, err := vmDb.GetLogList().Serialize()
		if err != nil {
			return err
		}
		sDB.pending.Put(&accountBlock.Hash, vmLogListKey, bytes)
	}

	sDB.undoLogger.InsertBlock(&accountBlock.Hash, undoLog)

	return nil
}

func (sDB *StateDB) Flush(snapshotBlock *ledger.SnapshotBlock, blocks []*ledger.AccountBlock,
	invalidAccountBlocks []*ledger.AccountBlock, location *chain_block.Location) error {
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

func (sDB *StateDB) updateUndoLocation(batch *leveldb.Batch, location *chain_block.Location) {
	batch.Put(chain_utils.CreateUndoLocationKey(), chain_utils.SerializeLocation(location))

}

func (sDB *StateDB) updateStateDbLocation(batch *leveldb.Batch, latestLocation *chain_block.Location) {
	batch.Put(chain_utils.CreateStateDbLocationKey(), chain_utils.SerializeLocation(latestLocation))

}
