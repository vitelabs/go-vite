package net

import (
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/circle"
)

type broadcaster struct {
	cacheSblocks []*ledger.SnapshotBlock
	cacheAblocks []*ledger.AccountBlock

	peers *peerSet
	log   log15.Logger

	st SyncState

	verifier Verifier
	bn       *blockNotifier

	mu     sync.Mutex
	statis circle.List // statistic latency of block propagation
}

func newBroadcaster(peers *peerSet, verifier Verifier, bn *blockNotifier) *broadcaster {
	return &broadcaster{
		cacheSblocks: make([]*ledger.SnapshotBlock, 0, 1000),
		cacheAblocks: make([]*ledger.AccountBlock, 0, 1000),
		peers:        peers,
		log:          log15.New("module", "net/broadcaster"),
		statis:       circle.NewList(records_24),
		verifier:     verifier,
		bn:           bn,
	}
}

func (b *broadcaster) ID() string {
	return "broadcaster"
}

func (b *broadcaster) Cmds() []ViteCmd {
	return []ViteCmd{NewAccountBlockCode, NewSnapshotBlockCode}
}

func (b *broadcaster) Handle(msg *p2p.Msg, sender Peer) (err error) {
	switch ViteCmd(msg.Cmd) {
	case NewSnapshotBlockCode:
		block := new(ledger.SnapshotBlock)
		if err = block.Deserialize(msg.Payload); err != nil {
			return err
		}

		hasSeen := b.bn.knownBlocks.lookAndRecord(block.Hash[:])

		if hasSeen {
			return nil
		}

		if err = b.verifier.VerifyNetSb(block); err != nil {
			return err
		}

		b.log.Info(fmt.Sprintf("receive new snapshotblock %s/%d from %s", block.Hash, block.Height, sender.RemoteAddr()))

		sender.SeeBlock(block.Hash)

		b.BroadcastSnapshotBlock(block)

		if b.canNotify() {
			b.bn.NotifySnapshotBlock(block, types.RemoteBroadcast)
		} else if len(b.cacheSblocks) < cap(b.cacheSblocks) {
			b.cacheSblocks = append(b.cacheSblocks, block)
		}

	case NewAccountBlockCode:
		block := new(ledger.AccountBlock)
		if err = block.Deserialize(msg.Payload); err != nil {
			return err
		}

		hasSeen := b.bn.knownBlocks.lookAndRecord(block.Hash[:])

		if hasSeen {
			return nil
		}

		if err = b.verifier.VerifyNetAb(block); err != nil {
			return err
		}

		b.log.Info(fmt.Sprintf("receive new accountblock %s from %s", block.Hash, sender.RemoteAddr()))

		sender.SeeBlock(block.Hash)

		b.BroadcastAccountBlock(block)

		if b.canNotify() {
			b.bn.NotifyAccountBlock(block, types.RemoteBroadcast)
		} else if len(b.cacheAblocks) < cap(b.cacheAblocks) {
			b.cacheAblocks = append(b.cacheAblocks, block)
		}
	}

	return nil
}

const records_1 = 3600
const records_12 = 12 * records_1
const records_24 = 24 * records_1

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

func (b *broadcaster) subSyncState(st SyncState) {
	b.st = st

	if b.canNotify() {
		for _, block := range b.cacheAblocks {
			b.bn.NotifyAccountBlock(block, types.RemoteBroadcast)
		}
		for _, block := range b.cacheSblocks {
			b.bn.NotifySnapshotBlock(block, types.RemoteBroadcast)
		}

		b.cacheSblocks = b.cacheSblocks[:0]
		b.cacheAblocks = b.cacheAblocks[:0]
	}
}

func (b *broadcaster) canNotify() bool {
	return b.st != Syncing && b.st != SyncNotStart
}

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	now := time.Now()
	defer monitor.LogTime("net/broadcast", "SnapshotBlock", now)

	ps := b.peers.UnknownBlock(block.Hash)
	for _, p := range ps {
		p.SendNewSnapshotBlock(block)
	}

	if block.Timestamp != nil && block.Height > currentHeight {
		delta := now.Sub(*block.Timestamp)
		b.mu.Lock()
		b.statis.Put(delta.Nanoseconds() / 1e6)
		b.mu.Unlock()
	}

	b.log.Debug(fmt.Sprintf("broadcast SnapshotBlock %s/%d to %d peers", block.Hash, block.Height, len(ps)))
}

func (b *broadcaster) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		b.BroadcastSnapshotBlock(block)
	}
}

func (b *broadcaster) BroadcastAccountBlock(block *ledger.AccountBlock) {
	now := time.Now()
	defer monitor.LogTime("net/broadcast", "AccountBlock", now)

	ps := b.peers.UnknownBlock(block.Hash)
	for _, p := range ps {
		p.SendNewAccountBlock(block)
	}

	if block.Timestamp != nil {
		delta := now.Sub(*block.Timestamp)
		b.mu.Lock()
		b.statis.Put(delta.Nanoseconds() / 1e6)
		b.mu.Unlock()
	}

	b.log.Debug(fmt.Sprintf("broadcast AccountBlock %s to %d peers", block.Hash, len(ps)))
}

func (b *broadcaster) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		b.BroadcastAccountBlock(block)
	}
}
