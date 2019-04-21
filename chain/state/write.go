package chain_state

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (sDB *StateDB) Write(block *vm_db.VmAccountBlock) error {
	batch := sDB.store.NewBatch()

	vmDb := block.VmDb
	accountBlock := block.AccountBlock
	var redoLog LogItem

	// write unsaved storage
	unsavedStorage := vmDb.GetUnsavedStorage()

	for _, kv := range unsavedStorage {

		// set latest kv
		batch.Put(chain_utils.CreateStorageValueKey(&accountBlock.AccountAddress, kv[0]), kv[1])
	}
	redoLog.Storage = unsavedStorage

	// write unsaved balance
	unsavedBalanceMap := vmDb.GetUnsavedBalanceMap()

	for tokenTypeId, balance := range unsavedBalanceMap {
		// set latest balance
		batch.Put(chain_utils.CreateBalanceKey(accountBlock.AccountAddress, tokenTypeId), balance.Bytes())

		//fmt.Println("PUT BALANCE", accountBlock.AccountAddress, tokenTypeId, balance)
	}

	redoLog.BalanceMap = unsavedBalanceMap

	// write unsaved code
	unsavedCode := vmDb.GetUnsavedContractCode()
	if unsavedCode != nil {
		codeKey := chain_utils.CreateCodeKey(accountBlock.AccountAddress)

		batch.Put(codeKey, unsavedCode)

		redoLog.Code = unsavedCode
	}

	// write unsaved contract meta
	unsavedContractMeta := vmDb.GetUnsavedContractMeta()
	if len(unsavedContractMeta) > 0 {
		redoLog.ContractMeta = make(map[types.Address][]byte, len(unsavedContractMeta))
		for addr, meta := range unsavedContractMeta {
			contractKey := chain_utils.CreateContractMetaKey(addr)
			gidContractKey := chain_utils.CreateGidContractKey(meta.Gid, &addr)

			metaBytes := meta.Serialize()
			batch.Put(contractKey, metaBytes)
			batch.Put(gidContractKey, nil)

			redoLog.ContractMeta[addr] = metaBytes

			// set meta
			sDB.cache.Set(contractAddrPrefix+string(addr.Bytes()), metaBytes)
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
		redoLog.VmLogList = map[types.Hash][]byte{*accountBlock.LogHash: bytes}
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
		redoLog.CallDepth = make(map[types.Hash]uint16, len(accountBlock.SendBlockList))

		for _, sendBlock := range accountBlock.SendBlockList {
			redoLog.CallDepth[sendBlock.Hash] = callDepth
			batch.Put(chain_utils.CreateCallDepthKey(sendBlock.Hash), callDepthBytes)
		}

	}

	// add storage redo log
	redoLog.Height = accountBlock.Height

	sDB.redo.AddLog(accountBlock.AccountAddress, redoLog)

	// write batch
	sDB.store.WriteAccountBlock(batch, block.AccountBlock)

	return nil
}

func (sDB *StateDB) WriteByRedo(blockHash types.Hash, addr types.Address, redoLog LogItem) {
	batch := sDB.store.NewBatch()

	// write unsaved storage
	for _, kv := range redoLog.Storage {
		// set latest kv
		batch.Put(chain_utils.CreateStorageValueKey(&addr, kv[0]), kv[1])
	}

	// write unsaved balance
	for tokenTypeId, balance := range redoLog.BalanceMap {
		// set latest balance
		batch.Put(chain_utils.CreateBalanceKey(addr, tokenTypeId), balance.Bytes())
		//fmt.Println("recover unconfirmed", addr, redoLog.Height, balance)
	}

	// write unsaved code
	unsavedCode := redoLog.Code
	if len(unsavedCode) > 0 {
		codeKey := chain_utils.CreateCodeKey(addr)

		batch.Put(codeKey, unsavedCode)
	}

	// write unsaved contract meta
	unsavedContractMeta := redoLog.ContractMeta

	for addr, metaBytes := range unsavedContractMeta {
		contractKey := chain_utils.CreateContractMetaKey(addr)

		gidContractKey := make([]byte, 0, 1+types.AddressSize+types.GidSize)
		gidContractKey = append(gidContractKey, chain_utils.GidContractKeyPrefix)
		gidContractKey = append(gidContractKey, addr.Bytes()...)
		gidContractKey = append(gidContractKey, metaBytes[:types.GidSize]...)

		batch.Put(contractKey, metaBytes)
		batch.Put(gidContractKey, nil)
		// set
		sDB.cache.Set(contractAddrPrefix+string(addr.Bytes()), metaBytes)
	}

	// write vm log

	for logHash, vmLogListBytes := range redoLog.VmLogList {

		batch.Put(chain_utils.CreateVmLogListKey(&logHash), vmLogListBytes)
	}

	// write call depth
	callDepthBytes := make([]byte, 2)
	for sendHash, callDepth := range redoLog.CallDepth {

		binary.BigEndian.PutUint16(callDepthBytes, callDepth)
		batch.Put(chain_utils.CreateCallDepthKey(sendHash), callDepthBytes)

	}
	sDB.store.WriteAccountBlockByHash(batch, blockHash)
}

// TODO redo
func (sDB *StateDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedBlocks []*ledger.AccountBlock) error {
	height := snapshotBlock.Height

	// next snapshot
	sDB.redo.NextSnapshot(height+1, confirmedBlocks)

	// write history
	snapshotRedoLog, _, err := sDB.redo.QueryLog(height)
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)

	if len(snapshotRedoLog) > 0 {

		redoKvMap, redoBalanceMap, err := parseRedoLog(snapshotRedoLog)
		if err != nil {
			return err
		}

		// put history storage kv
		putKeyTemplate := make([]byte, 1+types.AddressSize+types.HashSize+9)

		putKeyTemplate[0] = chain_utils.StorageHistoryKeyPrefix
		binary.BigEndian.PutUint64(putKeyTemplate[len(putKeyTemplate)-8:], height)

		for addr, kvMap := range redoKvMap {

			copy(putKeyTemplate[1:1+types.AddressSize], addr.Bytes())

			writeCache := false
			var addrStr string
			if types.IsBuiltinContractAddr(addr) {
				writeCache = true
				addrStr = string(addr.Bytes())

			}
			for keyStr, value := range kvMap {
				// record rollback key
				key := []byte(keyStr)
				copy(putKeyTemplate[1+types.AddressSize:], common.RightPadBytes(key, 32))
				putKeyTemplate[len(putKeyTemplate)-9] = byte(len(key))

				batch.Put(putKeyTemplate, value)
				if writeCache {
					sDB.cache.Set(snapshotValuePrefix+addrStr+keyStr, value)
				}
				//fmt.Printf("write history storage, addr: %s, key: %d, putKeyTemplate: %d, value: %d\n", addr, key, putKeyTemplate, value)
			}

			// set rollback key set
		}

		// put history balance
		putBalanceKeyTemplate := make([]byte, 1+types.AddressSize+types.TokenTypeIdSize+8)

		putBalanceKeyTemplate[0] = chain_utils.BalanceHistoryKeyPrefix

		binary.BigEndian.PutUint64(putBalanceKeyTemplate[len(putBalanceKeyTemplate)-8:], height)

		for addr, balanceMap := range redoBalanceMap {
			copy(putBalanceKeyTemplate[1:1+types.AddressSize], addr.Bytes())
			for tokenTypeId, balance := range balanceMap {
				copy(putBalanceKeyTemplate[1+types.AddressSize:], tokenTypeId.Bytes())

				batch.Put(putBalanceKeyTemplate, balance.Bytes())
			}
		}
	}

	// write snapshot
	sDB.store.WriteSnapshot(batch, confirmedBlocks)
	return nil

}
