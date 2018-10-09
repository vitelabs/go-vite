package sender

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetLatestBlockEventId() uint64
	GetEvent(eventId uint64) (byte, []types.Hash, error)
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
}
