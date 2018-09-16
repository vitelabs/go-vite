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
	GenesisSnapshot() *ledger.SnapshotBlock
	HeadSnapshot() (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (block *ledger.SnapshotBlock, returnErr error)
	GetConfirmBlock(block *ledger.AccountBlock) (*ledger.SnapshotBlock, error)
	GetConfirmTimes(snapshotBlock *ledger.SnapshotBlock) (uint64, error)
}

type AccountReader interface {
	HeadAccount(address *types.Address) (*ledger.AccountBlock, error)
	GetLatestAccountBlock(addr *types.Address) (block *ledger.AccountBlock, err error)
	GetAccountBlockByHash(blockHash *types.Hash) (block *ledger.AccountBlock, err error)
	GetContractGid(addr *types.Address) (*types.Gid, error)
	GetGenesisBlockFirst() *ledger.AccountBlock
	GetReceiveTimes(addr *types.Address, fromBlockHash *types.Hash) uint8
}

type ProducerReader interface {
	VerifyAccountProducer(block *ledger.AccountBlock) error
}
