package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) Rollback(deletedSnapshotSegments []*ledger.SnapshotChunk) error {
	if len(deletedSnapshotSegments) <= 0 {
		return nil
	}

	batch := iDB.store.NewBatch()

	openSendBlockHashMap := make(map[types.Hash]struct{})

	for _, seg := range deletedSnapshotSegments {
		iDB.deleteSnapshotBlock(batch, seg.SnapshotBlock)
		iDB.deleteAccountBlocks(batch, seg.AccountBlocks, openSendBlockHashMap)
	}

	for sendBlockHash := range openSendBlockHashMap {
		if err := iDB.deleteOnRoad(batch, sendBlockHash); err != nil {
			return err
		}
	}

	iDB.store.Write(batch)
	return nil
}

func (iDB *IndexDB) DeleteOnRoad(sendBlockHash types.Hash) error {
	batch := iDB.store.NewBatch()
	if err := iDB.deleteOnRoad(batch, sendBlockHash); err != nil {
		return err
	}
	iDB.store.Write(batch)
	return nil
}

func (iDB *IndexDB) deleteSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock) {
	if snapshotBlock != nil {
		iDB.deleteSnapshotBlockHash(batch, snapshotBlock.Hash)
		iDB.deleteSnapshotBlockHeight(batch, snapshotBlock.Height)
	}
}

func (iDB *IndexDB) deleteAccountBlocks(batch *leveldb.Batch, blocks []*ledger.AccountBlock, openSendBlockHashMap map[types.Hash]struct{}) {
	for _, block := range blocks {
		// delete account block hash index
		iDB.deleteAccountBlockHash(batch, block.Hash)

		// delete account block height index
		iDB.deleteAccountBlockHeight(batch, block.AccountAddress, block.Height)

		// delete confirmed index
		iDB.deleteConfirmHeight(batch, block.AccountAddress, block.Height)

		if block.IsReceiveBlock() {
			// delete receive index
			iDB.deleteReceive(batch, block.FromBlockHash)

			if _, ok := openSendBlockHashMap[block.FromBlockHash]; !ok {

				delete(openSendBlockHashMap, block.FromBlockHash)

				// insert onRoad
				iDB.insertOnRoad(batch, block.FromBlockHash, block.AccountAddress)
			}
		} else {
			openSendBlockHashMap[block.Hash] = struct{}{}
		}
		for _, sendBlock := range block.SendBlockList {
			// delete sendBlock hash index
			iDB.deleteAccountBlockHash(batch, sendBlock.Hash)

			// set open send
			openSendBlockHashMap[sendBlock.Hash] = struct{}{}
		}

	}

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
