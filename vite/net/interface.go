package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
)

// all query include from block
type Chain interface {
	// the second return value mean chunk befor/after file
	GetSubLedgerByHeight(start, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64)
	GetSubLedgerByHash(origin *types.Hash, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64, error)

	// query chunk
	GetConfirmSubLedger(start, end uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)

	// single
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)
	// batcher
	GetSnapshotBlocksByHash(origin *types.Hash, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)

	// single
	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)

	// batcher
	GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, start, count uint64, forward bool) ([]*ledger.AccountBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	Compressor() *compress.Compressor
}

type Verifier interface {
	//VerifyforP2P(block *ledger.AccountBlock) bool
	VerifyNetSb(block *ledger.SnapshotBlock) error
	VerifyNetAb(block *ledger.AccountBlock) error
}

// @section Subscriber
type SnapshotBlockCallback = func(block *ledger.SnapshotBlock, source types.BlockSource)
type AccountblockCallback = func(addr types.Address, block *ledger.AccountBlock, source types.BlockSource)
type SyncStateCallback = func(SyncState)

type Subscriber interface {
	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeAccountBlock(fn AccountblockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeAccountBlock(subId int)

	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeSnapshotBlock(subId int)

	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeSyncStatus(fn SyncStateCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeSyncStatus(subId int)

	SyncState() SyncState
}

// @section Broadcaster
type Broadcaster interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
	BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock)
	BroadcastAccountBlock(block *ledger.AccountBlock)
	BroadcastAccountBlocks(blocks []*ledger.AccountBlock)
}

// @section Broadcaster
type Fetcher interface {
	FetchSnapshotBlocks(start types.Hash, count uint64)

	// address is optional
	FetchAccountBlocks(start types.Hash, count uint64, address *types.Address)

	// add snapshot height
	FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64)
}

// @section Receiver
type Receiver interface {
	ReceiveSnapshotBlock(block *ledger.SnapshotBlock)
	ReceiveAccountBlock(block *ledger.AccountBlock)

	ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock)
	ReceiveAccountBlocks(blocks []*ledger.AccountBlock)

	ReceiveNewSnapshotBlock(block *ledger.SnapshotBlock)
	ReceiveNewAccountBlock(block *ledger.AccountBlock)

	SubscribeAccountBlock(fn AccountblockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeAccountBlock(subId int)

	// return the subId, use to unsubscibe
	// subId is always larger than 0
	SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int)
	// if subId is 0, then ignore
	UnsubscribeSnapshotBlock(subId int)
}

// @section Syncer
type Syncer interface {
	Status() *SyncStatus
	SubscribeSyncStatus(fn SyncStateCallback) (subId int)
	UnsubscribeSyncStatus(subId int)
	SyncState() SyncState
}

// @section Net
type Net interface {
	Syncer
	Fetcher
	Broadcaster
	Receiver
	Protocols() []*p2p.Protocol
	Start(svr p2p.Server) error
	Stop()
	Info() *NodeInfo
	Tasks() []*Task
	AddPlugin(plugin p2p.Plugin)
}
