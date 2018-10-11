package net

import (
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"time"
)

type Broadcaster interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
	BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock)

	BroadcastAccountBlock(block *ledger.AccountBlock)
	BroadcastAccountBlocks(blocks []*ledger.AccountBlock)
	//BroadcastAccountBlocks(mblocks map[types.Address][]*ledger.AccountBlock)
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

func (b *broadcaster) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		b.BroadcastSnapshotBlock(block)
	}
}

func (b *broadcaster) BroadcastAccountBlock(block *ledger.AccountBlock) {
	t := time.Now()

	peers := b.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		peer.SendNewAccountBlock(block)
	}

	monitor.LogDuration("net/broadcast", "a", time.Now().Sub(t).Nanoseconds())
}

func (b *broadcaster) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		b.BroadcastAccountBlock(block)
	}
}

//func (b *broadcaster) BroadcastAccountBlocks(mblocks map[types.Address][]*ledger.AccountBlock) {
//	t := time.Now()
//
//	for addr, blocks := range mblocks {
//		for _, peer := range b.peers.peers {
//			peer.SendAccountBlocks(&message.AccountBlocks{
//				Address: addr,
//				Blocks:  blocks,
//			}, 0)
//		}
//	}
//
//	monitor.LogDuration("net/broadcast", "as", time.Now().Sub(t).Nanoseconds())
//}
