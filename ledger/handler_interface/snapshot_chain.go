package handler_interface

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	protoTypes "github.com/vitelabs/go-vite/protocols/types"
	"math/big"
)

type SyncInfo struct {
	BeginHeight      *big.Int
	TargetHeight     *big.Int
	CurrentHeight    *big.Int
	IsFirstSyncDone  bool
	IsFirstSyncStart bool
}

type SnapshotChain interface {
	HandleGetBlocks(*protoTypes.GetSnapshotBlocksMsg, *protoTypes.Peer, uint64) error
	HandleSendBlocks(*protoTypes.SnapshotBlocksMsg, *protoTypes.Peer, uint64) error
	SyncPeer(*protoTypes.Peer)
	WriteMiningBlock(*ledger.SnapshotBlock) error
	GetLatestBlock() (*ledger.SnapshotBlock, error)
	GetBlockByHash(*types.Hash) (*ledger.SnapshotBlock, error)
	GetBlockByHeight(*big.Int) (*ledger.SnapshotBlock, error)
	GetFirstSyncInfo() *SyncInfo

	GetConfirmBlock(*ledger.AccountBlock) (*ledger.SnapshotBlock, error)
	GetConfirmTimes(*ledger.SnapshotBlock) (*big.Int, error)
}
