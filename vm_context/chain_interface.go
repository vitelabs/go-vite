package vm_context

import (
	"github.com/vitelabs/go-vite/chain/cache"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"time"
)

type Chain interface {
	GetAccount(address *types.Address) (*ledger.Account, error)
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)
	GetConfirmBlock(accountBlockHash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockHeadByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockBeforeTime(blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetStateTrie(hash *types.Hash) *trie.Trie

	NewStateTrie() *trie.Trie
	GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error)
	GetContractGid(addr *types.Address) (*types.Gid, error)

	SaList() *chain_cache.AdditionList

	GetReceiveBlockHeights(hash *types.Hash) ([]uint64, error)
}
