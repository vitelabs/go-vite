package rocket

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/trie"
	"github.com/vitelabs/go-vite/vm_context"
	"time"
)

type Chain interface {
	InsertAccountBlocks(vmAccountBlocks []*vm_context.VmAccountBlock) error
	GetAccount(address *types.Address) (*ledger.Account, error)
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error)
	GetSnapshotBlockBeforeTime(blockCreatedTime *time.Time) (*ledger.SnapshotBlock, error)

	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetStateTrie(hash *types.Hash) *trie.Trie

	NewStateTrie() *trie.Trie

	GetConfirmAccountBlock(snapshotHeight uint64, address *types.Address) (*ledger.AccountBlock, error)
	GetContractGid(addr *types.Address) (*types.Gid, error)

	InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error

	GetNeedSnapshotContent() ledger.SnapshotContent

	GenStateTrie(prevStateHash types.Hash, snapshotContent ledger.SnapshotContent) (*trie.Trie, error)
}
