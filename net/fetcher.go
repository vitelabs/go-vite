package net

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

var errNoSuitablePeer = errors.New("no suitable peer")
var errFetchTimeout = errors.New("timeout")
var errNoResource = errors.New("no resource")

type gid struct {
	index uint32 // atomic
}

func (g *gid) MsgID() MsgId {
	return atomic.AddUint32(&g.index, 1)
}

type MsgIder interface {
	MsgID() MsgId
}

// fetch filter
const maxMark = 3       // times
const timeThreshold = 4 // second
const expiration = 30   // 30s

type peerFetchResult struct {
	status reqState
	t      int64
}

type record struct {
	id       MsgId
	hash     types.Hash
	addAt    int64
	t        int64 // change state time
	mark     int
	st       reqState
	targets  map[peerId]*peerFetchResult // whether failed
	callback func(msg Msg, err error)
}

func (r *record) inc() {
	r.mark++
}

func (r *record) refresh() {
	r.st = reqPending
	r.mark = 0
	r.addAt = time.Now().Unix()
}

func (r *record) reset() {
	r.mark = 0
	r.targets = nil
	r.callback = nil
}

func (r *record) done(peer *Peer, msg Msg, err error) {
	now := time.Now().Unix()

	// when clean timeout, peer is nil, msg is a zero struct, err is timeout
	if peer != nil {
		if result, ok := r.targets[peer.Id]; ok {
			if err != nil {
				result.status = reqError
				result.t = now
			} else {
				result.status = reqDone
				result.t = now
			}
		}
	}

	if r.st != reqPending {
		return
	}

	if err == nil {
		// request done if any peer is done
		r.st = reqDone
		r.t = now

		if r.callback != nil {
			go r.callback(msg, nil)
		}
	} else {
		rest := 0
		for _, ret := range r.targets {
			if ret.status != reqError {
				rest++
			}
		}

		// all peers fail
		if rest == 0 {
			r.st = reqError
			r.t = now

			if r.callback != nil {
				go r.callback(msg, err)
			}
		}
	}
}

func (f *fetcher) clean(t int64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, r := range f.recordsById {
		if (t - r.addAt) > expiration {
			delete(f.recordsByHash, r.hash)
			delete(f.recordsById, r.id)

			r.done(nil, Msg{}, errFetchTimeout)

			// recycle
			for _, ret := range r.targets {
				f.peerFetchResultPool.Put(ret)
			}
			r.reset()
			f.pool.Put(r)
		}
	}
}

// will suppress fetch
func (f *fetcher) hold(hash types.Hash) (r *record, hold bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().Unix()

	var ok bool
	if r, ok = f.recordsByHash[hash]; ok {
		if r.st == reqError {
			r.refresh()
			return r, false
		} else if r.st == reqDone {
			if r.mark >= maxMark && (now-r.t) >= timeThreshold {
				r.refresh()
				return r, false
			}
		} else {
			// pending
			if r.mark >= maxMark*2 && (now-r.addAt) >= timeThreshold*2 {
				r.refresh()
				return r, false
			}
		}

		r.inc()
		return r, true
	}

	r = f.pool.Get().(*record)
	r.st = reqPending
	r.addAt = now
	r.id = f.idGen.MsgID()
	r.hash = hash
	r.targets = make(map[peerId]*peerFetchResult)

	f.recordsByHash[hash] = r
	f.recordsById[r.id] = r

	return r, false
}

func (f *fetcher) add(hash types.Hash) (r *record) {
	r = f.pool.Get().(*record)
	r.st = reqPending
	r.addAt = time.Now().Unix()
	r.targets = make(map[peerId]*peerFetchResult)
	r.id = f.idGen.MsgID()

	f.mu.Lock()
	f.recordsById[r.id] = r
	f.mu.Unlock()

	return r
}

func (f *fetcher) pending(id MsgId, peer *Peer) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r, ok := f.recordsById[id]; ok {
		if peer != nil {
			result := r.targets[peer.Id]
			result.status = reqPending
			result.t = time.Now().Unix()
		}
	}
}

func (f *fetcher) done(id MsgId, peer *Peer, msg Msg, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r, ok := f.recordsById[id]; ok {
		r.done(peer, msg, err)

		if err != nil {
			f.log.Warn(fmt.Sprintf("failed to fetch %s to %s: %v", r.hash, peer, err))
		}
	}
}

// height is 0 when fetch account block
func (f *fetcher) pickTargets(r *record, height uint64, peers *peerSet) peers {
	if height > 10 {
		height -= 10
	}

	ps := peers.pickReliable(height)

	if len(ps) == 0 {
		return nil
	}

	now := time.Now().Unix()

	f.mu.Lock()
	defer f.mu.Unlock()

	var hole bool
	for i, p := range ps {
		if result, ok := r.targets[p.Id]; ok {
			if result.status == reqDone {
				// as target again
				continue
			}

			// not as target
			if result.status == reqWaiting || result.status == reqPending || (result.status == reqError && now-result.t < 10) {
				hole = true
				ps[i] = nil
			}
		}
	}

	if hole {
		var j int
		for _, p := range ps {
			if p == nil {
				continue
			}
			ps[j] = p
			j++
		}

		ps = ps[:j]
	}

	if len(ps) > 3 {
		ps = ps[:3]
	}

	for _, p := range ps {
		result := f.peerFetchResultPool.Get().(*peerFetchResult)
		result.status = reqWaiting
		result.t = now
		r.targets[p.Id] = result
	}

	return ps
}

