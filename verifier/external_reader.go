package verifier

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
)

type Chain interface {
	AccountReader
	SnapshotReader
}

type Consensus interface {
	VerifyAccountProducer(block *ledger.AccountBlock) error
}

type SnapshotReader interface {
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetConfirmBlock(accountBlock *ledger.AccountBlock) *ledger.SnapshotBlock
	GetConfirmTimes(accountBlock *ledger.AccountBlock) uint64
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)
}

type AccountReader interface {
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetContractGid(addr *types.Address) (*types.Gid, error)
	AccountType(address *types.Address) (uint64, error)
	GetAccount(address *types.Address) (*ledger.Account, error)
	GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error)
	GetStateTrie(hash *types.Hash) *trie.Trie
	NewStateTrie() *trie.Trie
}
