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

func (iDB *IndexDB) RollbackSnapshotBlocks(deletedSnapshotSegments []*ledger.SnapshotChunk) error {
	batch := iDB.store.NewBatch()

	if err := iDB.rollback(batch, deletedSnapshotSegments); err != nil {
		return err
	}

	iDB.store.RollbackSnapshot(batch)
	return nil
}

func (iDB *IndexDB) DeleteOnRoad(sendBlockHash types.Hash) error {
	batch := iDB.store.NewBatch()
	if err := iDB.deleteOnRoad(batch, sendBlockHash); err != nil {
		return err
	}
	iDB.store.WriteDirectly(batch)
	return nil
}

func (iDB *IndexDB) rollback(batch *leveldb.Batch, deletedSnapshotSegments []*ledger.SnapshotChunk) error {
	openSendBlockHashMap := make(map[types.Hash]struct{})

	for _, seg := range deletedSnapshotSegments {
		iDB.deleteSnapshotBlock(batch, seg.SnapshotBlock)
		if err := iDB.deleteAccountBlocks(batch, seg.AccountBlocks, openSendBlockHashMap); err != nil {
			return err
		}
	}

	for sendBlockHash := range openSendBlockHashMap {
		iDB.deleteOnRoad(batch, sendBlockHash)
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

func (iDB *IndexDB) deleteAccountBlocks(batch *leveldb.Batch, blocks []*ledger.AccountBlock, sendBlockHashMap map[types.Hash]struct{}) error {
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
				if err := iDB.insertOnRoad(batch, block.FromBlockHash, block.AccountAddress); err != nil {
					return err
				}
			}
		} else {
			sendBlockHashMap[block.Hash] = struct{}{}
		}
		for _, sendBlock := range block.SendBlockList {
			// delete sendBlock hash index
			iDB.deleteAccountBlockHash(batch, sendBlock.Hash)

			// set open send
			sendBlockHashMap[sendBlock.Hash] = struct{}{}
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
