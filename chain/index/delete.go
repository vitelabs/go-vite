package chain_index

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

// TODO
func (iDB *IndexDB) Rollback(deletedSnapshotSegments []*chain_block.SnapshotSegment,
	location *chain_file_manager.Location) error {
	// clean mem
	iDB.memDb.Clean()

	batch := iDB.store.NewBatch()

	openSendBlockHashMap := make(map[types.Hash]struct{})

	for _, seg := range deletedSnapshotSegments {
		snapshotBlock := seg.SnapshotBlock
		iDB.deleteSnapshotBlockHash(batch, &snapshotBlock.Hash)
		iDB.deleteSnapshotBlockHeight(batch, snapshotBlock.Height)

		for _, block := range seg.AccountBlocks {
			// delete account block hash index
			iDB.deleteAccountBlockHash(batch, &block.Hash)

			// delete account block height index
			iDB.deleteAccountBlockHeight(batch, &block.AccountAddress, block.Height)

			// delete confirmed index
			iDB.deleteConfirmHeight(batch, &block.AccountAddress, block.Height)

			if block.IsReceiveBlock() {

				if _, ok := openSendBlockHashMap[block.FromBlockHash]; !ok {
					delete(openSendBlockHashMap, block.FromBlockHash)

					// delete receive index
					iDB.deleteReceive(batch, &block.FromBlockHash)

					// insert onRoad
					iDB.insertOnRoad(&block.FromBlockHash, &block.AccountAddress)
				}
			} else {
				openSendBlockHashMap[block.Hash] = struct{}{}
			}
		}
	}

	for sendBlockHash := range openSendBlockHashMap {
		if err := iDB.deleteOnRoad(batch, &sendBlockHash); err != nil {
			return err
		}
	}

	// set latest location
	iDB.setIndexDbLatestLocation(batch, location)
	if err := iDB.store.Write(batch); err != nil {
		return err
	}

	return nil
}

func (iDB *IndexDB) setIndexDbLatestLocation(batch interfaces.Batch, latestLocation *chain_file_manager.Location) {
	batch.Put(chain_utils.CreateIndexDbLatestLocationKey(), chain_utils.SerializeLocation(latestLocation))

}

func (iDB *IndexDB) deleteSnapshotBlockHash(batch interfaces.Batch, snapshotBlockHash *types.Hash) {
	batch.Delete(chain_utils.CreateSnapshotBlockHashKey(snapshotBlockHash))
}

func (iDB *IndexDB) deleteSnapshotBlockHeight(batch interfaces.Batch, snapshotBlockHeight uint64) {
	batch.Delete(chain_utils.CreateSnapshotBlockHeightKey(snapshotBlockHeight))
}

func (iDB *IndexDB) deleteAccountBlockHash(batch interfaces.Batch, blockHash *types.Hash) {
	batch.Delete(chain_utils.CreateAccountBlockHashKey(blockHash))
}

func (iDB *IndexDB) deleteAccountBlockHeight(batch interfaces.Batch, addr *types.Address, height uint64) {
	batch.Delete(chain_utils.CreateAccountBlockHeightKey(addr, height))
}

func (iDB *IndexDB) deleteReceive(batch interfaces.Batch, sendBlockHash *types.Hash) {
	batch.Delete(chain_utils.CreateReceiveKey(sendBlockHash))
}

func (iDB *IndexDB) deleteConfirmHeight(batch interfaces.Batch, addr *types.Address, height uint64) {
	batch.Delete(chain_utils.CreateConfirmHeightKey(addr, height))
}
