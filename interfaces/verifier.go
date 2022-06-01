package interfaces

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces/core"
)

// ConsensusVerifier is the interface that can verify block consensus.
type ConsensusVerifier interface {
	VerifyAccountProducer(block *core.AccountBlock) (bool, error)
	VerifyABsProducer(abs map[types.Gid][]*core.AccountBlock) ([]*core.AccountBlock, error)
	VerifySnapshotProducer(block *core.SnapshotBlock) (bool, error)
}
