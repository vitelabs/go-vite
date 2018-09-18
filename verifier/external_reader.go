package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type ChainReader interface {
	AccountReader
	SnapshotReader
}

type SnapshotReader interface {
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetConfirmBlock(accountBlock *ledger.AccountBlock) *ledger.SnapshotBlock
	GetConfirmTimes(accountBlock *ledger.AccountBlock) uint64
}

type AccountReader interface {
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetContractGid(addr *types.Address) (*types.Gid, error)
	//GetReceiveTimes(addr *types.Address, sendBlock *types.Hash) uint8
	//GetGenesisBlockFirst() *ledger.AccountBlock
}

type ConsensusReader interface {
	VerifyAccountProducer(block *ledger.AccountBlock) error
}
