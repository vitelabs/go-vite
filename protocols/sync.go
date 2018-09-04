package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type syncState int

const (
	syncNotStart syncState = iota
	syncing
	syncdone
	syncerr
)

var syncStatus = [...]string{
	syncNotStart: "Sync Not Start",
	syncing:      "Synchronising",
	syncdone:     "Sync done",
	syncerr:      "Sync error",
}

func (this syncState) String() string {
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
