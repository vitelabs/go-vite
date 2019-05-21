package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
	SBPReader() core.SBPStatReader
}

type onRoadPool interface {
	IsFrontOnRoadOfCaller(gid types.Gid, orAddr, caller types.Address, hash types.Hash) (bool, error)
}

type accountChain interface {
	vm_db.Chain

	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)
	IsReceived(sendBlockHash types.Hash) (bool, error)
	GetReceiveAbBySendAb(sendBlockHash types.Hash) (*ledger.AccountBlock, error)
	IsGenesisAccountBlock(block types.Hash) bool
}