type fetcher struct {
	idGen               MsgIder
	recordsById         map[MsgId]*record
	recordsByHash       map[types.Hash]*record
	mu                  sync.Mutex
	pool                sync.Pool
	peerFetchResultPool sync.Pool

	peers *peerSet

	st       SyncState
	receiver blockReceiver

	log log15.Logger

	blackBlocks map[types.Hash]struct{}
	sbp         bool

	term chan struct{}
}

func newFetcher(peers *peerSet, receiver blockReceiver, blackBlocks map[types.Hash]struct{}) *fetcher {
	if len(blackBlocks) == 0 {
		blackBlocks = make(map[types.Hash]struct{})
	}

	return &fetcher{
		idGen:         new(gid),
		recordsById:   make(map[MsgId]*record, 1000),
		recordsByHash: make(map[types.Hash]*record, 1000),
		pool: sync.Pool{
			New: func() interface{} {
				return &record{
					targets: make(map[peerId]*peerFetchResult),
				}
			},
		},
		peerFetchResultPool: sync.Pool{
			New: func() interface{} {
				return &peerFetchResult{}
			},
		},
		peers:       peers,
		receiver:    receiver,
		blackBlocks: blackBlocks,
		log:         netLog.New("module", "fetcher"),
	}
}

func (f *fetcher) setSBP(bool2 bool) {
	f.sbp = bool2
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
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-f.term:
			return
		case now := <-ticker.C:
			f.clean(now.Unix())
		}
	}
}

func (f *fetcher) subSyncState(st SyncState) {
	f.st = st
}

func (f *fetcher) name() string {
	return "fetcher"
}

func (f *fetcher) codes() []Code {
	return []Code{CodeSnapshotBlocks, CodeAccountBlocks, CodeException}
}

