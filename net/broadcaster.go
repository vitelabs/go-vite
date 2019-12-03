package net

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/bloom"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/tools/circle"
)

const filterCap = 100000
const rt = 0.0001
const defaultBroadcastTTL = 32

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

func createForardStrategy(strategy string, ps *peerSet) forwardStrategy {
	if strategy == "full" {
		return newFullForwardStrategy(ps)
	}

	return newCrossForwardStrategy(ps, 3, 10)
}

// forwardStrategy will pick peers to forward new blocks
type forwardStrategy interface {
	choosePeers(sender *Peer) peers
}

// fullForwardStrategy will choose all peers as forward targets except sender
type fullForward struct {
	ps *peerSet
}

func newFullForwardStrategy(ps *peerSet) forwardStrategy {
	return &fullForward{
		ps: ps,
	}
}

func (d *fullForward) choosePeers(sender *Peer) (l peers) {
	ourPeers := d.ps.peers()

	for _, p := range ourPeers {
		if p.Id == sender.Id {
			continue
		}
		l = append(l, p)
	}

	return
}

// redForwardStrategy will choose a part of common peers and all particular peers
// the selected common peers should less than min(commonMax, commonRation * commonCount)
type crossForward struct {
	ps *peerSet
	// choose how many peers from the common peers
	commonMax int
	// [0, 100]
	commonRatio int
}

