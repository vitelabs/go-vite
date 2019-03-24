package vm_db

import "github.com/vitelabs/go-vite/ledger"

func (db *vmDb) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return db.chain.GetGenesisSnapshotBlock()
}