func (f *fetcher) handle(msg Msg) (err error) {
	switch msg.Code {
	case CodeSnapshotBlocks:
		bs := new(SnapshotBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		for _, block := range bs.Blocks {
			if err = f.receiver.receiveSnapshotBlock(block, types.RemoteFetch); err != nil {
				return err
			}
		}

		if len(bs.Blocks) > 0 {
			f.done(msg.Id, msg.Sender, msg, nil)
			f.log.Info(fmt.Sprintf("receive snapshotblocks %s/%d from %s", bs.Blocks[len(bs.Blocks)-1].Hash, len(bs.Blocks), msg.Sender))
		}

	case CodeAccountBlocks:
		bs := new(AccountBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			msg.Recycle()
			return err
		}
		msg.Recycle()

		for _, block := range bs.Blocks {
			if err = f.receiver.receiveAccountBlock(block, types.RemoteFetch); err != nil {
				return err
			}
		}

		if len(bs.Blocks) > 0 {
			f.done(msg.Id, msg.Sender, msg, nil)
			f.log.Info(fmt.Sprintf("receive accountblocks %s/%d from %s", bs.Blocks[len(bs.Blocks)-1].Hash, len(bs.Blocks), msg.Sender))
		}

	case CodeException:
		f.done(msg.Id, msg.Sender, msg, errNoResource)
	}

	return nil
}

func (f *fetcher) FetchSnapshotBlocks(hash types.Hash, count uint64) {
	if _, ok := f.blackBlocks[hash]; ok {
		return
	}

	if !f.st.syncExited() {
		f.log.Info("in syncing flow, cannot fetch %s/%d", hash, count)
		return
	}

	// been suppressed
	r, hold := f.hold(hash)
	if hold {
		f.log.Info(fmt.Sprintf("fetch suppressed GetSnapshotBlocks %s/%d", hash, count))
		return
	}

	ps := f.pickTargets(r, 0, f.peers)

	if len(ps) == 0 {
		f.log.Warn(fmt.Sprintf("no suit peers for %s/%d", hash, count))
		return
	}

	for _, p := range ps {
		if p != nil {
			m := &GetSnapshotBlocks{
				From:    ledger.HashHeight{Hash: hash},
				Count:   count,
				Forward: false,
			}

			if err := p.send(CodeGetSnapshotBlocks, r.id, m); err != nil {
				f.log.Error(fmt.Sprintf("failed to send GetSnapshotBlocks %s/%d to %s: %v", hash, count, p, err))
				f.done(r.id, p, Msg{}, err)
			} else {
				f.log.Info(fmt.Sprintf("send GetSnapshotBlocks %s/%d to %s", hash, count, p))
				f.pending(r.id, p)
			}
		}
	}
}

// FetchSnapshotBlocksWithHeight fetch blocks:
//  ... count blocks ... {hash, height}
func (f *fetcher) FetchSnapshotBlocksWithHeight(hash types.Hash, height uint64, count uint64) {
	if _, ok := f.blackBlocks[hash]; ok {
		return
	}

	if !f.st.syncExited() {
		f.log.Info("in syncing flow, cannot fetch %s/%d", hash, count)
		return
	}

	r, hold := f.hold(hash)
	// been suppressed
	if hold {
		f.log.Info(fmt.Sprintf("fetch suppressed GetSnapshotBlocks %s/%d", hash, count))
		return
	}

	ps := f.pickTargets(r, height, f.peers)

	if len(ps) == 0 {
		f.log.Warn(fmt.Sprintf("no suit peers for %s/%d", hash, count))
		return
	}

	for _, p := range ps {
		if p != nil {
			m := &GetSnapshotBlocks{
				From:    ledger.HashHeight{Hash: hash},
				Count:   count,
				Forward: false,
			}

			if err := p.send(CodeGetSnapshotBlocks, r.id, m); err != nil {
				f.log.Error(fmt.Sprintf("failed to send GetSnapshotBlocks %s/%d to %s: %v", hash, count, p, err))
				f.done(r.id, p, Msg{}, err)
			} else {
				f.log.Info(fmt.Sprintf("send GetSnapshotBlocks %s/%d to %s", hash, count, p))
				f.pending(r.id, p)
			}
		}
	}
}

func (f *fetcher) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	if _, ok := f.blackBlocks[start]; ok {
		return
	}

	if !f.st.syncExited() {
		f.log.Info("in syncing flow, cannot fetch %s/%d", start, count)
		return
	}

	r, hold := f.hold(start)
	// been suppressed
	if hold {
		f.log.Info(fmt.Sprintf("fetch suppressed GetAccountBlocks %s/%d", start, count))
		return
	}

	ps := f.pickTargets(r, 0, f.peers)

	if len(ps) == 0 {
		f.log.Warn(fmt.Sprintf("no suit peers for %s/%d", start, count))
		return
	}

	for _, p := range ps {
		if p != nil {
			addr := ZERO_ADDRESS
			if address != nil {
				addr = *address
			}
			m := &GetAccountBlocks{
				Address: addr,
				From: ledger.HashHeight{
					Hash: start,
				},
				Count:   count,
				Forward: false,
			}

			if err := p.send(CodeGetAccountBlocks, r.id, m); err != nil {
				f.log.Error(fmt.Sprintf("failed to send GetAccountBlocks %s/%d to %s: %v", start, count, p, err))
				f.done(r.id, p, Msg{}, err)
			} else {
				f.log.Info(fmt.Sprintf("send GetAccountBlocks %s/%d to %s", start, count, p))
				f.pending(r.id, p)
			}
		}
	}
}

func (f *fetcher) FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64) {
	if _, ok := f.blackBlocks[start]; ok {
		return
	}

	if !f.st.syncExited() {
		f.log.Info("in syncing flow, cannot fetch %s/%d", start, count)
		return
	}

	r, hold := f.hold(start)
	// been suppressed
	if hold {
		f.log.Info(fmt.Sprintf("fetch suppressed GetAccountBlocks %s/%d", start, count))
		return
	}

	ps := f.pickTargets(r, sHeight, f.peers)

	if len(ps) == 0 {
		f.log.Warn(fmt.Sprintf("no suit peers for %s/%d", start, count))
		return
	}

	for _, p := range ps {
		if p != nil {
			addr := ZERO_ADDRESS
			if address != nil {
				addr = *address
			}
			m := &GetAccountBlocks{
				Address: addr,
				From: ledger.HashHeight{
					Hash: start,
				},
				Count:   count,
				Forward: false,
			}

			if err := p.send(CodeGetAccountBlocks, r.id, m); err != nil {
				f.log.Error(fmt.Sprintf("failed to send GetAccountBlocks %s/%d to %s: %v", start, count, p, err))
				f.done(r.id, p, Msg{}, err)
			} else {
				f.log.Info(fmt.Sprintf("send GetAccountBlocks %s/%d to %s", start, count, p))
				f.pending(r.id, p)
			}
		}
	}
}

func (f *fetcher) fetchSnapshotBlock(hash types.Hash, peer *Peer, callback func(msg Msg, err error)) {
	m := &GetSnapshotBlocks{
		From: ledger.HashHeight{
			Hash: hash,
		},
		Count:   1,
		Forward: false,
	}

	r := f.add(hash)
	r.callback = callback

	err := peer.send(CodeGetSnapshotBlocks, r.id, m)
	if err != nil {
		f.log.Warn(fmt.Sprintf("failed to query reliable %s to %s", hash, peer))
		f.done(r.id, peer, Msg{}, err)
	} else {
		f.log.Info(fmt.Sprintf("query reliable %s to %s", hash, peer))
	}
}
