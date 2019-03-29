package chain_state

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
)

func (sDB *StateDB) Rollback(deletedSnapshotSegments []*chain_block.SnapshotSegment, toLocation *chain_file_manager.Location) error {
	batch := new(leveldb.Batch)
	//blockHashList := make([]*types.Hash, 0, size)

	// undo balance and storage

	location, err := sDB.undo(batch, sDB.chain.GetLatestSnapshotBlock())
	if err != nil {
		return err
	}

	sDB.updateUndoLocation(batch, location)

	for _, seg := range deletedSnapshotSegments {
		for _, accountBlock := range seg.AccountBlocks {
			// delete log hash
			if accountBlock.LogHash != nil {
				batch.Delete(chain_utils.CreateVmLogListKey(accountBlock.LogHash))
			}

			// delete call depth
			if accountBlock.IsReceiveBlock() {
				for _, sendBlock := range accountBlock.SendBlockList {
					batch.Delete(chain_utils.CreateCallDepthKey(&sendBlock.Hash))
				}
			}
		}
	}

	sDB.updateStateDbLocation(batch, location)

	//if err := sDB.store.Write(batch, nil); err != nil {
	//	return err
	//}

	if err := sDB.undoLogger.DeleteTo(location); err != nil {
		return err
	}

	return nil
}

func (sDB *StateDB) deleteVmLogList(batch *leveldb.Batch, logHash *types.Hash) {
	batch.Delete(chain_utils.CreateVmLogListKey(logHash))
}

func (sDB *StateDB) deleteCode(batch *leveldb.Batch, addr *types.Address) {
	batch.Delete(chain_utils.CreateCodeKey(addr))
}

func (sDB *StateDB) deleteContractMeta(batch *leveldb.Batch, addr *types.Address) {
	batch.Delete(chain_utils.CreateContractMetaKey(addr))
}
