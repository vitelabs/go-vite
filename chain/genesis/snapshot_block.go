package chain_genesis

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"time"
)

func newGenesisSnapshotContent(accountBlocks []*vm_db.VmAccountBlock) ledger.SnapshotContent {
	sc := make(ledger.SnapshotContent, len(accountBlocks))
	for _, vmBlock := range accountBlocks {
		accountBlock := vmBlock.AccountBlock
		if hashHeight, ok := sc[accountBlock.AccountAddress]; !ok || hashHeight.Height < accountBlock.Height {
			sc[accountBlock.AccountAddress] = &ledger.HashHeight{
				Height: accountBlock.Height,
				Hash:   accountBlock.Hash,
			}
		}
	}

	return sc
}

func NewGenesisSnapshotBlock(accountBlocks []*vm_db.VmAccountBlock) *ledger.SnapshotBlock {
	// 2019-03-16 12:00:00 UTC/GMT +8
	genesisTimestamp := time.Unix(1552708800, 0)

	genesisSnapshotBlock := &ledger.SnapshotBlock{
		Height:          1,                 // height
		Timestamp:       &genesisTimestamp, // timestamp
		SnapshotContent: newGenesisSnapshotContent(accountBlocks),
	}

	genesisSnapshotBlock.Hash = genesisSnapshotBlock.ComputeHash()

	return genesisSnapshotBlock
}
