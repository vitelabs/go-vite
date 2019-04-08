package statistics

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

func (sDB *StatisticsDB) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	for _, v := range blocks {
		if v.AccountBlock.IsSendBlock() {
			sDB.newBlockSignal(v.AccountBlock)
		}
	}
	return nil
}

func (sDB *StatisticsDB) newBlockSignal(block *ledger.AccountBlock) {
	//todo
}

func (sDB *StatisticsDB) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}

func (sDB *StatisticsDB) newSnapshotBlocks(chunks []*ledger.SnapshotChunk) {

}

//--- methods below aren't used

func (sDB *StatisticsDB) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	return nil
}

func (sDB *StatisticsDB) PrepareInsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	return nil
}
func (sDB *StatisticsDB) InsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	return nil
}

func (sDB *StatisticsDB) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}
func (sDB *StatisticsDB) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	return nil
}

func (sDB *StatisticsDB) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	return nil
}
