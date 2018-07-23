package consensus

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type SnapshotReader interface {
}

type SnapshotHeader struct {
	Timestamp uint64
	Producer  *types.Address
}

type Verifier interface {
	Verify(reader SnapshotReader, block *ledger.SnapshotBlock) (bool, error)
}

type Seal interface {
	Seal() error
}
