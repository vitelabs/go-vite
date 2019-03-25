package util

import "github.com/vitelabs/go-vite/ledger"

type GlobalStatus struct {
	Seed          uint64
	SnapshotBlock *ledger.SnapshotBlock
}
