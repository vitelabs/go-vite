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
	index uint32 // atomic
}

func (g *gid) MsgID() p2p.MsgId {
	return atomic.AddUint32(&g.index, 1)
}

type MsgIder interface {
	MsgID() p2p.MsgId
}

// chain fetchPolicy implementation can choose suitable peers to fetch blocks
type fetchPolicy interface {
	accountTargets(height uint64) (l []Peer)
	snapshotTarget(height uint64) Peer
}

// chain fp is fetchPolicy implementation
type fp struct {
	peers *peerSet
}

// best peer , random peer, chain random taller peer
func (p *fp) accountTargets(height uint64) []Peer {
	var l, taller []Peer

	peerList := p.peers.peers()
	total := len(peerList)

	if total == 0 {
		return l
	}

	// best
	var per Peer
	var maxHeight uint64
	for _, p := range peerList {
		peerHeight := p.height()

		if peerHeight > maxHeight {
			maxHeight = peerHeight
			per = p
		}

		if peerHeight >= height {
			taller = append(taller, p)
		}
	}

	l = append(l, per)

	// random
	ran := rand.Intn(total)
	if per = peerList[ran]; per != l[0] {
		l = append(l, per)
	}

	// taller
	if len(taller) > 0 {
		ran = rand.Intn(len(taller))
		per = taller[ran]

		for _, per2 := range l {
			if per == per2 {
				return l
			}
		}

		l = append(l, per)
	}

	return l
}

func (p *fp) snapshotTarget(height uint64) Peer {
	if height == 0 {
		return p.peers.bestPeer()
	}

	ps := p.peers.pick(height)
	if len(ps) == 0 {
		return nil
	}

	i := rand.Intn(len(ps))
	return ps[i]
}

// fetch filter
const maxMark = 3       // times
const timeThreshold = 3 // second

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
	records map[types.Hash]*record
	lock    sync.RWMutex
}

func newFilter() *filter {
	return &filter{
		records: make(map[types.Hash]*record, 3600),
	}
}

func (f *filter) clean(t int64) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for hash, r := range f.records {
		if r._done && (t-r.doneAt) > timeThreshold {
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
		f.records[hash] = &record{addAt: now, mark: 1}
		return false
	}

	return true
}

func (f *filter) done(hash types.Hash) {
	f.lock.Lock()
	defer f.lock.Unlock()

	if r, ok := f.records[hash]; ok {
		r.done()
	}
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

func (f *fetcher) Cmds() []code {
	return []code{SnapshotBlocksCode, AccountBlocksCode}
}

func (f *fetcher) Handle(msg p2p.Msg, sender Peer) (err error) {
	switch code(msg.Code) {
	case SnapshotBlocksCode:
		bs := new(message.SnapshotBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			return err
		}

		for _, block := range bs.Blocks {
			if err = f.verifier.VerifyNetSb(block); err != nil {
				return err
			}

			f.notifier.notifySnapshotBlock(block, types.RemoteFetch)
		}

		if len(bs.Blocks) > 0 {
			f.filter.done(bs.Blocks[len(bs.Blocks)-1].Hash)
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

			f.notifier.notifyAccountBlock(block, types.RemoteFetch)
		}

		if len(bs.Blocks) > 0 {
			f.filter.done(bs.Blocks[len(bs.Blocks)-1].Hash)
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

	if p := f.policy.snapshotTarget(0); p != nil {
		m := &message.GetSnapshotBlocks{
			From:    ledger.HashHeight{Hash: start},
			Count:   count,
			Forward: false,
		}

		id := f.pool.MsgID()

		if err := p.send(GetSnapshotBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("send GetSnapshotBlocks[hash %s, count %d] to %s error: %v", start, count, p.Address(), err))
		} else {
			f.log.Info(fmt.Sprintf("send GetSnapshotBlocks[hash %s, count %d] to %s", start, count, p.Address()))
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

	if peerList := f.policy.accountTargets(0); len(peerList) != 0 {
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
			if err := p.send(GetAccountBlocksCode, id, m); err != nil {
				f.log.Error(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s error: %v", start, count, p.Address(), err))
			} else {
				f.log.Info(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s", start, count, p.Address()))
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

	if peerList := f.policy.accountTargets(sHeight); len(peerList) != 0 {
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
			if err := p.send(GetAccountBlocksCode, id, m); err != nil {
				f.log.Error(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s error: %v", start, count, p.Address(), err))
			} else {
				f.log.Info(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s", start, count, p.Address()))
			}
			monitor.LogEvent("net/fetch", "GetAccountBlocks_Send")
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
	}
}
