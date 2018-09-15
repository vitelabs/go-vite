package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type SyncState int

const (
	SyncNotStart SyncState = iota
	Syncing
	Syncdone
	Syncerr
)

var syncStatus = [...]string{
	SyncNotStart: "Sync Not Start",
	Syncing:      "Synchronising",
	Syncdone:     "Sync done",
	Syncerr:      "Sync error",
}

func (this SyncState) String() string {
	return syncStatus[this]
}

type BlockChain interface {
	GetAccountBlockMap(AccountSegment) (map[string][]*ledger.AccountBlock, error)
	GetSnapshotBlocks(*Segment) ([]*ledger.SnapshotBlock, error)
	GetLatestAccountBlock(addr string) (*ledger.AccountBlock, error)
	GetLatestSnapshotBlock() (*ledger.SnapshotBlock, error)
	GetGenesesBlock() (*ledger.SnapshotBlock, error)
	GetSubLedger(startHeight *big.Int, endHeight *big.Int) ([]*ledger.SnapshotBlock, []*ledger.AccountBlock, error)

	GetAbHashList(segment *Segment) ([]*types.Hash, error)
	GetSbHashList(segment *Segment) ([]*types.Hash, error)
	GetSnapshotContent(snapshotBlockHash *types.Hash)
	//GetSbAndSc(originBlockHash *types.Hash, count uint64, forward bool)([]*ledger.SnapshotBlock, []map, error)
}
