package chain_cache

import (
	"github.com/vitelabs/go-vite/chain_db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetUnConfirmedSubLedger() (map[types.Address][]*ledger.AccountBlock, error)
	GetUnConfirmedPartSubLedger(addrList []types.Address) (map[types.Address][]*ledger.AccountBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetConfirmSubLedgerBySnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) (map[types.Address][]*ledger.AccountBlock, error)
	GetSnapshotBlocksByHeight(height uint64, count uint64, forward, containSnapshotContent bool) ([]*ledger.SnapshotBlock, error)

	ChainDb() *chain_db.ChainDb
}
