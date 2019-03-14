package chain_index

import (
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

/*
 *	TODO
 *	1. accountId
 *	2. sendAccountId
 *	3. sendHeight
 *	4. toAccountId
 *	5. new account
 */
func (iDB *IndexDB) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) {
	accountBlock := vmAccountBlock.AccountBlock

	blockHash := &accountBlock.Hash
	accountId := uint64(1)
	// hash -> accountId & height
	iDB.insertAccountBlockHash(blockHash, accountId, accountBlock.Height)

	//if err := insertAccountBlockHeight(batch, accountId, accountBlock.Height, ""); err != nil {
	//	return err
	//}
	if accountBlock.IsReceiveBlock() {
		// close send block
		sendAccountId := uint64(2)
		sendHeight := uint64(12)
		iDB.insertReceiveHeight(blockHash, sendAccountId, sendHeight, accountId, accountBlock.Height)
	} else {
		// insert on road block
		toAccountId := uint64(3)
		iDB.insertOnRoad(blockHash, toAccountId, accountId, accountBlock.Height)
	}

	if accountBlock.LogHash != nil {
		// insert vm log list
		vmLogList := vmAccountBlock.VmDb.GetLogList()
		iDB.insertVmLogList(blockHash, accountBlock.LogHash, vmLogList)
	}
}

/*
 *	TODO
 *	1. location
 *	2. accountId
 */
func (iDB *IndexDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, confirmedSubLedger map[types.Address][]*ledger.AccountBlock) error {
	batch := iDB.store.NewBatch()
	// hash -> height
	if err := iDB.insertSnapshotBlockHash(batch, &snapshotBlock.Hash, snapshotBlock.Height); err != nil {
		return err
	}

	// height -> location
	if err := iDB.insertSnapshotBlockHeight(batch, snapshotBlock.Height, ""); err != nil {
		return err
	}
	// confirm block
	for _, hashHeight := range snapshotBlock.SnapshotContent {
		accountId := uint64(1)
		if err := iDB.insertConfirmHeight(batch, accountId, hashHeight.Height, snapshotBlock.Height); err != nil {
			return err
		}
	}
	// flush account block indexes
	for _, blocks := range confirmedSubLedger {
		iDB.flush(batch, blocks)
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

func (iDB *IndexDB) insertAccount(batch Batch, addr *types.Address, accountId uint64) error {
	return nil
}

func (iDB *IndexDB) insertAccountBlockHash(blockHash *types.Hash, accountId uint64, height uint64) {
	key, _ := dbutils.EncodeKey(AccountBlockHashKeyPrefix, blockHash.Bytes())
	value := SerializeAccountIdHeight(accountId, height)

	iDB.memDb.Put(blockHash, key, value)

}

func (iDB *IndexDB) insertAccountBlockHeight(batch Batch, accountId uint64, height uint64, location string) error {
	return nil
}

func (iDB *IndexDB) insertReceiveHeight(blockHash *types.Hash, sendAccountId, sendHeight, receiveAccountId, receiveHeight uint64) {
	key, _ := dbutils.EncodeKey(ReceiveHeightKeyPrefix, sendAccountId, sendHeight)
	value := SerializeAccountIdHeight(receiveAccountId, receiveHeight)

	iDB.memDb.Put(blockHash, key, value)
}

/*
 *	TODO
 *	1. 自增ID
 */

func (iDB *IndexDB) insertOnRoad(blockHash *types.Hash, toAccountId, sendAccountId, sendHeight uint64) {
	id := uint64(13)
	key, _ := dbutils.EncodeKey(OnRoadKeyPrefix, toAccountId, id)
	value := SerializeAccountIdHeight(sendAccountId, sendHeight)
	iDB.memDb.Put(blockHash, key, value)
}

func (iDB *IndexDB) insertVmLogList(blockHash *types.Hash, logHash *types.Hash, logList ledger.VmLogList) error {
	key, _ := dbutils.EncodeKey(VmLogListKeyPrefix, logHash.Bytes())

	value, err := logList.Serialize()

	if err != nil {
		return err
	}
	iDB.memDb.Put(blockHash, key, value)
	return nil
}

// TODO write failed
func (iDB *IndexDB) flush(batch Batch, blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		keyList, valueList := iDB.memDb.GetByBlockHash(&block.Hash)
		if len(keyList) > 0 {
			for index, key := range keyList {
				batch.Put(key, valueList[index])
			}
		}
	}
}

func (iDB *IndexDB) cleanMemDb(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		iDB.memDb.DeleteByBlockHash(&block.Hash)
	}
}

func (iDB *IndexDB) insertConfirmHeight(batch Batch, accountId uint64, height uint64, snapshotHeight uint64) error {
	return nil
}

func (iDB *IndexDB) insertSnapshotBlockHash(batch Batch, snapshotBlockHash *types.Hash, height uint64) error {
	return nil
}

func (iDB *IndexDB) insertSnapshotBlockHeight(batch Batch, snapshotBlockHeight uint64, location string) error {
	return nil
}

func SerializeAccountIdHeight(accountId, height uint64) []byte {
	return nil
}

func DeserializeAccountIdHeight(accountId, height uint64) []byte {
	return nil
}
