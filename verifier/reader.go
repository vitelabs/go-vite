package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
)

type consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	generator.Consensus
}

type onRoadPool interface {
	IsFrontOnRoadOfCaller(gid types.Gid, orAddr, caller types.Address, hash types.Hash) (bool, error)
}
