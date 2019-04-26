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

const defaultBroadcastTTL = 10

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
	seeBlock(types.Hash) bool
	send(c code, id p2p.MsgId, data p2p.Serializable) error
	catch(error)
}

type broadcastPeerSet interface {
	broadcastPeers() (l []broadcastPeer)
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

	return commonPeers(ourPeers, ppMap, sender.ID(), d.commonMax, d.commonRatio)
}

func commonPeers(ourPeers []broadcastPeer, ppMap map[peerId]struct{}, sender peerId, commonMax, commonRatio int) (l []broadcastPeer) {
	// cannot get ppMap
	if len(ppMap) == 0 {
		var j int
		for i, p := range ourPeers {
			if p.ID() == sender {
				continue
			}
			ourPeers[j] = ourPeers[i]
			j++
		}

		return ourPeers[:j]
	}

	var common, enoughIndex int
	var ok bool
	for i, p := range ourPeers {
		if p.ID() == sender {
			ourPeers[i] = nil
			continue
		}

		if _, ok = ppMap[p.ID()]; ok {
			common++
			if common > commonMax {
				ourPeers[i] = nil
			} else {
				enoughIndex = i // ourPeers has d.commonMax common peers until enoughIndex
			}
		}
	}

	// don`t have enough common peers
	if commonMax > common {
		commonMax = common
	}

	var max = common * commonRatio / 100
	if max == 0 {
		max = 1
	}

	var j int
	if max < commonMax {
		overPeerNum := commonMax - max
		for i, p := range ourPeers[:enoughIndex] {
			// p is sender, so set to nil
			if p == nil {
				continue
			}
			if _, ok = ppMap[p.ID()]; ok {
				ourPeers[i] = nil
				j++
				if j == overPeerNum {
					break
				}
			}
		}
	}

	j = 0
	for i, p := range ourPeers {
		if p == nil {
			continue
		}
		ourPeers[j] = ourPeers[i]
		j++
	}

	return ourPeers[:j]
}

var errMissingBroadcastBlock = errors.New("propagation missing block")

type newBlockListener interface {
	onNewAccountBlock(block *ledger.AccountBlock)
	onNewSnapshotBlock(block *ledger.SnapshotBlock)
}

type accountMsgPool struct {
	sync.Pool
}

func newAccountMsgPool() *accountMsgPool {
	return &accountMsgPool{
		Pool: sync.Pool{
			New: func() interface{} {
				return new(message.NewAccountBlock)
			},
		},
	}
}

func (p *accountMsgPool) get() *message.NewAccountBlock {
	return p.Pool.Get().(*message.NewAccountBlock)
}

func (p *accountMsgPool) put(msg *message.NewAccountBlock) {
	p.Pool.Put(msg)
}

type snapshotMsgPool struct {
	sync.Pool
}

func newSnapshotMsgPool() *snapshotMsgPool {
	return &snapshotMsgPool{
		Pool: sync.Pool{
			New: func() interface{} {
				return new(message.NewSnapshotBlock)
			},
		},
	}
}

func (p *snapshotMsgPool) get() *message.NewSnapshotBlock {
	return p.Pool.Get().(*message.NewSnapshotBlock)
}

func (p *snapshotMsgPool) put(msg *message.NewSnapshotBlock) {
	p.Pool.Put(msg)
}

type broadcaster struct {
	peers broadcastPeerSet

	strategy forwardStrategy

	st SyncState

	verifier Verifier
	feed     blockNotifier
	filter   blockFilter

	store blockStore

	listener newBlockListener

	mu        sync.Mutex
	statistic circle.List // statistic latency of block propagation
	chain     chainReader

	log log15.Logger
}

