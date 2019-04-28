package vm_db

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (db *vmDb) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return db.chain.GetGenesisSnapshotBlock()
}

func (db *vmDb) GetConfirmSnapshotHeader(blockHash types.Hash) (*ledger.SnapshotBlock, error) {
	return db.chain.GetConfirmSnapshotHeaderByAbHash(blockHash)
}
