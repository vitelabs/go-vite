package vm_db

import "github.com/vitelabs/go-vite/ledger"

func (db *vmDB) GetGenesisSnapshotHeader() *ledger.SnapshotBlock {
	return db.chain.GetGenesisSnapshotHeader()
}
