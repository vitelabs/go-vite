package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
)

type snapshotBlockReader interface {
	GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
	GetSnapshotBlockByHash(hash types.Hash) (*ledger.SnapshotBlock, error)
	GetSnapshotBlocks(blockHash types.Hash, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)
	GetSnapshotBlocksByHeight(height uint64, higher bool, count uint64) ([]*ledger.SnapshotBlock, error)
}

type accountBockReader interface {
	GetAccountBlockByHeight(addr types.Address, height uint64) (*ledger.AccountBlock, error)
	GetAccountBlockByHash(blockHash types.Hash) (*ledger.AccountBlock, error)
	GetAccountBlocks(blockHash types.Hash, count uint64) ([]*ledger.AccountBlock, error)
	GetAccountBlocksByHeight(addr types.Address, height uint64, count uint64) ([]*ledger.AccountBlock, error)
}

type ledgerReader interface {
	GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error)
}

type chainReader interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetGenesisSnapshotBlock() *ledger.SnapshotBlock
}

type syncCacher interface {
	GetSyncCache() interfaces.SyncCache
}

type Chain interface {
	snapshotBlockReader
	accountBockReader
	chainReader
	ledgerReader
	syncCacher
}

type Producer interface {
	// IsProducer can check an address is whether a producer
	IsProducer(address types.Address) bool
}

type Verifier interface {
	VerifyNetSb(block *ledger.SnapshotBlock) error
	VerifyNetAb(block *ledger.AccountBlock) error
}

// SnapshotBlockCallback will be invoked when receive a block,
// source mark where the block come from: broadcast, sync or fetch
type SnapshotBlockCallback = func(block *ledger.SnapshotBlock, source types.BlockSource)

// AccountBlockCallback will be invoked when receive a block,
// source mark where the block come from: broadcast, sync or fetch
type AccountBlockCallback = func(addr types.Address, block *ledger.AccountBlock, source types.BlockSource)

// SyncStateCallback will be invoked when sync state change, the param is state after change
type SyncStateCallback = func(SyncState)

// A BlockSubscriber implementation can be subscribed and Unsubscribed, when got chain block, should notify subscribers
type BlockSubscriber interface {
	// SubscribeAccountBlock return the subId, always larger than 0, use to unsubscribe
	SubscribeAccountBlock(fn AccountBlockCallback) (subId int)
	// UnsubscribeAccountBlock if subId is 0, then ignore
	UnsubscribeAccountBlock(subId int)

	// SubscribeSnapshotBlock return the subId, always larger than 0, use to unsubscribe
	SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int)
	// UnsubscribeSnapshotBlock if subId is 0, then ignore
	UnsubscribeSnapshotBlock(subId int)
}

type SyncStateSubscriber interface {
	// SubscribeSyncStatus return the subId, always larger than 0, use to unsubscribe
	SubscribeSyncStatus(fn SyncStateCallback) (subId int)
	// UnsubscribeSyncStatus if subId is 0, then ignore
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

	// FetchAccountBlocks address is optional
	FetchAccountBlocks(start types.Hash, count uint64, address *types.Address)

	// FetchAccountBlocksWithHeight add snapshot height
	FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64)
}

// A Syncer implementation can synchronise blocks to peers
type Syncer interface {
	SyncStateSubscriber
	Status() SyncStatus
	Detail() SyncDetail
}

type Net interface {
	p2p.Protocol
	Syncer
	Fetcher
	Broadcaster
	BlockSubscriber
	Start(svr p2p.P2P) error
	Stop() error
	Info() NodeInfo
}
