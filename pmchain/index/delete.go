package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (iDB *IndexDB) DeleteSnapshotBlocks(deletedSnapshotSegments []*chain_block.SnapshotSegment, unconfirmedBlocks []*ledger.AccountBlock) error {
	// snapshotBlocks []*ledger.SnapshotBlock,
	// subLedgerNeedDeleted map[types.Address][]*ledger.AccountBlock,
	// subLedgerNeedUnconfirmed map[types.Address][]*ledger.AccountBlock

	batch := iDB.store.NewBatch()

	for _, seg := range deletedSnapshotSegments {
		snapshotBlock := seg.SnapshotBlock
		iDB.deleteSnapshotBlockHash(batch, &snapshotBlock.Hash)
		iDB.deleteSnapshotBlockHeight(batch, snapshotBlock.Height)

		for _, block := range seg.AccountBlocks {
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

	for _, block := range unconfirmedBlocks {
		// remove confirmed indexes
		accountId := uint64(12)
		iDB.deleteConfirmHeight(batch, accountId, block.Height)

		// blocks move to unconfirmed pool
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
	key := make([]byte, 0, 1+types.HashSize)
	key = append(append(key, AccountBlockHashKeyPrefix), blockHash.Bytes()...)

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
