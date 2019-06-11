package util

import (
	"github.com/vitelabs/go-vite/ledger"
)

// Only exitst in contract receive block
type GlobalStatus interface {
	Seed(snapshotHeight uint64) (uint64, error) // Random number
	SnapshotBlock() *ledger.SnapshotBlock       // Confirm snapshot block of send block
}
