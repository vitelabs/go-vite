package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

/*
 *	TODO
 *	1. accountId
 *  2. location
 *	3. sendAccountId
 *	4. sendHeight
 *	5. new account
 */
func (iDB *IndexDB) InsertAccountBlock(accountBlock *ledger.AccountBlock) error {
	batch := iDB.db.NewBatch()
	accountId := uint64(1)
	if err := insertAccountBlockHash(batch, &accountBlock.Hash, accountId, accountBlock.Height); err != nil {
		return err
	}
	if err := insertAccountBlockHeight(batch, accountId, accountBlock.Height, ""); err != nil {
		return err
	}
	// close send block
	if accountBlock.IsReceiveBlock() {
		sendAccountId := uint64(2)
		sendHeight := uint64(12)
		if err := insertReceiveHeight(batch, sendAccountId, sendHeight, accountId, accountBlock.Height); err != nil {
			return err
		}
	}

	return iDB.db.Write(batch)
}

/*
 *	TODO
 *	1. location
 *	2. accountId
 */
func (iDB *IndexDB) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error {
	batch := iDB.db.NewBatch()
	if err := insertSnapshotBlockHash(batch, &snapshotBlock.Hash, snapshotBlock.Height); err != nil {
		return err
	}
	if err := insertSnapshotBlockHeight(batch, snapshotBlock.Height, ""); err != nil {
		return err
	}
	// confirm block
	for _, hashHeight := range snapshotBlock.SnapshotContent {
		accountId := uint64(1)
		if err := insertConfirmHeight(batch, accountId, hashHeight.Height, snapshotBlock.Height); err != nil {
			return err
		}
	}

	return iDB.db.Write(batch)
}

func (iDB *IndexDB) insertAccount(batch Batch, addr *types.Address, accountId uint64) error {
	return nil
}

func insertAccountBlockHash(batch Batch, blockHash *types.Hash, accountId uint64, height uint64) error {
	return nil
}

func insertAccountBlockHeight(batch Batch, accountId uint64, height uint64, location string) error {
	return nil
}

func insertReceiveHeight(batch Batch, sendAccountId, sendHeight, receiveAccountId, receiveHeight uint64) error {
	return nil
}

func insertConfirmHeight(batch Batch, accountId uint64, height uint64, snapshotHeight uint64) error {
	return nil
}

func insertSnapshotBlockHash(batch Batch, snapshotBlockHash *types.Hash, height uint64) error {
	return nil
}

func insertSnapshotBlockHeight(batch Batch, snapshotBlockHeight uint64, location string) error {
	return nil
}
