package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
	"time"
)

type EventListener interface {
	PrepareInsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error
	InsertAccountBlocks(blocks []*vm_db.VmAccountBlock) error

	PrepareInsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error
	InsertSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error

	PrepareDeleteAccountBlocks(blocks []*ledger.AccountBlock) error
	DeleteAccountBlocks(blocks []*ledger.AccountBlock) error

	PrepareDeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
	DeleteSnapshotBlocks(chunks []*ledger.SnapshotChunk) error
}

type Chain interface {
	IterateAccounts(iterateFunc func(addr types.Address, accountId uint64, err error) bool)

	QueryLatestSnapshotBlock() (*ledger.SnapshotBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock

	GetSnapshotHeightByHash(hash types.Hash) (uint64, error)

	GetUnconfirmedBlocks(addr types.Address) []*ledger.AccountBlock

	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)

	GetSnapshotHeaderBeforeTime(timestamp *time.Time) (*ledger.SnapshotBlock, error)

	GetSnapshotHeadersAfterOrEqualTime(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error)
}
