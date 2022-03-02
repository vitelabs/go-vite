package vm_db

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

func (vdb *vmDb) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return vdb.chain.GetGenesisSnapshotBlock()
}

func (db *vmDb) GetConfirmSnapshotHeader(blockHash types.Hash) (*ledger.SnapshotBlock, error) {
	return db.chain.GetConfirmSnapshotHeaderByAbHash(blockHash)
}

func (db *vmDb) GetConfirmedTimes(blockHash types.Hash) (uint64, error) {
	return db.chain.GetConfirmedTimes(blockHash)
}

func (db *vmDb) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	return db.chain.GetSnapshotBlockByHeight(height)
}
