package verifier

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/vm_db"
)

type cssConsensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
}

type onRoadPool interface {
	IsFrontOnRoadOfCaller(gid types.Gid, orAddr, caller types.Address, hash types.Hash) (bool, error)
}

type accountChain interface {
	vm_db.Chain

	IsReceived(sendBlockHash types.Hash) (bool, error)
	GetReceiveAbBySendAb(sendBlockHash types.Hash) (*ledger.AccountBlock, error)
	IsGenesisAccountBlock(block types.Hash) bool
	IsSeedConfirmedNTimes(blockHash types.Hash, n uint64) (bool, error)
}
