package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
	"github.com/vitelabs/go-vite/vm_db"
	"math/big"
)

func (sDB *StateDB) Write(block *vm_db.VmAccountBlock) error {
	accountId, err := sDB.chain.GetAccountId(&block.AccountBlock.AccountAddress)
	if err != nil {
		return err
	}

	vmDb := block.VmDb

	unsavedBalanceMap := vmDb.GetUnsavedBalanceMap()
	unsavedStorage := vmDb.GetUnsavedStorage()
	unsavedCode := vmDb.GetUnsavedContractCode()
	unsavedContractMeta := vmDb.GetUnsavedContractMeta()

	kvSize := len(unsavedBalanceMap) + len(unsavedStorage) + 2
	keyList := make([][]byte, 0, kvSize)
	valueList := make([][]byte, 0, kvSize)

	// write balance
	balanceKeyList, balanceValueList := sDB.prepareWriteBalance(accountId, unsavedBalanceMap)
	keyList = append(keyList, balanceKeyList...)
	valueList = append(valueList, balanceValueList...)

	// write storage
	storageKeyList, storageValueList := sDB.prepareWriteStorage(accountId, unsavedStorage)
	keyList = append(keyList, storageKeyList...)
	valueList = append(valueList, storageValueList...)

	// write code
	if len(unsavedCode) > 0 {
		keyList = append(keyList, chain_dbutils.CreateCodeKey(accountId))
		valueList = append(valueList, unsavedCode)
	}

	// write contract meta
	if unsavedContractMeta != nil {
		keyList = append(keyList, chain_dbutils.CreateContractMetaKey(accountId))
		valueList = append(valueList, unsavedContractMeta.Serialize())
	}

	if err := sDB.mvDB.Insert(&block.AccountBlock.Hash, keyList, valueList); err != nil {
		return err
	}
	return nil
}

func (sDB *StateDB) Flush(snapshotBlockHash *types.Hash, blocks []*ledger.AccountBlock) error {
	blockHashList := make([]*types.Hash, 0, len(blocks))

	for _, block := range blocks {
		blockHashList = append(blockHashList, &block.Hash)
	}
	if err := sDB.mvDB.Flush(blockHashList); err != nil {
		return err
	}

	return nil
}

func (sDB *StateDB) prepareWriteBalance(accountId uint64, balanceMap map[types.TokenTypeId]*big.Int) ([][]byte, [][]byte) {
	keyList := make([][]byte, 0, len(balanceMap))
	valueList := make([][]byte, 0, len(balanceMap))

	for tokenTypeId, balance := range balanceMap {
		keyList = append(keyList, chain_dbutils.CreateBalanceKey(accountId, &tokenTypeId))
		valueList = append(valueList, balance.Bytes())

	}
	return keyList, valueList
}

func (sDB *StateDB) prepareWriteStorage(accountId uint64, unsavedStorage [][2][]byte) ([][]byte, [][]byte) {
	keyList := make([][]byte, 0, len(unsavedStorage))
	valueList := make([][]byte, 0, len(unsavedStorage))

	for _, kv := range unsavedStorage {
		keyList = append(keyList, chain_dbutils.CreateStorageKeyPrefix(accountId, kv[0]))
		valueList = append(valueList, kv[1])
	}
	return keyList, valueList
}
