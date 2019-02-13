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
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

var errNoSuitablePeer = errors.New("no suitable peer")

type gid struct {
	index uint64 // atomic
}

func (g *gid) MsgID() uint64 {
	return atomic.AddUint64(&g.index, 1)
}

type MsgIder interface {
	MsgID() uint64
}

// a fetchPolicy implementation can choose suitable peers to fetch blocks
type fetchPolicy interface {
	pickAccount(height uint64) (l []Peer)
	pickSnap() Peer
}

// a fp is fetchPolicy implementation
type fp struct {
	peers *peerSet
}

func (p *fp) pickAccount(height uint64) []Peer {
	var l, taller []Peer

	all := p.peers.Peers()
	total := len(all)

	if total == 0 {
		return l
	}

	// best
	var peer Peer
	var maxHeight uint64
	for _, p := range all {
		ph := p.Height()

		if ph > maxHeight {
			maxHeight = ph
			peer = p
		}

		if ph >= height {
			taller = append(taller, p)
		}
	}

	l = append(l, peer)

	// random a peer
	ran := rand.Intn(total)
	if peer = all[ran]; peer != l[0] {
		l = append(l, peer)
	}

	// taller
	//if len(taller) > 0 {
	//	ran = rand.Intn(len(taller))
	//	peer = taller[ran]
	//
	//	for _, p := range l {
	//		if peer == p {
	//			return l
	//		}
	//	}
	//
	//	l = append(l, peer)
	//}

	return l
}

func (p *fp) pickSnap() Peer {
	return p.peers.BestPeer()
}

// fetch filter
const maxMark = 3
const timeThreshold = int64(3 * time.Second)

type record struct {
	addAt  int64
	doneAt int64
	mark   int
	_done  bool
}

func (r *record) inc() {
	r.mark += 1
}

func (r *record) reset() {
	r.mark = 0
	r._done = false
	r.addAt = time.Now().Unix()
}

func (r *record) done() {
	r.doneAt = time.Now().Unix()
	r._done = true
}

type filter struct {
	records map[types.Hash]record
	lock    sync.RWMutex
}

func newFilter() *filter {
	return &filter{
		records: make(map[types.Hash]record, 3600),
	}
}

func (f *filter) clean(t int64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for hash, r := range f.records {
		if r._done && (t-r.doneAt) > int64(timeThreshold) {
			delete(f.records, hash)
		}
	}
}

// will suppress fetch
func (f *filter) hold(hash types.Hash) bool {
	f.lock.Lock()
	defer f.lock.Unlock()

	now := time.Now().Unix()
	if r, ok := f.records[hash]; ok {
		if r._done {
			if r.mark >= maxMark && (now-r.doneAt) >= timeThreshold {
				r.reset()
				return false
			}
		} else {
			if r.mark >= maxMark*2 && (now-r.addAt) >= timeThreshold*2 {
				r.reset()
				return false
			}
		}

		r.inc()
	} else {
		f.records[hash] = record{addAt: now}
		return false
	}

	return true
}

func (f *filter) done(hash types.Hash) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if r, ok := f.records[hash]; ok {
		r.done()
	} else {
		r = record{addAt: time.Now().Unix()}
		r.done()
		f.records[hash] = r
	}
}

func (f *filter) has(hash types.Hash) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()

	r, ok := f.records[hash]
	return ok && r._done
}

func (f *filter) fail(hash types.Hash) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	if r, ok := f.records[hash]; ok {
		if r._done {
			return
		}

		delete(f.records, hash)
	}
}

type fetcher struct {
	filter *filter

	st       SyncState
	verifier Verifier
	notifier blockNotifier

	policy fetchPolicy
	pool   MsgIder

	log log15.Logger

	term chan struct{}
}

