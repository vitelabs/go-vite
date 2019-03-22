package chain_index

import (
	"github.com/vitelabs/go-vite/chain/block"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
)

// TODO
func (iDB *IndexDB) DeleteSnapshotBlocks(deletedSnapshotSegments []*chain_block.SnapshotSegment,
	unconfirmedBlocks []*ledger.AccountBlock) error {
	//batch := iDB.store.NewBatch()
	//
	//deletedHashMap := make(map[types.Hash]struct{})
	//needDeleteOnRoadHashMap := make(map[types.Hash]struct{})
	//
	//for _, seg := range deletedSnapshotSegments {
	//	snapshotBlock := seg.SnapshotBlock
	//	iDB.deleteSnapshotBlockHash(batch, &snapshotBlock.Hash)
	//	iDB.deleteSnapshotBlockHeight(batch, snapshotBlock.Height)
	//
	//	for _, block := range seg.AccountBlocks {
	//		deletedHashMap[block.Hash] = struct{}{}
	//		// delete account block hash index
	//		iDB.deleteAccountBlockHash(batch, &block.Hash)
	//
	//		accountId, err := iDB.GetAccountId(&block.AccountAddress)
	//		if err != nil {
	//			return err
	//		}
	//		if accountId <= 0 {
	//			return errors.New(fmt.Sprintf("accountId is 0, block is %+v", block))
	//		}
	//		// delete account block height index
	//		iDB.deleteAccountBlockHeight(batch, accountId, block.Height)
	//
	//		// delete confirmed index
	//		iDB.deleteConfirmHeight(batch, accountId, block.Height)
	//
	//		if block.IsReceiveBlock() {
	//			// delete send block close index
	//			delete(needDeleteOnRoadHashMap, block.FromBlockHash)
	//			if _, ok := deletedHashMap[block.FromBlockHash]; !ok {
	//				sendAccountId, sendHeight, err := iDB.getAccountIdHeight(&block.FromBlockHash)
	//				if err != nil {
	//					return err
	//				}
	//				if sendAccountId <= 0 {
	//					return errors.New(fmt.Sprintf("sendAccountId is 0, block is %+v", block))
	//				}
	//
	//				iDB.deleteReceive(batch, sendAccountId, sendHeight)
	//
	//				// insert onRoad
	//				if err := iDB.insertOnRoad(&block.FromBlockHash, accountId); err != nil {
	//					return err
	//				}
	//			}
	//		} else {
	//			needDeleteOnRoadHashMap[block.Hash] = struct{}{}
	//		}
	//
	//		if block.LogHash != nil {
	//			// delete vm log list
	//			iDB.deleteVmLogList(batch, block.LogHash)
	//		}
	//
	//	}
	//}
	//
	//for hash := range needDeleteOnRoadHashMap {
	//	iDB.deleteOnRoad(batch, &hash)
	//}
	//
	//for _, block := range unconfirmedBlocks {
	//	// remove confirmed indexes
	//	accountId, err := iDB.GetAccountId(&block.AccountAddress)
	//	if err != nil {
	//		return err
	//	}
	//	if accountId <= 0 {
	//		return errors.New(fmt.Sprintf("accountId is 0, block is %+v", block))
	//	}
	//
	//	iDB.deleteConfirmHeight(batch, accountId, block.Height)
	//}
	//if err := iDB.store.Write(batch); err != nil {
	//	return err
	//}
	//// clean mem
	//iDB.memDb.Clean()

	return nil
}

func (iDB *IndexDB) setIndexDbLatestLocation(batch interfaces.Batch, latestLocation *chain_block.Location) {
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

func (iDB *IndexDB) deleteConfirmHeight(batch interfaces.Batch, hash *types.Hash) {
	batch.Delete(chain_utils.CreateConfirmHeightKey(hash))
}
