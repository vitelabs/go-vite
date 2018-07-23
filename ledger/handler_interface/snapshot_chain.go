package handler_interface

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
)

type SyncInfo struct {
	BeginHeight   *big.Int
	TargetHeight  *big.Int
	CurrentHeight *big.Int
}

type SnapshotChain interface {
	HandleGetBlocks(msg *protoTypes.GetSnapshotBlocksMsg, peer *protoTypes.Peer) error
	HandleSendBlocks(msg *protoTypes.SnapshotBlocksMsg, peer *protoTypes.Peer) error
	SyncPeer(peer *protoTypes.Peer)
	WriteMiningBlock(block *ledger.SnapshotBlock) error
	GetNeedSnapshot() ([]*ledger.AccountBlock, error)
	GetLatestBlock() (*ledger.SnapshotBlock, error)
	GetBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetBlockByHeight(height *big.Int) (*ledger.SnapshotBlock, error)
	GetFirstSyncInfo() (*SyncInfo)
}
