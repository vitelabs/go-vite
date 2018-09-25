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

type ConsensusVerifier interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error)
}

type HashHeight struct {
	Hash   types.Hash
	Height uint64
}
