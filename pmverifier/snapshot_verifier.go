package pmverifier

import (
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/pmchain"
)

type SnapshotVerifier struct {
	reader pmchain.Chain
	cs     consensus.Verifier
}
