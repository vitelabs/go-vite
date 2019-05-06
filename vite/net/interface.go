package net

import (
	"fmt"

	"github.com/go-errors/errors"

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

type syncChain interface {
	syncCacher
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
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

type ChunkReader interface {
	Peek() *Chunk
	Pop(endHash types.Hash)
}

// Chunk means a chain chunk, contains snapshot blocks and dependent account blocks,
// SnapshotRange means the chunk range, the first HashHeight is the prevHash and prevHeight.
// eg. Chunk is from 101 to 200, SnapshotChunks is [101 ... 200], but SnapshotRange is [100, 200].
// AccountRange like SnapshotRange but describe every account chain.
// HashMap record all blocks hash in Chunk, it can quickly tell if a block is in the chunk.
type Chunk struct {
	SnapshotChunks []ledger.SnapshotChunk
	SnapshotRange  [2]*ledger.HashHeight
	AccountRange   map[types.Address][2]*ledger.HashHeight
	HashMap        map[types.Hash]struct{}
	Source         types.BlockSource
}

func newChunk(prevHash types.Hash, prevHeight uint64, endHash types.Hash, endHeight uint64, source types.BlockSource) (c *Chunk) {
	c = &Chunk{
		// chunk will add account block first, then the snapshot block
		SnapshotChunks: make([]ledger.SnapshotChunk, 1, endHeight-prevHeight),
		SnapshotRange: [2]*ledger.HashHeight{
			{prevHeight, prevHash},
			{endHeight, endHash},
		},
		AccountRange: make(map[types.Address][2]*ledger.HashHeight),
		HashMap:      make(map[types.Hash]struct{}),
		Source:       source,
	}

	return c
}

func (c *Chunk) addSnapshotBlock(block *ledger.SnapshotBlock) error {
	var prevHash types.Hash
	var prevHeight uint64
	var chunkLength = len(c.SnapshotChunks)

	if chunkLength < 2 {
		prevHash, prevHeight = c.SnapshotRange[0].Hash, c.SnapshotRange[0].Height
	} else {
		prevBlock := c.SnapshotChunks[chunkLength-2].SnapshotBlock
		prevHash, prevHeight = prevBlock.Hash, prevBlock.Height
	}

	if block.PrevHash != prevHash || block.Height != prevHeight+1 {
		return fmt.Errorf("snapshot blocks not continuous: %s/%d %s/%s/%d", prevHash, prevHeight, block.PrevHash, block.Hash, block.Height)
	}

	if block.Height > c.SnapshotRange[1].Height {
		return fmt.Errorf("chunk overflow: %d %d", c.SnapshotRange[1].Height, block.Height)
	}

	c.SnapshotChunks[chunkLength-1].SnapshotBlock = block
	if chunkLength < cap(c.SnapshotChunks) {
		c.SnapshotChunks = c.SnapshotChunks[:chunkLength+1]
	}
	c.HashMap[block.Hash] = struct{}{}

	return nil
}

func (c *Chunk) addAccountBlock(block *ledger.AccountBlock) error {
	addr := block.AccountAddress
	if rng, ok := c.AccountRange[addr]; ok {
		if rng[1].Hash != block.PrevHash || rng[1].Height+1 != block.Height {
			return fmt.Errorf("account blocks not continuous: %s/%d %s/%s/%d", rng[1].Hash, rng[1].Height, block.PrevHash, block.Hash, block.Height)
		}
		rng[1].Height = block.Height
		rng[1].Hash = block.Hash
	} else {
		c.AccountRange[addr] = [2]*ledger.HashHeight{
			{block.Height - 1, block.PrevHash},
			{block.Height, block.Hash},
		}
	}

	var chunkLength = len(c.SnapshotChunks)
	c.SnapshotChunks[chunkLength-1].AccountBlocks = append(c.SnapshotChunks[chunkLength-1].AccountBlocks, block)
	c.HashMap[block.Hash] = struct{}{}

	return nil
}

func (c *Chunk) done() error {
	endHash, endHeight := c.SnapshotRange[1].Hash, c.SnapshotRange[1].Height

	lastChunk := c.SnapshotChunks[len(c.SnapshotChunks)-1]
	lastBlock := lastChunk.SnapshotBlock
	if lastBlock == nil {
		return errors.New("missing end snapshot block")
	}

	lastHash, lastHeight := lastBlock.Hash, lastBlock.Height

	if endHash != lastHash || endHeight != lastHeight {
		return fmt.Errorf("error end: %s/%d %s/%d", endHash, endHeight, lastHash, lastHeight)
	}

	return nil
}

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

	FetchSnapshotBlocksWithHeight(hash types.Hash, height uint64, count uint64)

	// FetchAccountBlocks address is optional
	FetchAccountBlocks(start types.Hash, count uint64, address *types.Address)

	// FetchAccountBlocksWithHeight add snapshot height
	FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64)
}

// A Syncer implementation can synchronise blocks to peers
type Syncer interface {
	SyncStateSubscriber
	ChunkReader
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
