package vm_context

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
)

type Chain interface {
	GetAccount(address *types.Address) (*ledger.Account, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)

	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetStateTrie(hash *types.Hash) *trie.Trie

	NewStateTrie() *trie.Trie
	GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error)
	GetContractGid(addr *types.Address) (*types.Gid, error)
}
