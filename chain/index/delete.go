package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) RollbackAccountBlocks(accountBlocks []*ledger.AccountBlock) error {
	batch := iDB.store.NewBatch()
	if err := iDB.rollback(batch, []*ledger.SnapshotChunk{{
		AccountBlocks: accountBlocks,
	}}); err != nil {
		return err
	}

	iDB.store.RollbackAccountBlocks(batch, accountBlocks)
	return nil
}

func (iDB *IndexDB) RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk, unconfirmedBlocks []*ledger.AccountBlock) error {
	batch := iDB.store.NewBatch()

	err := iDB.rollback(batch, deletedSnapshotSegments)

	if err != nil {
		return err
	}

	iDB.store.RollbackSnapshot(batch)

	// recover unconfirmed
	for _, block := range unconfirmedBlocks {
		if err := iDB.InsertAccountBlock(block); err != nil {
			return err
		}
	}

	return nil
}

func (iDB *IndexDB) DeleteOnRoad(toAddress types.Address, sendBlockHash types.Hash) {
	batch := iDB.store.NewBatch()
	iDB.deleteOnRoad(batch, toAddress, sendBlockHash)
	iDB.store.WriteDirectly(batch)
}

func (iDB *IndexDB) rollback(batch *leveldb.Batch, deletedSnapshotSegments []*ledger.SnapshotChunk) error {
	openSendBlock := make(map[types.Hash]*ledger.AccountBlock)

	for _, seg := range deletedSnapshotSegments {
		if err := iDB.deleteAccountBlocks(batch, seg.AccountBlocks, openSendBlock); err != nil {
			return err
		}
		iDB.deleteSnapshotBlock(batch, seg.SnapshotBlock)
	}

	for sendBlockHash, sendBlock := range openSendBlock {
		iDB.deleteOnRoad(batch, sendBlock.ToAddress, sendBlockHash)
	}
	return nil
}

func (iDB *IndexDB) deleteSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock) {
	if snapshotBlock != nil {
		iDB.deleteSnapshotBlockHash(batch, snapshotBlock.Hash)
		iDB.deleteSnapshotBlockHeight(batch, snapshotBlock.Height)

		// delete confirmed index
		for addr, hashHeight := range snapshotBlock.SnapshotContent {
			iDB.deleteConfirmHeight(batch, addr, hashHeight.Height)
		}
	}
}

func (iDB *IndexDB) deleteAccountBlocks(batch *leveldb.Batch, blocks []*ledger.AccountBlock, sendBlockHashMap map[types.Hash]*ledger.AccountBlock) error {
	for _, block := range blocks {
		// delete account block hash index
		iDB.deleteAccountBlockHash(batch, block.Hash)

		// delete account block height index
		iDB.deleteAccountBlockHeight(batch, block.AccountAddress, block.Height)

		if block.IsReceiveBlock() {
			// delete receive index
			iDB.deleteReceive(batch, block.FromBlockHash)

			if _, ok := sendBlockHashMap[block.FromBlockHash]; ok {
				delete(sendBlockHashMap, block.FromBlockHash)
			} else {
				// insert onRoad
				iDB.insertOnRoad(batch, block.AccountAddress, block.FromBlockHash)
			}
		} else {
			sendBlockHashMap[block.Hash] = block
		}
		for _, sendBlock := range block.SendBlockList {
			// delete sendBlock hash index
			iDB.deleteAccountBlockHash(batch, sendBlock.Hash)

			// set open send
			sendBlockHashMap[sendBlock.Hash] = sendBlock
		}

	}
	return nil

}

func (iDB *IndexDB) deleteSnapshotBlockHash(batch *leveldb.Batch, snapshotBlockHash types.Hash) {
	batch.Delete(chain_utils.CreateSnapshotBlockHashKey(&snapshotBlockHash))
}

func (iDB *IndexDB) deleteSnapshotBlockHeight(batch *leveldb.Batch, snapshotBlockHeight uint64) {
	batch.Delete(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlockHeight))
}

func (iDB *IndexDB) deleteAccountBlockHash(batch *leveldb.Batch, accountBlockHash types.Hash) {
	batch.Delete(chain_utils.CreateAccountBlockHashKey(&accountBlockHash))
}

func (iDB *IndexDB) deleteAccountBlockHeight(batch *leveldb.Batch, addr types.Address, height uint64) {
	batch.Delete(chain_utils.CreateAccountBlockHeightKey(&addr, height))
}

func (iDB *IndexDB) deleteReceive(batch *leveldb.Batch, sendBlockHash types.Hash) {
	batch.Delete(chain_utils.CreateReceiveKey(&sendBlockHash))
}

func (iDB *IndexDB) deleteConfirmHeight(batch *leveldb.Batch, addr types.Address, height uint64) {
	batch.Delete(chain_utils.CreateConfirmHeightKey(&addr, height))
}
