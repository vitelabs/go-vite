package chain_genesis

import (
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

func createGenesisSnapshotBlock() *ledger.SnapshotBlock {
	// 2019-03-16 12:00:00 UTC/GMT +8
	genesisTimestamp := time.Unix(1552708800, 0)

	genesisSnapshotBlock := &ledger.SnapshotBlock{
		Height:    1,                 // height
		Timestamp: &genesisTimestamp, // timestamp
	}

	// compute hash
	genesisSnapshotBlock.Hash = genesisSnapshotBlock.ComputeHash()

}
