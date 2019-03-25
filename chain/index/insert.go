package chain_index

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) InsertAccountBlock(accountBlock *ledger.AccountBlock) error {
	blockHash := &accountBlock.Hash
	if ok, err := iDB.HasAccount(&accountBlock.AccountAddress); err != nil {
		return err
	} else if !ok {
		iDB.createAccount(blockHash, &accountBlock.AccountAddress)
	}
	// hash -> addr & height
	iDB.memDb.Put(blockHash, chain_utils.CreateAccountBlockHashKey(blockHash),
		append(accountBlock.AccountAddress.Bytes(), chain_utils.Uint64ToBytes(accountBlock.Height)...))

	if accountBlock.IsReceiveBlock() {
		if len(accountBlock.SendBlockList) > 0 {
			chain_utils.CreateCallDepthKey(&accountBlock.Hash)
		}

		// close send block
		iDB.memDb.Put(blockHash, chain_utils.CreateReceiveKey(&accountBlock.FromBlockHash), blockHash.Bytes())

		// receive on road
		if err := iDB.receiveOnRoad(blockHash, &accountBlock.FromBlockHash); err != nil {
			return err
		}
	} else {
		// insert on road block
		iDB.insertOnRoad(blockHash, &accountBlock.ToAddress)
	}

	for _, sendBlock := range accountBlock.SendBlockList {
		// insert on road block
		if err := iDB.InsertAccountBlock(sendBlock); err != nil {
			return err
		}
	}
	return nil
}

func (iDB *IndexDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock,
	confirmedBlocks []*ledger.AccountBlock,
	snapshotBlockLocation *chain_block.Location,
	abLocationsList []*chain_block.Location,
	invalidBlocks []*ledger.AccountBlock,
	latestLocation *chain_block.Location) error {

	batch := iDB.store.NewBatch()

	heightBytes := chain_utils.Uint64ToBytes(snapshotBlock.Height)
	// hash -> height
	batch.Put(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlock.Hash), heightBytes)

	// height -> location
	batch.Put(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlock.Height),
		append(snapshotBlock.Hash.Bytes(), chain_utils.SerializeLocation(snapshotBlockLocation)...))

	// confirm block
	for addr, hashHeight := range snapshotBlock.SnapshotContent {
		batch.Put(chain_utils.CreateConfirmHeightKey(&addr, hashHeight.Height), heightBytes)
	}

	// flush account block indexes
	for index, block := range confirmedBlocks {
		// height -> account block location
		batch.Put(chain_utils.CreateAccountBlockHeightKey(&block.AccountAddress, block.Height),
			append(block.Hash.Bytes(), chain_utils.SerializeLocation(abLocationsList[index])...))

		iDB.memDb.Flush(batch, &block.Hash)
	}

	// latest on road id
	batch.Put(chain_utils.CreateLatestOnRoadIdKey(), chain_utils.Uint64ToBytes(iDB.latestOnRoadId))

	// latest location
	iDB.setIndexDbLatestLocation(batch, latestLocation)

	// write index db
	if err := iDB.store.Write(batch); err != nil {
		return err
	}

	// clean mem store
	for _, accountBlock := range invalidBlocks {
		iDB.memDb.DeleteByBlockHash(&accountBlock.Hash)
	}

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
