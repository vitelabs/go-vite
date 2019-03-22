package chain_genesis

import (
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

func newGenesisSnapshotContent() ledger.SnapshotContent {
	return nil
}

func NewGenesisSnapshotBlock() *ledger.SnapshotBlock {
	// 2019-03-16 12:00:00 UTC/GMT +8
	genesisTimestamp := time.Unix(1552708800, 0)

	genesisSnapshotBlock := &ledger.SnapshotBlock{
		Height:          1,                 // height
		Timestamp:       &genesisTimestamp, // timestamp
		SnapshotContent: newGenesisSnapshotContent(),
	}

	genesisSnapshotBlock.Hash = genesisSnapshotBlock.ComputeHash()

	return genesisSnapshotBlock
}
