package chain_state

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func (sDB *StateDB) Write(block *vm_db.VmAccountBlock) error {
	address := &block.AccountBlock.AccountAddress
	vmDb := block.VmDb

	unsavedBalanceMap := vmDb.GetUnsavedBalanceMap()
	unsavedStorage, deletedKeys := vmDb.GetUnsavedStorage()
	unsavedCode := vmDb.GetUnsavedContractCode()
	unsavedContractMeta := vmDb.GetUnsavedContractMeta()
	unsavedVmLogList := vmDb.GetLogList()

	kvSize := len(unsavedBalanceMap) + len(unsavedStorage) + 3
	keyList := make([][]byte, 0, kvSize)
	valueList := make([][]byte, 0, kvSize)

	// write balance
	balanceKeyList, balanceValueList := sDB.prepareWriteBalance(address, unsavedBalanceMap)
	keyList = append(keyList, balanceKeyList...)
	valueList = append(valueList, balanceValueList...)

	// write storage
	storageKeyList, storageValueList := sDB.prepareWriteStorage(address, unsavedStorage, deletedKeys)
	keyList = append(keyList, storageKeyList...)
	valueList = append(valueList, storageValueList...)

	// write code
	if len(unsavedCode) > 0 {
		keyList = append(keyList, chain_utils.CreateCodeKey(address))
		valueList = append(valueList, unsavedCode)
	}

	// write contract meta
	if unsavedContractMeta != nil {
		keyList = append(keyList, chain_utils.CreateContractMetaKey(address))
		valueList = append(valueList, unsavedContractMeta.Serialize())
	}

	// write log list
	logHash := block.AccountBlock.LogHash
	if logHash != nil {
		// insert vm log list
		keyList = append(keyList, chain_utils.CreateVmLogListKey(logHash))
		value, err := unsavedVmLogList.Serialize()
		if err != nil {
			return err
		}
		valueList = append(valueList, value)
	}

	if err := sDB.mvDB.Insert(&block.AccountBlock.Hash, keyList, valueList); err != nil {
		return err
	}
	return nil
}

func (sDB *StateDB) Flush(blocks []*ledger.AccountBlock, invalidAccountBlocks []*ledger.AccountBlock,
	latestLocation *chain_block.Location) error {

	blockHashList := make([]*types.Hash, 0, len(blocks))

	for _, block := range blocks {
		blockHashList = append(blockHashList, &block.Hash)
	}

	if err := sDB.flushLog.Write(blockHashList); err != nil {
		return err
	}

	if err := sDB.mvDB.Flush(blockHashList, latestLocation); err != nil {
		return err
	}

	for _, block := range invalidAccountBlocks {
		sDB.mvDB.DeletePendingBlock(&block.Hash)
	}

	return nil
}

func (sDB *StateDB) prepareWriteBalance(addr *types.Address, balanceMap map[types.TokenTypeId]*big.Int) ([][]byte, [][]byte) {
	keyList := make([][]byte, 0, len(balanceMap))
	valueList := make([][]byte, 0, len(balanceMap))

	for tokenTypeId, balance := range balanceMap {
		keyList = append(keyList, chain_utils.CreateBalanceKey(addr, &tokenTypeId))
		valueList = append(valueList, balance.Bytes())
	}
	return keyList, valueList
}

func (sDB *StateDB) prepareWriteStorage(addr *types.Address, unsavedStorage [][2][]byte, deletedKeys map[string]struct{}) ([][]byte, [][]byte) {
	keyList := make([][]byte, 0, len(unsavedStorage))
	valueList := make([][]byte, 0, len(unsavedStorage))

	for _, kv := range unsavedStorage {
		keyList = append(keyList, chain_utils.CreateStorageKeyPrefix(addr, kv[0]))
		valueList = append(valueList, kv[1])
	}

	for keyStr := range deletedKeys {
		keyList = append(keyList, chain_utils.CreateStorageKeyPrefix(addr, []byte(keyStr)))
		valueList = append(valueList, nil)
	}
	return keyList, valueList
}
