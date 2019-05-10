package net

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

var errNoSuitablePeer = errors.New("no suitable peer")

type gid struct {
	index uint32 // atomic
}

func (g *gid) MsgID() p2p.MsgId {
	return atomic.AddUint32(&g.index, 1)
}

type MsgIder interface {
	MsgID() p2p.MsgId
}

type fetchTarget struct {
	peers interface {
		sortPeers() peers
	}
	chooseTop func() bool
}

func (p *fetchTarget) account(height uint64, r *record) (pe Peer) {
	var total, top, ran int

	ps := p.peers.sortPeers()

	// remove failed peers
	var i, j int
	var id peerId
Loop:
	for i, pe = range ps {
		if pe.Height()+heightDelta < height {
			break Loop
		}

		id = pe.ID()
		for _, fid := range r.fails {
			if id == fid {
				continue Loop
			}
		}

		ps[j] = ps[i]
		j++
	}
	ps = ps[:j]

	total = len(ps)
	if total == 0 {
		return nil
	}

	top = total / 3

	if p.chooseTop() && top > 1 {
		ran = rand.Intn(top)
	} else {
		ran = rand.Intn(total)
	}

	pe = ps[ran]

	return
}

const heightDelta = 10

func (p *fetchTarget) snapshot(height uint64, r *record) (pe Peer) {
	ps := p.peers.sortPeers()

	var id peerId
Loop:
	for _, pe = range ps {
		if pe.Height()+heightDelta < height {
			return nil
		}

		id = pe.ID()
		for _, fid := range r.fails {
			if id == fid {
				continue Loop
			}
		}

		return pe
	}

	return nil
}

// fetch filter
const maxMark = 3       // times
const timeThreshold = 3 // second
const expiration = 60   // 60s
const keepFails = 3
const fail3wait = 10 // fail 3 times, should wait 10s for next fetch

type record struct {
	id         p2p.MsgId
	addAt      int64
	t          int64
	mark       int
	blockCount int // fetch block count
	st         reqState
	fails      []peerId
}

func (r *record) inc() {
	r.mark += 1
}

func (r *record) reset() {
	r.mark = 0
	r.st = reqPending
	r.addAt = time.Now().Unix()
	r.fails = r.fails[:0]
}

func (r *record) done() {
	r.st = reqDone
	r.t = time.Now().Unix()
}

func (r *record) fail(pid peerId) {
	// record maybe done
	if r.st == reqPending {
		r.st = reqError
		r.t = time.Now().Unix()

		fails := len(r.fails)
		if fails < keepFails {
			// add one room
			r.fails = r.fails[:fails+1]
			fails++
		}

		for i := fails - 1; i > 0; i-- {
			r.fails[i] = r.fails[i-1]
		}
		// add to first
		r.fails[0] = pid
	}
}

func (r *record) failNoPeers() {
	if r.st == reqPending {
		r.st = reqError
		r.t = time.Now().Unix()
	}
}

type filter struct {
	idGen    MsgIder
	idToHash map[p2p.MsgId]types.Hash
	records  map[types.Hash]*record
	mu       sync.Mutex
	pool     sync.Pool
}

func newFilter() *filter {
	return &filter{
		idGen:    new(gid),
		idToHash: make(map[p2p.MsgId]types.Hash, 1000),
		records:  make(map[types.Hash]*record, 1000),
		pool: sync.Pool{
			New: func() interface{} {
				return &record{
					fails: make([]peerId, 0, keepFails),
				}
			},
		},
	}
}

func (f *filter) clean(t int64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for hash, r := range f.records {
		if (t - r.addAt) > expiration {
			delete(f.records, hash)
			delete(f.idToHash, r.id)

			f.pool.Put(r)
		}
	}
}

