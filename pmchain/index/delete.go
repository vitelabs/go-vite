package chain_index

import (
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (iDB *IndexDB) DeleteSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock, subLedgerNeedDeleted map[types.Address][]*ledger.AccountBlock, subLedgerNeedUnconfirmed map[types.Address][]*ledger.AccountBlock) error {
	batch := iDB.store.NewBatch()

	for _, snapshotBlock := range snapshotBlocks {
		iDB.deleteSnapshotBlockHash(batch, &snapshotBlock.Hash)
		iDB.deleteSnapshotBlockHeight(batch, snapshotBlock.Height)
	}

	for _, blocks := range subLedgerNeedDeleted {
		for _, block := range blocks {
			iDB.deleteAccountBlockHash(batch, &block.Hash)

			accountId := uint64(12)
			iDB.deleteAccountBlockHeight(batch, accountId, block.Height)

			if block.IsReceiveBlock() {
				// delete send block close index
				sendAccountId := uint64(14)
				sendHeight := uint64(20)
				iDB.deleteReceiveHeight(batch, sendAccountId, sendHeight)

				// insert onRoad
				if false {
					iDB.insertOnRoad(&block.FromBlockHash, accountId, sendAccountId, sendHeight)
				}
			} else {
				// remove on road
				toAccountId := uint64(3232)
				iDB.deleteOnRoad(batch, toAccountId, accountId, block.Height)
			}

			if block.LogHash != nil {
				// delete vm log list
				iDB.deleteVmLogList(batch, block.LogHash)
			}
		}
	}

	for _, blocks := range subLedgerNeedUnconfirmed {
		for _, block := range blocks {
			// remove confirmed indexes
			accountId := uint64(12)
			iDB.deleteConfirmHeight(batch, accountId, block.Height)

			// blocks move to unconfirmed pool
		}
	}
	if err := iDB.store.Write(batch); err != nil {
		return err
	}
	return nil

}

func (iDB *IndexDB) deleteSnapshotBlockHash(batch Batch, snapshotBlockHash *types.Hash) {

}

func (iDB *IndexDB) deleteSnapshotBlockHeight(batch Batch, snapshotBlockHeight uint64) {

}

func (iDB *IndexDB) deleteAccountBlockHash(batch Batch, blockHash *types.Hash) {
	key, _ := dbutils.EncodeKey(AccountBlockHashKeyPrefix, blockHash.Bytes())

	batch.Delete(key)
}

func (iDB *IndexDB) deleteAccountBlockHeight(batch Batch, accountId uint64, height uint64) {

}

func (iDB *IndexDB) deleteReceiveHeight(batch Batch, sendAccountId, sendHeight uint64) {

}

func (iDB *IndexDB) deleteOnRoad(batch Batch, toAccountId, sendAccountId, sendHeight uint64) {

}

func (iDB *IndexDB) deleteVmLogList(batch Batch, logHash *types.Hash) {
}

func (iDB *IndexDB) deleteConfirmHeight(batch Batch, accountId uint64, height uint64) {
}
