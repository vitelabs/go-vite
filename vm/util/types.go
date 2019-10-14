package util

import (
	"github.com/vitelabs/go-vite/ledger"
)

// GlobalStatus contains confirm snapshot block and seed of contract response block
type GlobalStatus interface {
	Seed() (uint64, error)                // Random number, returns same number
	Random() (uint64, error)              // Random number, returns different number every time calls
	SnapshotBlock() *ledger.SnapshotBlock // Confirm snapshot block of send block
}
