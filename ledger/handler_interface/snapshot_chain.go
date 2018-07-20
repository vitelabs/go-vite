package handler_interface

import (
	"github.com/vitelabs/go-vite/protocols"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type SyncInfo struct {
	BeginHeight *big.Int
	TargetHeight *big.Int
	CurrentHeight *big.Int
}

type Snapshot_chain interface {
	HandleGetBlocks(msg *protocols.GetSnapshotBlocksMsg, peer *protocols.Peer) error
	HandleSendBlocks (msg protocols.SnapshotBlocksMsg, peer *protocols.Peer) error
	SyncPeer (peer *protocols.Peer)
	WriteMiningBlock (block *ledger.SnapshotBlock) error
	GetNeedSnapshot () ([]*ledger.AccountBlock, error)
	StopAllWrite ()
	StartAllWrite ()
	GetLatestBlock () (*ledger.SnapshotBlock, error)
	GetBlockByHash (hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetBlockByHeight (height *big.Int) (*ledger.SnapshotBlock, error)
	GetFirstSyncInfo () (*SyncInfo)
}