func newCrossForwardStrategy(ps *peerSet, commonMax int, commonRatio int) forwardStrategy {
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

func (d *crossForward) choosePeers(sender *Peer) (l peers) {
	ppMap := sender.peers()
	ourPeers := d.ps.peers()

	return commonPeers(ourPeers, ppMap, sender.Id, d.commonMax, d.commonRatio)
}

func commonPeers(ourPeers peers, ppMap map[peerId]struct{}, sender peerId, commonMax, commonRatio int) (l peers) {
	// cannot get ppMap
	if len(ppMap) == 0 {
		var j int
		for i, p := range ourPeers {
			if p.Id == sender {
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
		if p.Id == sender {
			ourPeers[i] = nil
			continue
		}

		if _, ok = ppMap[p.Id]; ok {
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
			if _, ok = ppMap[p.Id]; ok {
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

//type accountMsgPool struct {
//	sync.Pool
//}
//
//func newAccountMsgPool() *accountMsgPool {
//	return &accountMsgPool{
//		Pool: sync.Pool{
//			New: func() interface{} {
//				return new(NewAccountBlock)
//			},
//		},
//	}
//}
//
//func (p *accountMsgPool) get() *NewAccountBlock {
//	return p.Pool.Get().(*NewAccountBlock)
//}
//
//func (p *accountMsgPool) put(msg *NewAccountBlock) {
//	p.Pool.Put(msg)
//}
//
//type snapshotMsgPool struct {
//	sync.Pool
//}
//
//func newSnapshotMsgPool() *snapshotMsgPool {
//	return &snapshotMsgPool{
//		Pool: sync.Pool{
//			New: func() interface{} {
//				return new(NewSnapshotBlock)
//			},
//		},
//	}
//}
//
//func (p *snapshotMsgPool) get() *NewSnapshotBlock {
//	return p.Pool.Get().(*NewSnapshotBlock)
//}
//
//func (p *snapshotMsgPool) put(msg *NewSnapshotBlock) {
//	p.Pool.Put(msg)
//}
type pickItem struct {
	total     int32
	picked    int32
	failed    int32
	createAt  int64
	resetting int32
	life      int64
}

func (p *pickItem) inc() {
	atomic.AddInt32(&p.total, 1)
}
func (p *pickItem) fail() {
	atomic.AddInt32(&p.failed, 1)
}
func (p *pickItem) pick() {
	atomic.AddInt32(&p.picked, 1)
}
func (p *pickItem) reset(now int64) {
	if !p.expired(now) {
		return
	}

	if atomic.CompareAndSwapInt32(&p.resetting, 0, 1) {
		p.createAt = now
		p.total = 0
		p.failed = 0
		p.picked = 0

		atomic.StoreInt32(&p.resetting, 0)
	}
}
func (p *pickItem) expired(now int64) bool {
	return p.createAt+p.life < now
}

type ringStatic struct {
	rw    sync.RWMutex
	items []*pickItem
	index int32
	size  int32
	d     int64
}

func newRingStatic(size int32, d int64) *ringStatic {
	r := &ringStatic{
		items: make([]*pickItem, size),
		index: 0,
		size:  size,
		d:     d,
	}

	now := time.Now().Unix()
	for i := range r.items {
		r.items[i] = &pickItem{
			total:    0,
			picked:   0,
			failed:   0,
			createAt: now,
			life:     d,
		}
	}

	return r
}
func (r *ringStatic) get() *pickItem {
	now := time.Now().Unix()

	index := atomic.LoadInt32(&r.index)
	item := r.items[index]
	if item.expired(now) {
		index = (index + 1) & (r.size - 1)
		atomic.StoreInt32(&r.index, index)
		item = r.items[index]
		item.reset(now)
	}

	return item
}
func (r *ringStatic) failedRatio() float32 {
	var failed int32
	var pick int32

	for _, item := range r.items {
		failed += item.failed
		pick += item.picked
	}

	if pick == 0 {
		pick++
	}

	return float32(failed) / float32(pick)
}

type broadChainReader interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetConfirmedTimes(blockHash types.Hash) (uint64, error)
}

type broadcaster struct {
	peers *peerSet

	strategy forwardStrategy

	st SyncState

	verifier Verifier
	feed     blockNotifier
	filter   *bloom.Filter

	rings *ringStatic

	store blockStore

	mu        sync.Mutex
	statistic circle.List // statistic latency of block propagation
	chain     broadChainReader

	log log15.Logger
}

func newBroadcaster(peers *peerSet, verifier Verifier, feed blockNotifier,
	store blockStore, strategy forwardStrategy, chain broadChainReader) *broadcaster {
	return &broadcaster{
		peers:     peers,
		statistic: circle.NewList(records24h),
		verifier:  verifier,
		feed:      feed,
		store:     store,
		filter:    bloom.New(filterCap, rt),
		strategy:  strategy,
		chain:     chain,
		rings:     newRingStatic(8, 2),
		log:       netLog.New("module", "broadcaster"),
	}
}

func (b *broadcaster) name() string {
	return "broadcaster"
}

func (b *broadcaster) codes() []Code {
	return []Code{CodeNewAccountBlock, CodeNewSnapshotBlock}
}

func (b *broadcaster) handle(msg Msg) (err error) {
	defer monitor.LogTime("broadcast", "handle", time.Now())

	switch msg.Code {
	case CodeNewSnapshotBlock:
		nb := &NewSnapshotBlock{}
		if err = nb.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		if nb.Block == nil {
			return errMissingBroadcastBlock
		}

		block := nb.Block

		if block.Height+100 < b.chain.GetLatestSnapshotBlock().Height {
			b.log.Warn(fmt.Sprintf("receive new snapshotblock %s/%d from %s: too old", block.Hash, block.Height, msg.Sender))
			return
		}

		b.log.Info(fmt.Sprintf("receive new snapshotblock %s/%d from %s", block.Hash, block.Height, msg.Sender))

		// check if block has exist first
		if exist := b.filter.Test(block.Hash[:]); exist {
			return nil
		}

		// use the compute hash, because computeHash can`t be forged
		hash := block.ComputeHash()

		// check if has exist or record, return true if has exist
		if exist := b.filter.TestAndAdd(hash[:]); exist {
			return nil
		}

		if err = b.verifier.VerifyNetSnapshotBlock(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new snapshotblock %s/%d from %s error: %v", hash, block.Height, msg.Sender, err))
			return err
		}

		if nb.TTL > 0 {
			nb.TTL--
			b.forwardSnapshotBlock(nb, msg.Sender)
		}

		if b.st.syncExited() {
			b.feed.notifySnapshotBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueSnapshotBlock(block)
			b.log.Info(fmt.Sprintf("syncing, don`t give %s/%d to pool", hash, block.Height))
		}

	case CodeNewAccountBlock:
		nb := &NewAccountBlock{}
		if err = nb.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		if nb.Block == nil {
			return errMissingBroadcastBlock
		}

		block := nb.Block

		b.log.Info(fmt.Sprintf("receive new accountblock %s from %s", block.Hash, msg.Sender))

		// check if block has exist first
		if exist := b.filter.Test(block.Hash[:]); exist {
			return nil
		}

		// use the compute hash, because computeHash can`t be forged
		hash := block.ComputeHash()

		// check if has exist or record, return true if has exist
		if exist := b.filter.TestAndAdd(hash[:]); exist {
			return nil
		}

		pickItem := b.rings.get()
		pickItem.inc()

		checkFailedRatio := b.rings.failedRatio()
		shouldCheck := false
		if checkFailedRatio > 0.3 { // if check failed ratio > 0.3, then check all
			shouldCheck = true
		} else if rand.Intn(10) < 3 { // if check failed ratio < 0.3, then check 30%
			shouldCheck = true
		}
		if shouldCheck {
			pickItem.pick()
			if confirmTimes, _ := b.chain.GetConfirmedTimes(hash); confirmTimes > 100 {
				pickItem.fail()
				b.log.Warn(fmt.Sprintf("receive new accountblock %s from %s: confirmed times %d too old", block.Hash, msg.Sender, confirmTimes))
				return
			}
		}

		if err = b.verifier.VerifyNetAccountBlock(block); err != nil {
			b.log.Error(fmt.Sprintf("verify new accountblock %s from %s error: %v", hash, msg.Sender, err))
			return err
		}

		if nb.TTL > 0 {
			nb.TTL--
			b.forwardAccountBlock(nb, msg.Sender)
		}

		if b.st.syncExited() {
			b.feed.notifyAccountBlock(block, types.RemoteBroadcast)
		} else {
			b.store.enqueueAccountBlock(block)
			b.log.Info(fmt.Sprintf("syncing, don`t give %s/%d to pool", hash, block.Height))
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

func (b *broadcaster) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	if b.st == Syncing {
		b.log.Warn(fmt.Sprintf("failed to broadcast snapshotblock %s/%d: syncing", block.Hash, block.Height))
		return
	}

	var msg = &NewSnapshotBlock{
		Block: block,
		TTL:   defaultBroadcastTTL,
	}

	data, err := msg.Serialize()
	if err != nil {
		b.log.Error(fmt.Sprintf("failed to broadcast new snapshotblock %d/%d: %v", block.Hash, block.Height, err))
		return
	}

	b.filter.Add(block.Hash.Bytes())

	var rawMsg = Msg{
		Code:    CodeNewSnapshotBlock,
		Id:      0,
		Payload: data,
	}

	ps := b.peers.peers()
	for _, p := range ps {
		err = p.WriteMsg(rawMsg)
		if err != nil {
			p.catch(err)
			b.log.Error(fmt.Sprintf("failed to broadcast snapshotblock %s/%d to %s: %v", block.Hash, block.Height, p, err))
		} else {
			b.log.Info(fmt.Sprintf("broadcast snapshotblock %s/%d to %s", block.Hash, block.Height, p))
		}
	}
}

func (b *broadcaster) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		b.BroadcastSnapshotBlock(block)
	}
}

func (b *broadcaster) BroadcastAccountBlock(block *ledger.AccountBlock) {
	if b.st == Syncing {
		b.log.Warn(fmt.Sprintf("failed to broadcast accountblock %s/%d: syncing", block.Hash, block.Height))
		return
	}

	var msg = &NewAccountBlock{
		Block: block,
		TTL:   defaultBroadcastTTL,
	}

	data, err := msg.Serialize()
	if err != nil {
		b.log.Error(fmt.Sprintf("failed to broadcast new accountblock %d: %v", block.Hash, err))
		return
	}

	b.filter.Add(block.Hash.Bytes())

	var rawMsg = Msg{
		Code:    CodeNewAccountBlock,
		Id:      0,
		Payload: data,
	}

	ps := b.peers.peers()
	for _, p := range ps {
		err = p.WriteMsg(rawMsg)
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

func (b *broadcaster) forwardSnapshotBlock(msg *NewSnapshotBlock, sender *Peer) {
	data, err := msg.Serialize()
	if err != nil {
		b.log.Error(fmt.Sprintf("failed to forward snapshotblock %d/%d: %v", msg.Block.Hash, msg.Block.Height, err))
		return
	}

	var rawMsg = Msg{
		Code:    CodeNewSnapshotBlock,
		Id:      0,
		Payload: data,
	}

	pl := b.strategy.choosePeers(sender)
	for _, p := range pl {
		if p.knownBlocks.TestAndAdd(msg.Block.Hash.Bytes()) {
			continue
		} else {
			if err = p.WriteMsg(rawMsg); err != nil {
				p.catch(err)
				b.log.Error(fmt.Sprintf("failed to forward snapshotblock %s/%d to %s: %v", msg.Block.Hash, msg.Block.Height, p, err))
			} else {
				b.log.Info(fmt.Sprintf("forward snapshotblock %s/%d to %s", msg.Block.Hash, msg.Block.Height, p))
			}
		}
	}

	if b.chain != nil {
		now := time.Now()
		current := b.chain.GetLatestSnapshotBlock().Height
		if msg.Block.Timestamp != nil && msg.Block.Height > current {
			delta := now.Sub(*msg.Block.Timestamp)
			b.mu.Lock()
			b.statistic.Put(delta.Nanoseconds() / 1e6)
			b.mu.Unlock()
		}
	}
}

func (b *broadcaster) forwardAccountBlock(msg *NewAccountBlock, sender *Peer) {
	data, err := msg.Serialize()
	if err != nil {
		b.log.Error(fmt.Sprintf("failed to forward accountblock %d: %v", msg.Block.Hash, err))
		return
	}

	var rawMsg = Msg{
		Code:    CodeNewAccountBlock,
		Id:      0,
		Payload: data,
	}

	pl := b.strategy.choosePeers(sender)
	for _, p := range pl {
		if p.knownBlocks.TestAndAdd(msg.Block.Hash.Bytes()) {
			continue
		} else {
			if err = p.WriteMsg(rawMsg); err != nil {
				p.catch(err)
				b.log.Error(fmt.Sprintf("failed to forward accountblock %s to %s: %v", msg.Block.Hash, p, err))
			} else {
				b.log.Info(fmt.Sprintf("forward accountblock %s to %s", msg.Block.Hash, p))
			}
		}
	}
}

type broadcastStatus struct {
	checkFailedRatio float32
	latency          []int64
}

func (b *broadcaster) status() broadcastStatus {
	return broadcastStatus{
		checkFailedRatio: b.rings.failedRatio(),
		latency:          b.Statistic(),
	}
}
