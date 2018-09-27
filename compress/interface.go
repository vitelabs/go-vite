package compress

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetConfirmSubLedger(fromHeight uint64, toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
}
