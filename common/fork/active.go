package fork

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type ActiveManager struct {
	chain Chain
}

func newActiveManager(chain Chain) *ActiveManager {
	return &ActiveManager{
		chain: chain,
	}
}

// TODO
func (am *ActiveManager) IsActive(point ForkPointItem) bool {
	cooperateForkPoint, ok := forkPointMap["CooperateFork"]
	if !ok {
		panic("check cooperate fork failed. CooperateFork is not existed.")
	}

	if point.Height <= cooperateForkPoint.Height {
		// For backward compatibility, auto active old fork point
		return true
	}

	return false
}

func (am *ActiveManager) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	panic("implement me")
}

func (am *ActiveManager) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	panic("implement me")
}

func (am *ActiveManager) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	panic("implement me")
}

func (am *ActiveManager) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	panic("implement me")
}

func (am *ActiveManager) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	panic("implement me")
}

func (am *ActiveManager) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	panic("implement me")
}

func (am *ActiveManager) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	panic("implement me")
}

func (am *ActiveManager) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	panic("implement me")
}
