package util

import "github.com/vitelabs/go-vite/ledger"

type GlobalStatus struct {
	Seed          interface{}
	SnapshotBlock *ledger.SnapshotBlock
}
