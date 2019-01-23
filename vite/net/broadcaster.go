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

// A blockStore implementation can store blocks in queue,
// when node is syncing, blocks from remote broadcaster can be stored.
// dequeue these blocks when sync done.
type blockStore interface {
	enqueueAccountBlock(block *ledger.AccountBlock)
	dequeueAccountBlock() (block *ledger.AccountBlock)

	enqueueSnapshotBlock(block *ledger.SnapshotBlock)
	dequeueSnapshotBlock() (block *ledger.SnapshotBlock)
}

type memBlockStore struct {
	rw sync.RWMutex

	aIndex  int
	ablocks []*ledger.AccountBlock

	sIndex  int
	sblocks []*ledger.SnapshotBlock
}

func newMemBlockStore(max int) blockStore {
	return &memBlockStore{
		ablocks: make([]*ledger.AccountBlock, 0, max),
		sblocks: make([]*ledger.SnapshotBlock, 0, max),
	}
}

func (m *memBlockStore) enqueueAccountBlock(block *ledger.AccountBlock) {
	m.rw.Lock()
	defer m.rw.Unlock()

	if len(m.ablocks) < cap(m.ablocks) {
		m.ablocks = append(m.ablocks, block)
	}
}

func (m *memBlockStore) dequeueAccountBlock() (block *ledger.AccountBlock) {
	m.rw.Lock()
	defer m.rw.Unlock()

	if m.aIndex > len(m.ablocks)-1 {
		m.ablocks = m.ablocks[:0]
		return
	}

	block = m.ablocks[m.aIndex]
	m.aIndex++

	return
}

func (m *memBlockStore) enqueueSnapshotBlock(block *ledger.SnapshotBlock) {
	m.rw.Lock()
	defer m.rw.Unlock()

	if len(m.sblocks) < cap(m.sblocks) {
		m.sblocks = append(m.sblocks, block)
	}
}

func (m *memBlockStore) dequeueSnapshotBlock() (block *ledger.SnapshotBlock) {
	m.rw.Lock()
	defer m.rw.Unlock()

	if m.sIndex > len(m.sblocks)-1 {
		m.sblocks = m.sblocks[:0]
		return
	}

	block = m.sblocks[m.sIndex]
	m.sIndex++

	return
}

type broadcaster struct {
	peers *peerSet

	height uint64

	st SyncState

	verifier Verifier
	feed     blockNotifier
	filter   blockFilter

	store blockStore

	mu     sync.Mutex
	statis circle.List // statistic latency of block propagation

	log log15.Logger
}

func newBroadcaster(peers *peerSet, verifier Verifier, feed blockNotifier, store blockStore) *broadcaster {
	return &broadcaster{
		peers:    peers,
		log:      log15.New("module", "net/broadcaster"),
		statis:   circle.NewList(records_24),
		verifier: verifier,
		feed:     feed,
		store:    store,
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

		sender.SeeBlock(block.Hash)

		b.log.Debug(fmt.Sprintf("receive new snapshotblock %s/%d from %s", block.Hash, block.Height, sender.RemoteAddr()))

		// check if block has exist first
		if exist := b.filter.has(block.Hash[:]); exist {
			return nil
		}

		// use the compute hash, because computeHash can`t be forged
		hash := block.ComputeHash()

		// check if has exist or record, return true if has exist
		if exist := b.filter.lookAndRecord(hash[:]); exist {
			return nil
		}

		if err = b.verifier.VerifyNetSb(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new snapshotblock %s/%d from %s error: %v", hash, block.Height, sender.RemoteAddr(), err))
			return err
		}

		b.BroadcastSnapshotBlock(block)

		if b.canNotify() {
			b.feed.notifySnapshotBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueSnapshotBlock(block)
		}

	case NewAccountBlockCode:
		block := new(ledger.AccountBlock)
		if err = block.Deserialize(msg.Payload); err != nil {
			return err
		}

		sender.SeeBlock(block.Hash)
		b.log.Debug(fmt.Sprintf("receive new accountblock %s from %s", block.Hash, sender.RemoteAddr()))

		// check if block has exist first
		if exist := b.filter.has(block.Hash[:]); exist {
			return nil
		}

		// use the compute hash, because computeHash can`t be forged
		hash := block.ComputeHash()

		// check if has exist or record, return true if has exist
		if exist := b.filter.lookAndRecord(hash[:]); exist {
			return nil
		}

		if err = b.verifier.VerifyNetAb(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new accountblock %s from %s error: %v", hash, sender.RemoteAddr(), err))
			return err
		}

		b.BroadcastAccountBlock(block)

		if b.canNotify() {
			b.feed.notifyAccountBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueAccountBlock(block)
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
		for block := b.store.dequeueSnapshotBlock(); block != nil; block = b.store.dequeueSnapshotBlock() {
			b.feed.notifySnapshotBlock(block, types.RemoteBroadcast)
		}
		for block := b.store.dequeueAccountBlock(); block != nil; block = b.store.dequeueAccountBlock() {
			b.feed.notifyAccountBlock(block, types.RemoteBroadcast)
		}
	}
}

func (b *broadcaster) canNotify() bool {
	return b.st != Syncing && b.st != SyncNotStart
}

func (b *broadcaster) setHeight(height uint64) {
	b.height = height
}

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	now := time.Now()
	defer monitor.LogTime("net/broadcast", "SnapshotBlock", now)

	ps := b.peers.UnknownBlock(block.Hash)
	for _, p := range ps {
		p.SendNewSnapshotBlock(block)
	}

	if block.Timestamp != nil && block.Height > b.height {
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
