package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
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

	//sbHashList := make([]*types.Hash, 0, len(deletedSnapshotSegments))
	undo := make(map[uint64]uint64)
	for _, seg := range deletedSnapshotSegments {
		for _, block := range seg.AccountBlocks {
			accountId := uint64(10)
			undoLog := sDB.mvDB.GetUndoLog(accountId, block.Height)

			currentPointer := 0
			undoLogLen := len(undoLog)
			for currentPointer < undoLogLen {
				undoKeyId := chain_dbutils.FixedBytesToUint64(undoLog[currentPointer : currentPointer+4])
				if _, ok := undo[undoKeyId]; !ok {
					undo[undoKeyId] = chain_dbutils.FixedBytesToUint64(undoLog[currentPointer+4 : currentPointer+8])
				}

				currentPointer += 8
			}
		}
		//sbHashList = append(sbHashList, &seg.SnapshotBlock.Hash)
	}

	return nil
}