func newFetcher(peers *peerSet, pool MsgIder, verifier Verifier, notifier blockNotifier) *fetcher {
	return &fetcher{
		filter:   newFilter(),
		policy:   &fp{peers},
		pool:     pool,
		notifier: notifier,
		verifier: verifier,
		log:      log15.New("module", "net/fetcher"),
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
	ticker := time.NewTicker(5 * time.Minute)
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

func (f *fetcher) canFetch() bool {
	return f.st == Syncdone || f.st == Syncerr
}

func (f *fetcher) ID() string {
	return "fetcher"
}

func (f *fetcher) Cmds() []ViteCmd {
	return []ViteCmd{SnapshotBlocksCode, AccountBlocksCode}
}

func (f *fetcher) Handle(msg *p2p.Msg, sender Peer) (err error) {
	switch ViteCmd(msg.Cmd) {
	case SnapshotBlocksCode:
		bs := new(message.SnapshotBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			return err
		}

		for _, block := range bs.Blocks {
			if err = f.verifier.VerifyNetSb(block); err != nil {
				return err
			}

			f.filter.done(block.Hash)
			f.notifier.notifySnapshotBlock(block, types.RemoteFetch)
		}

	case AccountBlocksCode:
		bs := new(message.AccountBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			return err
		}

		for _, block := range bs.Blocks {
			if err = f.verifier.VerifyNetAb(block); err != nil {
				return err
			}

			f.filter.done(block.Hash)
			f.notifier.notifyAccountBlock(block, types.RemoteFetch)
		}
	}

	return nil
}

func (f *fetcher) FetchSnapshotBlocks(start types.Hash, count uint64) {
	monitor.LogEvent("net/fetch", "GetSnapshotBlocks")

	// been suppressed
	if f.filter.hold(start) {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetSnapshotBlocks[hash %s, count %d]", start, count))
		return
	}

	if !f.canFetch() {
		f.log.Debug("not ready")
		return
	}

	if p := f.policy.pickSnap(); p != nil {
		m := &message.GetSnapshotBlocks{
			From:    ledger.HashHeight{Hash: start},
			Count:   count,
			Forward: false,
		}

		id := f.pool.MsgID()

		if err := p.Send(GetSnapshotBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("send %s to %s error: %v", m, p, err))
		} else {
			f.log.Info(fmt.Sprintf("send %s to %s done", m, p))
		}
		monitor.LogEvent("net/fetch", "GetSnapshotBlocks_Send")
	} else {
		f.log.Error(errNoSuitablePeer.Error())
	}
}

func (f *fetcher) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	monitor.LogEvent("net/fetch", "GetAccountBlocks")

	// been suppressed
	if f.filter.hold(start) {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetAccountBlocks[hash %s, count %d]", start, count))
		return
	}

	if !f.canFetch() {
		f.log.Warn("not ready")
		return
	}

	if peerList := f.policy.pickAccount(0); len(peerList) != 0 {
		addr := NULL_ADDRESS
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

		id := f.pool.MsgID()

		for _, p := range peerList {
			if err := p.Send(GetAccountBlocksCode, id, m); err != nil {
				f.log.Error(fmt.Sprintf("send %s to %s error: %v", m, p, err))
			} else {
				f.log.Info(fmt.Sprintf("send %s to %s done", m, p))
			}
			monitor.LogEvent("net/fetch", "GetAccountBlocks_Send")
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
	}
}

func (f *fetcher) FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64) {
	monitor.LogEvent("net/fetch", "GetAccountBlocks_S")

	// been suppressed
	if f.filter.hold(start) {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetAccountBlocks[hash %s, count %d]", start, count))
		return
	}

	if !f.canFetch() {
		f.log.Warn("not ready")
		return
	}

	if peerList := f.policy.pickAccount(sHeight); len(peerList) != 0 {
		addr := NULL_ADDRESS
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

		id := f.pool.MsgID()

		for _, p := range peerList {
			if err := p.Send(GetAccountBlocksCode, id, m); err != nil {
				f.log.Error(fmt.Sprintf("send %s to %s error: %v", m, p, err))
			} else {
				f.log.Info(fmt.Sprintf("send %s to %s done", m, p))
			}
			monitor.LogEvent("net/fetch", "GetAccountBlocks_Send")
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
	}
}
