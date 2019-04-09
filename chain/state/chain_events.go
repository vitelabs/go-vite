package chain_state

import (
	"github.com/vitelabs/go-vite/ledger"
)

func (sDB *StateDB) InsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	sDB.storageRedo.SetSnapshot(sDB.chain.GetLatestSnapshotBlock().Height+1, nil, true)
	return nil
}

func (sDB *StateDB) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	currentHeight := sDB.chain.GetLatestSnapshotBlock().Height + 1
	logMap, hasRedo, err := sDB.storageRedo.QueryLog(currentHeight)
	if err != nil {
		panic(err)
	}

	sDB.storageRedo.SetSnapshot(currentHeight, logMap, hasRedo)
	return nil
}
