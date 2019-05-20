package consensus

import (
	"encoding/json"
	"fmt"

	"github.com/vitelabs/go-vite/vm_db"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
)

// (start, end]
func (cs *consensus) OnChainGC(start *ledger.SnapshotBlock, end *ledger.SnapshotBlock) error {
	if start == nil || end == nil {
		panic(fmt.Sprintf("start[%t] or end[%t] is nil.", start == nil, end == nil))
	}

	stime := start.Timestamp
	etime := end.Timestamp

	sIndex := cs.rw.dayPoints.Time2Index(*stime)
	eIndex := cs.rw.dayPoints.Time2Index(*etime)

	for i := sIndex; i <= eIndex; i++ {
		day, err := cs.rw.dayPoints.GetByIndex(i)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("load day index[%d] fail.", i))
		}
		byt, _ := json.Marshal(day)
		cs.mLog.Info("reload day[%d] stats for chain gc. %s", i, string(byt))
	}
	return nil
}

func (cs *consensus) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	// ignore
	return nil
}

func (cs *consensus) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	// ignore
	return nil
}

func (cs *consensus) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}

func (cs *consensus) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	for _, v := range chunks {
		block := v.SnapshotBlock
		if block == nil {
			continue
		}
		bTime := block.Timestamp

		index := cs.snapshot.Time2Index(*bTime)

		stime, _ := cs.snapshot.Index2Time(index)
		if stime.Equal(*bTime) {
			snapshotBlock, err := cs.rw.rw.GetSnapshotBlockByHeight(block.Height - 1)
			if err != nil {
				cs.mLog.Error("can't get block", "height", block.Height-1)
				return nil
			}
			cs.snapshot.triggerLoad(snapshotBlock)
		}
	}
	return nil
}

func (cs *consensus) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	// ignore
	return nil
}

func (cs *consensus) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	// ignore
	return nil
}

func (cs *consensus) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}

func (cs *consensus) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}
