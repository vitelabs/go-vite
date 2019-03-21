package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/utils"
	"github.com/vitelabs/go-vite/vm_db"
)

func (iDB *IndexDB) InsertAccountBlock(vmAccountBlock *vm_db.VmAccountBlock) error {
	accountBlock := vmAccountBlock.AccountBlock

	blockHash := &accountBlock.Hash
	// hash -> addr & height
	iDB.memDb.Put(blockHash,
		chain_utils.CreateAccountBlockHashKey(blockHash),
		append(accountBlock.AccountAddress.Bytes(), chain_utils.Uint64ToFixedBytes(accountBlock.Height)...))

	if accountBlock.IsReceiveBlock() {
		// close send block
		iDB.memDb.Put(blockHash, chain_utils.CreateReceiveKey(blockHash), accountBlock.FromBlockHash.Bytes())

		// receive on road
		//if err := iDB.receiveOnRoad(blockHash, &accountBlock.FromBlockHash); err != nil {
		//	return err
		//}
	}
	//else {
	// insert on road block
	//iDB.insertOnRoad(blockHash, &accountBlock.ToAddress)
	//}

	if accountBlock.LogHash != nil {
		// insert vm log list
		vmLogList := vmAccountBlock.VmDb.GetLogList()
		iDB.insertVmLogList(blockHash, accountBlock.LogHash, vmLogList)
	}
	return nil
}

func (iDB *IndexDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock,
	confirmedBlocks []*ledger.AccountBlock,
	snapshotBlockLocation *chain_block.Location,
	abLocationsList []*chain_block.Location) error {

	batch := iDB.store.NewBatch()

	heightBytes := chain_utils.Uint64ToFixedBytes(snapshotBlock.Height)
	// hash -> height
	batch.Put(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash), heightBytes)

	// height -> location
	batch.Put(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height), chain_utils.SerializeLocation(snapshotBlockLocation))

	// confirm block
	for _, hashHeight := range snapshotBlock.SnapshotContent {
		batch.Put(chain_utils.CreateConfirmHeightKey(&hashHeight.Hash), heightBytes)
	}
	// flush account block indexes
	for index, block := range confirmedBlocks {
		// height -> account block location
		batch.Put(chain_utils.CreateAccountBlockHeightKey(&block.AccountAddress, block.Height),
			chain_utils.SerializeLocation(abLocationsList[index]))

		iDB.memDb.Flush(batch, &block.Hash)
	}

	if err := iDB.store.Write(batch); err != nil {
		return err
	}

	// clean mem store
	for _, block := range confirmedBlocks {
		iDB.memDb.DeleteByBlockHash(&block.Hash)
	}

	return nil
}

func (iDB *IndexDB) insertVmLogList(blockHash *types.Hash, logHash *types.Hash, logList ledger.VmLogList) error {
	value, err := logList.Serialize()

	if err != nil {
		return err
	}
	iDB.memDb.Put(blockHash, chain_utils.CreateVmLogListKey(logHash), value)
	return nil
}
