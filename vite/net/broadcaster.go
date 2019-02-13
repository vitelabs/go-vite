package net

import (
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vite/net/circle"
)

type broadcaster struct {
	peers *peerSet
	log   log15.Logger

	mu     sync.Mutex
	statis circle.List // statistic latency of block propagation
}

const records_1 = 3600
const records_12 = 12 * records_1
const records_24 = 24 * records_1

func newBroadcaster(peers *peerSet) *broadcaster {
	return &broadcaster{
		peers:  peers,
		log:    log15.New("module", "net/broadcaster"),
		statis: circle.NewList(records_24),
	}
}

func (b *broadcaster) Statistic() []int64 {
	ret := make([]int64, 4)
	var t1, t12, t24 float64
	first := true
	var i int
	records_1f, records_12f := float64(records_1), float64(records_12)

	b.mu.Lock()
	defer b.mu.Unlock()

	count := int64(b.statis.Size())
	count_f := float64(count)
	var v_f float64
	b.statis.TraverseR(func(key circle.Key) bool {
		v, ok := key.(int64)
		if !ok {
			return false
		}

		if first {
			ret[0] = v
			first = false
		}

		v_f = float64(v)
		if count < records_1 {
			t1 += v_f / count_f
		} else if count < records_12 {
			t12 += v_f / count_f

			if i < records_1 {
				t1 += v_f / records_1f
			}
		} else {
			t24 += v_f / count_f

			if i < records_1 {
				t1 += v_f / records_1f
			}
			if i < records_12 {
				t12 += v_f / records_12f
			}
		}

		i++
		return true
	})

	ret[1], ret[2], ret[3] = int64(t1), int64(t12), int64(t24)

	return ret
}

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	now := time.Now()
	defer monitor.LogTime("net/broadcast", "SnapshotBlock", now)

	peers := b.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		peer.SendNewSnapshotBlock(block)
	}

	if block.Timestamp != nil && block.Height > currentHeight {
		delta := now.Sub(*block.Timestamp)
		b.mu.Lock()
		b.statis.Put(delta.Nanoseconds() / 1e6)
		b.mu.Unlock()
	}

	b.log.Debug(fmt.Sprintf("broadcast SnapshotBlock %s/%d to %d peers", block.Hash, block.Height, len(peers)))
}

func (b *broadcaster) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		b.BroadcastSnapshotBlock(block)
	}
}

func (b *broadcaster) BroadcastAccountBlock(block *ledger.AccountBlock) {
	now := time.Now()
	defer monitor.LogTime("net/broadcast", "AccountBlock", now)

	peers := b.peers.UnknownBlock(block.Hash)
	for _, peer := range peers {
		peer.SendNewAccountBlock(block)
	}

	if block.Timestamp != nil {
		delta := now.Sub(*block.Timestamp)
		b.mu.Lock()
		b.statis.Put(delta.Nanoseconds() / 1e6)
		b.mu.Unlock()
	}

	b.log.Debug(fmt.Sprintf("broadcast AccountBlock %s to %d peers", block.Hash, len(peers)))
}

func (b *broadcaster) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		b.BroadcastAccountBlock(block)
	}
}
