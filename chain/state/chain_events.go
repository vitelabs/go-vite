package chain_state

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (sDB *StateDB) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil

}
func (sDB *StateDB) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}

func (sDB *StateDB) PrepareInsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	return nil
}
func (sDB *StateDB) InsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	sDB.storageRedo.SetSnapshot(sDB.chain.GetLatestSnapshotBlock().Height+1, nil, true)
	return nil
}

func (sDB *StateDB) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}
func (sDB *StateDB) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}

func (sDB *StateDB) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
func (sDB *StateDB) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	currentHeight := sDB.chain.GetLatestSnapshotBlock().Height + 1
	logMap, hasRedo, err := sDB.storageRedo.QueryLog(currentHeight)
	if err != nil {
		panic(err)
	}

	sDB.storageRedo.SetSnapshot(sDB.chain.GetLatestSnapshotBlock().Height, logMap, hasRedo)
	return nil
}
