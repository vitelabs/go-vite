package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (sDB *StateDB) DeleteInvalidAccountBlocks(invalidSubLedger map[types.Address][]*ledger.AccountBlock) {
	for _, blocks := range invalidSubLedger {
		sDB.mvDB.DeletePendingBlocks(blocks)
	}
}

func (sDB *StateDB) DeleteSubLedger(deletedSnapshotSegments []*chain_block.SnapshotSegment) error {
	size := 0
	for _, seg := range deletedSnapshotSegments {
		size += len(seg.AccountBlocks)
	}

	blockHashList := make([]*types.Hash, 0, size)
	for _, seg := range deletedSnapshotSegments {
		for _, accountBlock := range seg.AccountBlocks {
			blockHashList = append(blockHashList, &accountBlock.Hash)
		}
	}
	return sDB.mvDB.Undo(blockHashList)
}