// will suppress fetch
func (f *filter) hold(hash types.Hash) (r *record, hold bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().Unix()

	var ok bool
	if r, ok = f.records[hash]; ok {
		if r.st == reqError {
			if len(r.fails) < keepFails {
				r.st = reqPending
				return r, false
			}

			if now-r.t >= fail3wait {
				r.reset()
				return r, false
			}
		} else if r.st == reqDone {
			if r.mark >= maxMark && (now-r.t) >= timeThreshold {
				r.reset()
				return r, false
			}
		} else {
			// pending
			if r.mark >= maxMark*2 && (now-r.addAt) >= timeThreshold*2 {
				r.reset()
				return r, false
			}
		}

		r.inc()
		return r, true
	}

	r = f.pool.Get().(*record)
	r.addAt = now
	r.mark = 0
	r.id = f.idGen.MsgID()
	r.st = reqPending
	r.fails = r.fails[:0]

	f.records[hash] = r
	f.idToHash[r.id] = hash

	return r, false
}

func (f *filter) done(id p2p.MsgId) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if hash, ok := f.idToHash[id]; ok {
		var r *record
		if r, ok = f.records[hash]; ok {
			r.done()
		}
	}
}

func (f *filter) fail(id p2p.MsgId, pid peerId) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if hash, ok := f.idToHash[id]; ok {
		var r *record
		if r, ok = f.records[hash]; ok {
			r.fail(pid)
		}
	}
}

func (f *filter) failNoPeers(id p2p.MsgId) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if hash, ok := f.idToHash[id]; ok {
		var r *record
		if r, ok = f.records[hash]; ok {
			r.failNoPeers()
		}
	}
}

type fetcher struct {
	filter *filter

	st       SyncState
	receiver blockReceiver

	policy *fetchTarget

	log log15.Logger

	term chan struct{}
}

func newFetcher(peers *peerSet, receiver blockReceiver) *fetcher {
	return &fetcher{
		filter: newFilter(),
		policy: &fetchTarget{peers, func() bool {
			return rand.Intn(10) > 5
		}},
		receiver: receiver,
		log:      netLog.New("module", "fetcher"),
	}
}

func (f *fetcher) start() {
	f.term = make(chan struct{})
	go f.cleanLoop()
}

func (f *fetcher) stop() {
	if f.term == nil {
		return
	}

	select {
	case <-f.term:
	default:
		close(f.term)
	}
}

func (f *fetcher) cleanLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-f.term:
			return
		case now := <-ticker.C:
			f.filter.clean(now.Unix())
		}
	}
}

func (f *fetcher) subSyncState(st SyncState) {
	f.st = st
}

func (f *fetcher) name() string {
	return "fetcher"
}

func (f *fetcher) codes() []p2p.Code {
	return []p2p.Code{p2p.CodeSnapshotBlocks, p2p.CodeAccountBlocks, p2p.CodeException}
}

func (f *fetcher) handle(msg p2p.Msg, sender Peer) (err error) {
	switch msg.Code {
	case p2p.CodeSnapshotBlocks:
		bs := new(message.SnapshotBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		f.log.Info(fmt.Sprintf("receive %d snapshotblocks from %s", len(bs.Blocks), sender.String()))

		for _, block := range bs.Blocks {
			if err = f.receiver.receiveSnapshotBlock(block, types.RemoteFetch); err != nil {
				return err
			}
		}

		f.log.Info(fmt.Sprintf("receive %d snapshotblocks from %s done", len(bs.Blocks), sender.String()))

		if len(bs.Blocks) > 0 {
			f.filter.done(msg.Id)
		}

	case p2p.CodeAccountBlocks:
		bs := new(message.AccountBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		f.log.Info(fmt.Sprintf("receive %d accountblocks from %s", len(bs.Blocks), sender.String()))

		for _, block := range bs.Blocks {
			if err = f.receiver.receiveAccountBlock(block, types.RemoteFetch); err != nil {
				return err
			}
		}

		f.log.Info(fmt.Sprintf("receive %d accountblocks from %s done", len(bs.Blocks), sender.String()))

		if len(bs.Blocks) > 0 {
			f.filter.done(msg.Id)
		}

	case p2p.CodeException:
		f.filter.fail(msg.Id, sender.ID())
	}

	return nil
}

func (f *fetcher) FetchSnapshotBlocks(hash types.Hash, count uint64) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	// been suppressed
	r, hold := f.filter.hold(hash)
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetSnapshotBlocks[hash %s, count %d]", hash, count))
		return
	}

	if p := f.policy.snapshot(0, r); p != nil {
		m := &message.GetSnapshotBlocks{
			From:    ledger.HashHeight{Hash: hash},
			Count:   count,
			Forward: false,
		}

		if err := p.send(p2p.CodeGetSnapshotBlocks, r.id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetSnapshotBlocks[hash %s, count %d] to %s: %v", hash, count, p, err))
			f.filter.fail(r.id, p.ID())
		} else {
			f.log.Info(fmt.Sprintf("send GetSnapshotBlocks[hash %s, count %d] to %s", hash, count, p))
		}
	} else {
		f.log.Error(fmt.Sprintf("failed to fetch GetSnapshotBlocks[hash %s, count %d]: %v", hash, count, errNoSuitablePeer))
		f.filter.failNoPeers(r.id)
	}
}

