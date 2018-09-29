package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"time"
)

type Broadcaster interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
	BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock)
	BroadcastAccountBlocks(addr types.Address, blocks []*ledger.AccountBlock)
}

type broadcaster struct {
	peers *peerSet
}

func newBroadcaster(peers *peerSet) *broadcaster {
	return &broadcaster{
		peers: peers,
	}
}

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	t := time.Now()

	peers := b.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		peer.SendNewSnapshotBlock(block)
	}

	monitor.LogDuration("net/broadcast", "s", time.Now().Sub(t).Nanoseconds())
}

func (b *broadcaster) BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock) {
	t := time.Now()

	peers := b.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		peer.SendAccountBlocks(addr, []*ledger.AccountBlock{block}, 0)
	}

	monitor.LogDuration("net/broadcast", "a", time.Now().Sub(t).Nanoseconds())
}

func (b *broadcaster) BroadcastAccountBlocks(addr types.Address, blocks []*ledger.AccountBlock) {
	t := time.Now()

	for _, block := range blocks {
		b.BroadcastAccountBlock(addr, block)
	}

	monitor.LogDuration("net/broadcast", "as", time.Now().Sub(t).Nanoseconds())
}
