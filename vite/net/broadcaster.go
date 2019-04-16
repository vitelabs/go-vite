package net

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/vnode"
	"github.com/vitelabs/go-vite/vite/net/circle"
	"github.com/vitelabs/go-vite/vite/net/message"
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

//type ForwardMode string
//
//const (
//	ForwardModeFull  ForwardMode = "full"
//	ForwardModeCross ForwardMode = "cross"
//)

func createForardStrategy(strategy string, ps broadcastPeerSet) forwardStrategy {
	if strategy == "full" {
		return newFullForwardStrategy(ps)
	}

	return newCrossForwardStrategy(ps, 3, 10)
}

// forwardStrategy will pick peers to forward new blocks
type forwardStrategy interface {
	choosePeers(sender broadcastPeer) []broadcastPeer
}

type broadcastPeer interface {
	ID() vnode.NodeID
	peers() map[vnode.NodeID]struct{}
	seeBlock(types.Hash)
	hasBlock(hash types.Hash) bool
	send(c code, id p2p.MsgId, data p2p.Serializable) error
	catch(error)
	sendNewSnapshotBlock(block *ledger.SnapshotBlock) error
	sendNewAccountBlock(block *ledger.AccountBlock) error
}

type broadcastPeerSet interface {
	broadcastPeers() (l []broadcastPeer)
	unknownBlock(types.Hash) (l []broadcastPeer)
}

// fullForwardStrategy will choose all peers as forward targets except sender
type fullForward struct {
	ps broadcastPeerSet
}

func newFullForwardStrategy(ps broadcastPeerSet) forwardStrategy {
	return &fullForward{
		ps: ps,
	}
}

func (d *fullForward) choosePeers(sender broadcastPeer) (l []broadcastPeer) {
	ourPeers := d.ps.broadcastPeers()

	for _, p := range ourPeers {
		if p.ID() == sender.ID() {
			continue
		}
		l = append(l, p)
	}

	return
}

// redForwardStrategy will choose a part of common peers and all particular peers
// the selected common peers should less than min(commonMax, commonRation * commonCount)
type crossForward struct {
	ps broadcastPeerSet
	// choose how many peers from the common peers
	commonMax int
	// [0, 100]
	commonRatio int
}

func newCrossForwardStrategy(ps broadcastPeerSet, commonMax int, commonRatio int) forwardStrategy {
	if commonRatio < 0 {
		commonRatio = 0
	} else if commonRatio > 100 {
		commonRatio = 100
	}

	return &crossForward{
		ps:          ps,
		commonMax:   commonMax,
		commonRatio: commonRatio,
	}
}

func (d *crossForward) choosePeers(sender broadcastPeer) (l []broadcastPeer) {
	ppMap := sender.peers()
	ourPeers := d.ps.broadcastPeers()

	var commonCount int
	var ok bool
	for _, p := range ourPeers {
		if _, ok = ppMap[p.ID()]; ok {
			commonCount++
		}
	}

	var commonMax = d.commonMax

	// commonMax should small than (commonTotal * commonRatio / 100) and d.commonMax
	var commonTotalFromRatio = commonCount * d.commonRatio / 100
	if commonTotalFromRatio == 0 {
		commonTotalFromRatio = 1
	}
	if commonTotalFromRatio < commonMax {
		commonMax = commonTotalFromRatio
	}

	var common int
	for _, p := range ourPeers {
		if _, ok = ppMap[p.ID()]; ok {
			if common > commonMax {
				continue
			}
			common++
		}

		l = append(l, p)
	}

	return
}

var errMissingBroadcastBlock = errors.New("propagation missing block")

type newBlockListener interface {
	onNewAccountBlock(block *ledger.AccountBlock)
	onNewSnapshotBlock(block *ledger.SnapshotBlock)
}

type broadcaster struct {
	peers broadcastPeerSet

	strategy forwardStrategy

	height uint64

	st SyncState

	verifier Verifier
	feed     blockNotifier
	filter   blockFilter

	store blockStore

	listener newBlockListener

	mu        sync.Mutex
	statistic circle.List // statistic latency of block propagation

	log log15.Logger
}

func newBroadcaster(peers broadcastPeerSet, verifier Verifier, feed blockNotifier,
	store blockStore, strategy forwardStrategy, listener newBlockListener) *broadcaster {
	return &broadcaster{
		peers:     peers,
		statistic: circle.NewList(records24h),
		verifier:  verifier,
		feed:      feed,
		store:     store,
		filter:    newBlockFilter(filterCap),
		strategy:  strategy,
		log:       netLog.New("module", "broadcaster"),
		listener:  listener,
	}
}

func (b *broadcaster) name() string {
	return "broadcaster"
}

func (b *broadcaster) codes() []code {
	return []code{NewAccountBlockCode, NewSnapshotBlockCode}
}

