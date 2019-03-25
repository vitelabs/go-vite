package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/compress"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
)

type Chain interface {
	GetSubLedgerByHeight(start, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64)
	GetSubLedgerByHash(origin *types.Hash, count uint64, forward bool) ([]*ledger.CompressedFileMeta, [][2]uint64, error)

	GetConfirmSubLedger(start, end uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error)

	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error)

	GetSnapshotBlocksByHash(origin *types.Hash, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height, count uint64, forward, content bool) ([]*ledger.SnapshotBlock, error)

	GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error)
	GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error)

	GetAccountBlocksByHash(addr types.Address, origin *types.Hash, count uint64, forward bool) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, start, count uint64, forward bool) ([]*ledger.AccountBlock, error)

	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock

	// Compressor could compress/uncompress chain file
	Compressor() *compress.Compressor
}

type Verifier interface {
	VerifyNetSb(block *ledger.SnapshotBlock) error
	VerifyNetAb(block *ledger.AccountBlock) error
}

// source is use to statistic where the block come from, broadcast, sync or fetch
type SnapshotBlockCallback = func(block *ledger.SnapshotBlock, source types.BlockSource)
type AccountblockCallback = func(addr types.Address, block *ledger.AccountBlock, source types.BlockSource)
type SyncStateCallback = func(SyncState)

// A BlockSubscriber implementation can be subscribed and Unsubscribed, when got chain block, should notify subscribers
type BlockSubscriber interface {
	// SubscribeAccountBlock return the subId, always larger than 0, use to unsubscribe
	SubscribeAccountBlock(fn AccountblockCallback) (subId int)
	// UnsubscribeAccountBlock, if subId is 0, then ignore
	UnsubscribeAccountBlock(subId int)

	// SubscribeSnapshotBlock return the subId, always larger than 0, use to unsubscribe
	SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int)
	// UnsubscribeSnapshotBlock, if subId is 0, then ignore
	UnsubscribeSnapshotBlock(subId int)
}

type SyncStateSubscriber interface {
	// SubscribeSyncStatus return the subId, always larger than 0, use to unsubscribe
	SubscribeSyncStatus(fn SyncStateCallback) (subId int)
	// UnsubscribeSyncStatus, if subId is 0, then ignore
	UnsubscribeSyncStatus(subId int)

	// SyncState return the latest sync state_bak
	SyncState() SyncState
}

type Subscriber interface {
	BlockSubscriber
	SyncStateSubscriber
}

// A Broadcaster implementation can send blocks to the peers connected
type Broadcaster interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
	BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock)
	BroadcastAccountBlock(block *ledger.AccountBlock)
	BroadcastAccountBlocks(blocks []*ledger.AccountBlock)
}

// A Fetcher implementation can request the wanted blocks to peers
type Fetcher interface {
	FetchSnapshotBlocks(start types.Hash, count uint64)

	// FetchAccountBlocks, address is optional
	FetchAccountBlocks(start types.Hash, count uint64, address *types.Address)

	// FetchAccountBlocksWithHeight, add snapshot height
	FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64)
}

// A Syncer implementation can synchronise blocks to peers
type Syncer interface {
	SyncStateSubscriber
	Status() SyncStatus
	Detail() SyncDetail
}

type Net interface {
	Syncer
	Fetcher
	Broadcaster
	BlockSubscriber
	Protocols() []*p2p.Protocol
	Start(svr p2p.Server) error
	Stop()
	Info() NodeInfo
	AddPlugin(plugin p2p.Plugin)
}
