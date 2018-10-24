package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"time"
)

type broadcaster struct {
	peers *peerSet
	log   log15.Logger
}

func newBroadcaster(peers *peerSet) *broadcaster {
	return &broadcaster{
		peers: peers,
		log:   log15.New("module", "net/broadcaster"),
	}
}

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	defer monitor.LogTime("net/broadcast", "SnapshotBlock", time.Now())

	peers := b.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		peer.SendNewSnapshotBlock(block)
	}

	b.log.Info(fmt.Sprintf("broadcast NewSnapshotBlock %s to %d peers", block.Hash, len(peers)))
}

func (b *broadcaster) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		b.BroadcastSnapshotBlock(block)
	}
}

func (b *broadcaster) BroadcastAccountBlock(block *ledger.AccountBlock) {
	defer monitor.LogTime("net/broadcast", "AccountBlock", time.Now())

	peers := b.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		peer.SendNewAccountBlock(block)
	}

	b.log.Info(fmt.Sprintf("broadcast NewAccountBlock %s to %d peers", block.Hash, len(peers)))
}

func (b *broadcaster) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		b.BroadcastAccountBlock(block)
	}
}
