package sender

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type Chain interface {
	GetLatestBlockEventId() (uint64, error)
	GetEvent(eventId uint64) (byte, []types.Hash, error)
	GetConfirmSubLedgerBySnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) (map[types.Address][]*ledger.AccountBlock, error)
	vm_context.Chain
}
