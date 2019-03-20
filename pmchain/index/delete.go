package chain_index

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/utils"
)

func (iDB *IndexDB) DeleteInvalidAccountBlocks(invalidSubLedger map[types.Address][]*ledger.AccountBlock) {
	for _, blocks := range invalidSubLedger {
		for _, accountBlock := range blocks {
			iDB.memDb.DeleteByBlockHash(&accountBlock.Hash)
		}
	}
}

// TODO
func (iDB *IndexDB) DeleteSnapshotBlocks(deletedSnapshotSegments []*chain_block.SnapshotSegment, unconfirmedBlocks []*ledger.AccountBlock) error {
	batch := iDB.store.NewBatch()

	deletedHashMap := make(map[types.Hash]struct{})

	for _, seg := range deletedSnapshotSegments {
		snapshotBlock := seg.SnapshotBlock
		iDB.deleteSnapshotBlockHash(batch, &snapshotBlock.Hash)
		iDB.deleteSnapshotBlockHeight(batch, snapshotBlock.Height)

		for _, block := range seg.AccountBlocks {
			deletedHashMap[block.Hash] = struct{}{}
			// delete account block hash index
			iDB.deleteAccountBlockHash(batch, &block.Hash)

			accountId, err := iDB.GetAccountId(&block.AccountAddress)
			if err != nil {
				return err
			}
			if accountId <= 0 {
				return errors.New(fmt.Sprintf("accountId is 0, block is %+v", block))
			}
			// delete account block height index
			iDB.deleteAccountBlockHeight(batch, accountId, block.Height)

			// delete confirmed index
			iDB.deleteConfirmHeight(batch, accountId, block.Height)

			if block.IsReceiveBlock() {
				// delete send block close index
				if _, ok := deletedHashMap[block.FromBlockHash]; !ok {
					sendAccountId, sendHeight, err := iDB.getAccountIdHeight(&block.FromBlockHash)
					if err != nil {
						return err
					}
					if sendAccountId <= 0 {
						return errors.New(fmt.Sprintf("sendAccountId is 0, block is %+v", block))
					}

					iDB.deleteReceiveHeight(batch, sendAccountId, sendHeight)

					// insert onRoad
					if err := iDB.insertOnRoad(&block.FromBlockHash, accountId); err != nil {
						return err
					}
				}
			} else {
				// remove on road
				//toAccountId, err := iDB.GetAccountId(&block.ToAddress)
				//if err != nil {
				//	return err
				//}
				//if toAccountId <= 0 {
				//	return errors.New(fmt.Sprintf("toAccountId is 0, block is %+v", block))
				//}
				//iDB.deleteOnRoad(batch, toAccountId, accountId, block.Height)
			}

			if block.LogHash != nil {
				// delete vm log list
				iDB.deleteVmLogList(batch, block.LogHash)
			}

		}
	}

	for _, block := range unconfirmedBlocks {
		// remove confirmed indexes
		accountId, err := iDB.GetAccountId(&block.AccountAddress)
		if err != nil {
			return err
		}
		if accountId <= 0 {
			return errors.New(fmt.Sprintf("accountId is 0, block is %+v", block))
		}

		iDB.deleteConfirmHeight(batch, accountId, block.Height)
	}
	if err := iDB.store.Write(batch); err != nil {
		return err
	}
	// clean mem
	iDB.memDb.Clean()

	return nil

}

func (iDB *IndexDB) deleteSnapshotBlockHash(batch interfaces.Batch, snapshotBlockHash *types.Hash) {
	key := chain_utils.CreateSnapshotBlockHashKey(snapshotBlockHash)
	batch.Delete(key)
}

func (iDB *IndexDB) deleteSnapshotBlockHeight(batch interfaces.Batch, snapshotBlockHeight uint64) {
	key := chain_utils.CreateSnapshotBlockHeightKey(snapshotBlockHeight)
	batch.Delete(key)
}

func (iDB *IndexDB) deleteAccountBlockHash(batch interfaces.Batch, blockHash *types.Hash) {
	key := chain_utils.CreateAccountBlockHashKey(blockHash)
	batch.Delete(key)
}

func (iDB *IndexDB) deleteAccountBlockHeight(batch interfaces.Batch, accountId uint64, height uint64) {
	key := chain_utils.CreateAccountBlockHeightKey(accountId, height)
	batch.Delete(key)
}

func (iDB *IndexDB) deleteReceiveHeight(batch interfaces.Batch, sendAccountId, sendHeight uint64) {
	key := chain_utils.CreateReceiveHeightKey(sendAccountId, sendHeight)
	batch.Delete(key)
}

func (iDB *IndexDB) deleteVmLogList(batch interfaces.Batch, logHash *types.Hash) {
	key := chain_utils.CreateVmLogListKey(logHash)
	batch.Delete(key)
}

func (iDB *IndexDB) deleteConfirmHeight(batch interfaces.Batch, accountId uint64, height uint64) {
	key := chain_utils.CreateConfirmHeightKey(accountId, height)
	batch.Delete(key)
}