func (b *broadcaster) handle(msg p2p.Msg, sender Peer) (err error) {
	defer monitor.LogTime("broadcast", "handle", time.Now())

	switch code(msg.Code) {
	case NewSnapshotBlockCode:
		nb := new(message.NewSnapshotBlock)
		if err = nb.Deserialize(msg.Payload); err != nil {
			return err
		}

		if nb.Block == nil {
			return errMissingBroadcastBlock
		}

		block := nb.Block
		sender.seeBlock(block.Hash)

		if b.listener != nil {
			b.listener.onNewSnapshotBlock(block)
		}

		b.log.Info(fmt.Sprintf("receive new snapshotblock %s/%d from %s", block.Hash, block.Height, sender))

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

		b.log.Info(fmt.Sprintf("record new snapshotblock %s/%d from %s", block.Hash, block.Height, sender))

		if err = b.verifier.VerifyNetSb(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new snapshotblock %s/%d from %s error: %v", hash, block.Height, sender, err))
			return err
		}

		if nb.TTL > 0 {
			nb.TTL--
			b.forwardSnapshotBlock(nb, sender)
		}

		if b.st.syncExited() {
			b.feed.notifySnapshotBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueSnapshotBlock(block)
		}

	case NewAccountBlockCode:
		nb := new(message.NewAccountBlock)
		if err = nb.Deserialize(msg.Payload); err != nil {
			return err
		}

		if nb.Block == nil {
			return errMissingBroadcastBlock
		}

		block := nb.Block
		sender.seeBlock(block.Hash)

		if b.listener != nil {
			b.listener.onNewAccountBlock(block)
		}

		b.log.Info(fmt.Sprintf("receive new accountblock %s from %s", block.Hash, sender))

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

		b.log.Info(fmt.Sprintf("record new accountblock %s/%d from %s", block.Hash, block.Height, sender))

		if err = b.verifier.VerifyNetAb(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new accountblock %s from %s error: %v", hash, sender, err))
			return err
		}

		if nb.TTL > 0 {
			nb.TTL--
			b.forwardAccountBlock(nb, sender)
		}

		if b.st.syncExited() {
			b.feed.notifyAccountBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueAccountBlock(block)
		}
	}

	return nil
}

const records1h = 3600
const records12h = 12 * records1h
const records24h = 24 * records1h

func (b *broadcaster) Statistic() []int64 {
	ret := make([]int64, 4)
	var t1, t12, t24 float64
	first := true
	var i int
	records1hf, records12hf := float64(records1h), float64(records12h)

	b.mu.Lock()
	defer b.mu.Unlock()

	count := int64(b.statistic.Size())
	countF := float64(count)
	var vf float64
	b.statistic.TraverseR(func(key circle.Key) bool {
		v, ok := key.(int64)
		if !ok {
			return false
		}

		if first {
			ret[0] = v
			first = false
		}

		vf = float64(v)
		if count < records1h {
			t1 += vf / countF
		} else if count < records12h {
			t12 += vf / countF

			if i < records1h {
				t1 += vf / records1hf
			}
		} else {
			t24 += vf / countF

			if i < records1h {
				t1 += vf / records1hf
			}
			if i < records12h {
				t12 += vf / records12hf
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

	if b.st.syncExited() {
		for block := b.store.dequeueSnapshotBlock(); block != nil; block = b.store.dequeueSnapshotBlock() {
			b.feed.notifySnapshotBlock(block, types.RemoteBroadcast)
		}
		for block := b.store.dequeueAccountBlock(); block != nil; block = b.store.dequeueAccountBlock() {
			b.feed.notifyAccountBlock(block, types.RemoteBroadcast)
		}
	}
}

func (b *broadcaster) setHeight(height uint64) {
	b.height = height
}

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	now := time.Now()
	defer monitor.LogTime("broadcast", "broadcast", now)

	var err error
	ps := b.peers.unknownBlock(block.Hash)
	for _, p := range ps {
		err = p.sendNewSnapshotBlock(block)
		if err != nil {
			b.log.Error(fmt.Sprintf("failed to broadcast snapshotblock %s/%d to %s: %v", block.Hash, block.Height, p, err))
		} else {
			b.log.Info(fmt.Sprintf("broadcast snapshotblock %s/%d to %s", block.Hash, block.Height, p))
		}
	}

	if block.Timestamp != nil && block.Height > b.height {
		delta := now.Sub(*block.Timestamp)
		b.mu.Lock()
		b.statistic.Put(delta.Nanoseconds() / 1e6)
		b.mu.Unlock()
	}
}

func (b *broadcaster) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		b.BroadcastSnapshotBlock(block)
	}
}

func (b *broadcaster) BroadcastAccountBlock(block *ledger.AccountBlock) {
	now := time.Now()
	defer monitor.LogTime("broadcast", "broadcast", now)

	var err error
	ps := b.peers.unknownBlock(block.Hash)
	for _, p := range ps {
		err = p.sendNewAccountBlock(block)
		if err != nil {
			b.log.Error(fmt.Sprintf("failed to broadcast accountblock %s/%d to %s: %v", block.Hash, block.Height, p, err))
		} else {
			b.log.Info(fmt.Sprintf("broadcast accountblock %s/%d to %s", block.Hash, block.Height, p))
		}
	}
}

func (b *broadcaster) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		b.BroadcastAccountBlock(block)
	}
}

func (b *broadcaster) forwardSnapshotBlock(msg *message.NewSnapshotBlock, sender broadcastPeer) {
	defer monitor.LogTime("broadcast", "forward", time.Now())

	pl := b.strategy.choosePeers(sender)
	for _, p := range pl {
		if p.hasBlock(msg.Block.Hash) {
			continue
		}

		p.seeBlock(msg.Block.Hash)
		if err := p.send(NewSnapshotBlockCode, 0, msg); err != nil {
			p.catch(err)
		}
	}
}

func (b *broadcaster) forwardAccountBlock(msg *message.NewAccountBlock, sender broadcastPeer) {
	defer monitor.LogTime("broadcast", "forward", time.Now())

	pl := b.strategy.choosePeers(sender)
	for _, p := range pl {
		if p.hasBlock(msg.Block.Hash) {
			continue
		}

		p.seeBlock(msg.Block.Hash)
		if err := p.send(NewAccountBlockCode, 0, msg); err != nil {
			p.catch(err)
		}
	}
}
