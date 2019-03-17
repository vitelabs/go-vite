package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (sDB *StateDB) DeleteInvalidAccountBlocks(invalidSubLedger map[types.Address][]*ledger.AccountBlock) {
	for _, blocks := range invalidSubLedger {
		sDB.mvDB.DeletePendingByBlockHash(blocks)
	}
}

// TODO
func (sDB *StateDB) DeleteSubLedger(deletedSnapshotSegments []*chain_block.SnapshotSegment) error {
	// clean pending
	sDB.mvDB.pending.Clean()

	sbHashList := make([]*types.Hash, 0, len(deletedSnapshotSegments))
	for _, seg := range deletedSnapshotSegments {
		sbHashList = append(sbHashList, &seg.SnapshotBlock.Hash)
	}
	_, err := sDB.getKeyListBySbHashList(sbHashList)
	if err != nil {
		return err
	}

	return nil
}
