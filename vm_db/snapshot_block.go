package vm_db

import "github.com/vitelabs/go-vite/ledger"

func (vdb *vmDb) GetGenesisSnapshotBlock() *ledger.SnapshotBlock {
	return vdb.chain.GetGenesisSnapshotBlock()
}
