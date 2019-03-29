package chain_state

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/ledger"
)

// TODO
func (sDB *StateDB) Rollback(deletedSnapshotSegments []*chain_block.SnapshotSegment, toLocation *chain_file_manager.Location) error {
	batch := sDB.store.NewBatch()
	//blockHashList := make([]*types.Hash, 0, size)

	var firstSb = deletedSnapshotSegments[0].SnapshotBlock
	if firstSb == nil {
		firstSb = sDB.chain.GetLatestSnapshotBlock()
	}
	for _, seg := range deletedSnapshotSegments {
		if seg.SnapshotBlock != nil {

		}
		for _, accountBlock := range seg.AccountBlocks {

			// delete code
			if accountBlock.Height <= 1 {
				batch.Delete(chain_utils.CreateCodeKey(accountBlock.AccountAddress))
			}

			// delete contract meta
			if accountBlock.BlockType == ledger.BlockTypeSendCreate {
				batch.Delete(chain_utils.CreateContractMetaKey(accountBlock.AccountAddress))
			}

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

	//sDB.updateStateDbLocation(batch, location)

	//if err := sDB.store.Write(batch, nil); err != nil {
	//	return err
	//}
	//
	//if err := sDB.undoLogger.DeleteTo(location); err != nil {
	//	return err
	//}

	sDB.store.Write(batch)
	return nil
}
