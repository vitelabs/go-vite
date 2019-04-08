package util

import "github.com/vitelabs/go-vite/ledger"

type GlobalStatus interface {
	Seed() (uint64, error)
	SnapshotBlock() *ledger.SnapshotBlock
}