// FetchSnapshotBlocksWithHeight fetch blocks:
//  ... count blocks ... {hash, height}
func (f *fetcher) FetchSnapshotBlocksWithHeight(hash types.Hash, height uint64, count uint64) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	r, hold := f.filter.hold(hash)
	// been suppressed
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetSnapshotBlocks[hash %s, count %d]", hash, count))
		return
	}

	if p := f.policy.snapshot(height, r); p != nil {
		m := &message.GetSnapshotBlocks{
			From:    ledger.HashHeight{Hash: hash},
			Count:   count,
			Forward: false,
		}

		if err := p.send(p2p.CodeGetSnapshotBlocks, r.id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetSnapshotBlocks[hash %s, count %d] to %s: %v", hash, count, p, err))
			f.filter.fail(r.id, p.ID())
		} else {
			f.log.Info(fmt.Sprintf("send GetSnapshotBlocks[hash %s, count %d] to %s", hash, count, p))
		}
	} else {
		f.log.Error(fmt.Sprintf("failed to fetch GetSnapshotBlocks[hash %s, count %d]: %v", hash, count, errNoSuitablePeer))
		f.filter.failNoPeers(r.id)
	}
}

func (f *fetcher) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	r, hold := f.filter.hold(start)
	// been suppressed
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetAccountBlocks[hash %s, count %d]", start, count))
		return
	}

	if p := f.policy.account(0, r); p != nil {
		addr := nilAddress
		if address != nil {
			addr = *address
		}
		m := &message.GetAccountBlocks{
			Address: addr,
			From: ledger.HashHeight{
				Hash: start,
			},
			Count:   count,
			Forward: false,
		}

		if err := p.send(p2p.CodeGetAccountBlocks, r.id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetAccountBlocks[hash %s, count %d] to %s: %v", start, count, p, err))
			f.filter.fail(r.id, p.ID())
		} else {
			f.log.Info(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s", start, count, p))
		}
	} else {
		f.log.Error(fmt.Sprintf("failed to fetch GetAccountBlocks[hash %s, count %d]: %v", start, count, errNoSuitablePeer))
		f.filter.failNoPeers(r.id)
	}
}

func (f *fetcher) FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	r, hold := f.filter.hold(start)
	// been suppressed
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetAccountBlocks[hash %s, count %d]", start, count))
		return
	}

	if p := f.policy.account(sHeight, r); p != nil {
		addr := nilAddress
		if address != nil {
			addr = *address
		}
		m := &message.GetAccountBlocks{
			Address: addr,
			From: ledger.HashHeight{
				Hash: start,
			},
			Count:   count,
			Forward: false,
		}

		if err := p.send(p2p.CodeGetAccountBlocks, r.id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetAccountBlocks[hash %s, count %d] to %s: %v", start, count, p, err))
			f.filter.fail(r.id, p.ID())
		} else {
			f.log.Info(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s", start, count, p))
		}
	} else {
		f.log.Error(fmt.Sprintf("failed to fetch GetAccountBlocks[hash %s, count %d]: %v", start, count, errNoSuitablePeer))
		f.filter.failNoPeers(r.id)
	}
}
