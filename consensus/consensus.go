package consensus

import (
	"github.com/vitelabs/go-vite/ledger"
)

type ConsensusVerifier interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error)
}
