package vm_db

import "github.com/vitelabs/go-vite/ledger"

func (db *vmDb) GetGenesisSnapshotHeader() *ledger.SnapshotBlock {
	return db.chain.GetGenesisSnapshotHeader()
}