func newBroadcaster(peers broadcastPeerSet, verifier Verifier, feed blockNotifier,
	store blockStore, strategy forwardStrategy, listener newBlockListener, chain chainReader) *broadcaster {
	return &broadcaster{
		peers:     peers,
		statistic: circle.NewList(records24h),
		verifier:  verifier,
		feed:      feed,
		store:     store,
		filter:    newBlockFilter(filterCap),
		strategy:  strategy,
		chain:     chain,
		listener:  listener,
		log:       netLog.New("module", "broadcaster"),
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
		start := time.Now()
		nb := &message.NewSnapshotBlock{}
		if err = nb.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		if nb.Block == nil {
			return errMissingBroadcastBlock
		}

		unmarshalAt := time.Now()
		b.log.Debug(fmt.Sprintf("unmarshal new snapshotblock %s/%d from %s [%s]", nb.Block.Hash, nb.Block.Height, sender, unmarshalAt.Sub(start)))

		block := nb.Block
		sender.seeBlock(block.Hash)

		if b.listener != nil {
			b.listener.onNewSnapshotBlock(block)
		}

		receiveAt := time.Now()
		b.log.Info(fmt.Sprintf("receive new snapshotblock %s/%d from %s [%s]", block.Hash, block.Height, sender, receiveAt.Sub(unmarshalAt)))

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

		recordAt := time.Now()
		b.log.Info(fmt.Sprintf("record new snapshotblock %s/%d from %s [%s]", block.Hash, block.Height, sender, recordAt.Sub(receiveAt)))

		if err = b.verifier.VerifyNetSb(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new snapshotblock %s/%d from %s error: %v", hash, block.Height, sender, err))
			return err
		}
		verifyAt := time.Now()
		b.log.Debug(fmt.Sprintf("verify new snapshotblock %s/%d from %s [%s]", hash, block.Height, sender, verifyAt.Sub(recordAt)))

		if nb.TTL > 0 {
			nb.TTL--
			b.forwardSnapshotBlock(nb, sender)
		}
		propagateAt := time.Now()
		b.log.Debug(fmt.Sprintf("propagate new snapshotblock %s/%d from %s [%s]", hash, block.Height, sender, propagateAt.Sub(verifyAt)))

		if b.st.syncExited() {
			b.feed.notifySnapshotBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueSnapshotBlock(block)
		}
		b.log.Debug(fmt.Sprintf("notify new snapshotblock %s/%d from %s [%s]", hash, block.Height, sender, time.Now().Sub(propagateAt)))

	case NewAccountBlockCode:
		start := time.Now()
		nb := &message.NewAccountBlock{}
		if err = nb.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		if nb.Block == nil {
			return errMissingBroadcastBlock
		}

		unmarshalAt := time.Now()
		b.log.Debug(fmt.Sprintf("unmarshal new accountblock %s from %s [%s]", nb.Block.Hash, sender, unmarshalAt.Sub(start)))

		block := nb.Block
		sender.seeBlock(block.Hash)

		if b.listener != nil {
			b.listener.onNewAccountBlock(block)
		}

		receiveAt := time.Now()
		b.log.Info(fmt.Sprintf("receive new accountblock %s from %s [%s]", block.Hash, sender, receiveAt.Sub(unmarshalAt)))

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

		recordAt := time.Now()
		b.log.Info(fmt.Sprintf("record new accountblock %s from %s [%s]", block.Hash, sender, recordAt.Sub(receiveAt)))

		if err = b.verifier.VerifyNetAb(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new accountblock %s from %s error: %v", hash, sender, err))
			return err
		}

		verifyAt := time.Now()
		b.log.Debug(fmt.Sprintf("verify new accountblock %s from %s [%s]", hash, sender, verifyAt.Sub(recordAt)))

		if nb.TTL > 0 {
			nb.TTL--
			b.forwardAccountBlock(nb, sender)
		}

		propagateAt := time.Now()
		b.log.Debug(fmt.Sprintf("propagate new accountblock %s from %s [%s]", hash, sender, propagateAt.Sub(verifyAt)))

		if b.st.syncExited() {
			b.feed.notifyAccountBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueAccountBlock(block)
		}

		b.log.Debug(fmt.Sprintf("notify new accountblock %s from %s [%s]", hash, sender, time.Now().Sub(propagateAt)))
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

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	now := time.Now()
	defer monitor.LogTime("broadcast", "broadcast", now)

	var msg = &message.NewSnapshotBlock{
		Block: block,
		TTL:   defaultBroadcastTTL,
	}

	var err error
	ps := b.peers.broadcastPeers()
	for _, p := range ps {
		if p.seeBlock(block.Hash) {
			continue
		}

		err = p.send(NewSnapshotBlockCode, 0, msg)
		if err != nil {
			p.catch(err)
			b.log.Error(fmt.Sprintf("failed to broadcast snapshotblock %s/%d to %s: %v", block.Hash, block.Height, p, err))
		} else {
			b.log.Info(fmt.Sprintf("broadcast snapshotblock %s/%d to %s", block.Hash, block.Height, p))
		}
	}

	if b.chain != nil {
		current := b.chain.GetLatestSnapshotBlock().Height
		if block.Timestamp != nil && block.Height > current {
			delta := now.Sub(*block.Timestamp)
			b.mu.Lock()
			b.statistic.Put(delta.Nanoseconds() / 1e6)
			b.mu.Unlock()
		}
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

	var msg = &message.NewAccountBlock{
		Block: block,
		TTL:   defaultBroadcastTTL,
	}

	var err error
	ps := b.peers.broadcastPeers()
	for _, p := range ps {
		if p.seeBlock(block.Hash) {
			continue
		}

		err = p.send(NewAccountBlockCode, 0, msg)
		if err != nil {
			p.catch(err)
			b.log.Error(fmt.Sprintf("failed to broadcast accountblock %s to %s: %v", block.Hash, p, err))
		} else {
			b.log.Info(fmt.Sprintf("broadcast accountblock %s to %s", block.Hash, p))
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
		if p.seeBlock(msg.Block.Hash) {
			continue
		}

		if err := p.send(NewSnapshotBlockCode, 0, msg); err != nil {
			p.catch(err)
			b.log.Error(fmt.Sprintf("failed to forward snapshotblock %s/%d to %s: %v", msg.Block.Hash, msg.Block.Height, p, err))
		} else {
			b.log.Info(fmt.Sprintf("forward snapshotblock %s/%d to %s", msg.Block.Hash, msg.Block.Height, p))
		}
	}
}

func (b *broadcaster) forwardAccountBlock(msg *message.NewAccountBlock, sender broadcastPeer) {
	defer monitor.LogTime("broadcast", "forward", time.Now())

	pl := b.strategy.choosePeers(sender)
	for _, p := range pl {
		if p.seeBlock(msg.Block.Hash) {
			continue
		}

		if err := p.send(NewAccountBlockCode, 0, msg); err != nil {
			p.catch(err)
			b.log.Error(fmt.Sprintf("failed to forward accountblock %s to %s: %v", msg.Block.Hash, p, err))
		} else {
			b.log.Info(fmt.Sprintf("forward accountblock %s to %s", msg.Block.Hash, p))
		}
	}
}
