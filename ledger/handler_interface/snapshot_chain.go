package handler_interface

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
)

type SyncInfo struct {
	BeginHeight     *big.Int
	TargetHeight    *big.Int
	CurrentHeight   *big.Int
	IsFirstSyncDone bool
}

type SnapshotChain interface {
	HandleGetBlocks(msg *protoTypes.GetSnapshotBlocksMsg, peer *protoTypes.Peer) error
	HandleSendBlocks(msg *protoTypes.SnapshotBlocksMsg, peer *protoTypes.Peer) error
	SyncPeer(peer *protoTypes.Peer)
	WriteMiningBlock(block *ledger.SnapshotBlock) error
	GetLatestBlock() (*ledger.SnapshotBlock, error)
	GetBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	GetBlockByHeight(height *big.Int) (*ledger.SnapshotBlock, error)
	GetFirstSyncInfo() *SyncInfo
}
