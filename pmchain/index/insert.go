package chain_index

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/utils"
	"github.com/vitelabs/go-vite/vm_db"
)

func (iDB *IndexDB) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) error {
	accountBlock := vmAccountBlock.AccountBlock

	blockHash := &accountBlock.Hash
	accountId, err := iDB.GetAccountId(&accountBlock.AccountAddress)
	if err != nil {
		return errors.New(fmt.Sprintf("iDB.GetAccountId failed, error is %s", err.Error()))
	}
	if accountId <= 0 {
		// need create account
		accountId = iDB.createAccount(blockHash, &accountBlock.AccountAddress)
	}
	// hash -> accountId & height
	iDB.insertAccountBlockHash(blockHash, accountId, accountBlock.Height)

	if accountBlock.IsReceiveBlock() {
		// close send block
		sendAccountId, sendHeight, err := iDB.getAccountIdHeight(&accountBlock.FromBlockHash)
		if err != nil {
			return errors.New(fmt.Sprintf("iDB.GetAccountHeightByHash failed, error is %s, accountBlock.FromBlockHash is %s", err.Error(), accountBlock.FromBlockHash))
		}

		iDB.insertReceiveHeight(blockHash, sendAccountId, sendHeight, accountId, accountBlock.Height)

		// receive on road
		if err := iDB.receiveOnRoad(blockHash, &accountBlock.FromBlockHash); err != nil {
			return err
		}
	} else {
		// insert on road block
		toAccountId, err := iDB.GetAccountId(&accountBlock.ToAddress)
		if err != nil {
			return errors.New(fmt.Sprintf("iDB.GetAccountId failed, error is %s, toAccountId is %d", err.Error(), toAccountId))
		}

		if toAccountId <= 0 {
			// need create account
			toAccountId = iDB.createAccount(blockHash, &accountBlock.ToAddress)
		}

		iDB.insertOnRoad(blockHash, toAccountId)
	}

	if accountBlock.LogHash != nil {
		// insert vm log list
		vmLogList := vmAccountBlock.VmDb.GetLogList()
		iDB.insertVmLogList(blockHash, accountBlock.LogHash, vmLogList)
	}
	return nil
}

func (iDB *IndexDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedSubLedger map[types.Address][]*ledger.AccountBlock, sbLocation *chain_block.Location, subLedgerLocations map[types.Address][]*chain_block.Location) error {
	batch := iDB.store.NewBatch()
	// hash -> height
	iDB.insertSnapshotBlockHash(batch, &snapshotBlock.Hash, snapshotBlock.Height)

	// height -> location
	iDB.insertSnapshotBlockHeight(batch, snapshotBlock.Height, sbLocation)

	// confirm block
	for addr, hashHeight := range snapshotBlock.SnapshotContent {
		accountId, err := iDB.GetAccountId(&addr)
		if err != nil {
			return errors.New(fmt.Sprintf("iDB.GetAccountId failed, error is %s", err.Error()))
		}

		iDB.insertConfirmHeight(batch, accountId, hashHeight.Height, snapshotBlock.Height)
	}
	// flush account block indexes
	for addr, blocks := range confirmedSubLedger {

		accountId, err := iDB.GetAccountId(&addr)
		if err != nil {
			return errors.New(fmt.Sprintf("iDB.GetAccountId failed, error is %s", err.Error()))
		}
		locations := subLedgerLocations[addr]
		// height -> account block location

		blockHashList := make([]*types.Hash, 0, len(blocks))
		for index, block := range blocks {
			blockHashList = append(blockHashList, &block.Hash)
			iDB.insertAccountBlockHeight(batch, accountId, block.Height, locations[index])
		}

		iDB.flush(batch, blockHashList)

	}

	if err := iDB.store.Write(batch); err != nil {
		return err
	}
	// clean mem store
	for _, blocks := range confirmedSubLedger {
		iDB.cleanMemDb(blocks)
	}

	return nil
}

func (iDB *IndexDB) insertAccountBlockHash(blockHash *types.Hash, accountId uint64, height uint64) {
	key := chain_utils.CreateAccountBlockHashKey(blockHash)
	value := chain_utils.SerializeAccountIdHeight(accountId, height)

	iDB.memDb.Put(blockHash, key, value)

}

func (iDB *IndexDB) insertAccountBlockHeight(batch interfaces.Batch, accountId uint64, height uint64, location *chain_block.Location) {
	key := chain_utils.CreateAccountBlockHeightKey(accountId, height)
	value := chain_utils.SerializeLocation(location)
	batch.Put(key, value)
}

func (iDB *IndexDB) insertReceiveHeight(blockHash *types.Hash, sendAccountId, sendHeight, receiveAccountId, receiveHeight uint64) {
	key := make([]byte, 0, 17)
	key = chain_utils.CreateReceiveHeightKey(sendAccountId, sendHeight)

	value := chain_utils.SerializeAccountIdHeight(receiveAccountId, receiveHeight)

	iDB.memDb.Put(blockHash, key, value)
}

func (iDB *IndexDB) insertVmLogList(blockHash *types.Hash, logHash *types.Hash, logList ledger.VmLogList) error {
	key := chain_utils.CreateVmLogListKey(logHash)

	value, err := logList.Serialize()

	if err != nil {
		return err
	}
	iDB.memDb.Put(blockHash, key, value)
	return nil
}

func (iDB *IndexDB) flush(batch interfaces.Batch, blockHashList []*types.Hash) {
	iDB.memDb.Flush(batch, blockHashList)
}

func (iDB *IndexDB) cleanMemDb(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		iDB.memDb.DeleteByBlockHash(&block.Hash)
	}
}

func (iDB *IndexDB) insertConfirmHeight(batch interfaces.Batch, accountId uint64, height uint64, snapshotHeight uint64) {
	key := chain_utils.CreateConfirmHeightKey(accountId, height)
	batch.Put(key, chain_utils.SerializeHeight(snapshotHeight))
}

func (iDB *IndexDB) insertSnapshotBlockHash(batch interfaces.Batch, snapshotBlockHash *types.Hash, height uint64) {
	key := chain_utils.CreateSnapshotBlockHashKey(snapshotBlockHash)
	batch.Put(key, chain_utils.SerializeHeight(height))
}

func (iDB *IndexDB) insertSnapshotBlockHeight(batch interfaces.Batch, snapshotBlockHeight uint64, location *chain_block.Location) {
	key := chain_utils.CreateSnapshotBlockHeightKey(snapshotBlockHeight)
	batch.Put(key, chain_utils.SerializeLocation(location))
}
