package onroad

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (manager *Manager) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	for _, v := range blocks {
		if v.AccountBlock.IsSendBlock() {
			manager.newSignalToWorker(v.AccountBlock)
		}
	}
	return nil
}

func (manager *Manager) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}

func (manager *Manager) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
func (manager *Manager) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}

func (manager *Manager) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}
func (manager *Manager) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}

func (manager *Manager) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
func (manager *Manager) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
