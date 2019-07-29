package pool

import (
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"time"
)

func (pl *pool) PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	// ignore
	return nil
}

func (pl *pool) InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error {
	// ignore
	return nil
}

func (pl *pool) PrepareInsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}

func (pl *pool) InsertSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	for _, v := range chunks {
		block := v.SnapshotBlock
		if block == nil {
			continue
		}
		tps := 0
		if v.AccountBlocks != nil {
			tps = len(v.AccountBlocks)
		}
		fmt.Printf("[Insert] Height:%d, Hash:%s, Timestamp:%s, Producer:%s, Time:%s, Cnt:%d\n", block.Height, block.Hash, block.Timestamp, block.Producer(), time.Now(), tps)
	}
	return nil
}

func (pl *pool) PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	// ignore
	return nil
}

func (pl *pool) DeleteAccountBlocks(blocks []*ledger.AccountBlock) error {
	// ignore
	return nil
}

func (pl *pool) PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}

func (pl *pool) DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error {
	// ignore
	return nil
}
