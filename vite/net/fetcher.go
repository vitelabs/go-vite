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

// fetchTargets implementation can choose suitable peers to fetch blocks
type fetchTargets interface {
	account(height uint64) Peer
	snapshot(height uint64) Peer
}

type fp struct {
	peers *peerSet
}

// a random taller peer
func (p *fp) account(height uint64) Peer {
	var ps peers
	if height == 0 {
		ps = p.peers.sortPeers()
	} else {
		ps = p.peers.pick(height)
	}

	if len(ps) == 0 {
		return p.peers.bestPeer()
	}

	top := len(ps) / 3
	if top == 0 {
		return ps[0]
	}

	ran := rand.Intn(top)
	return ps[ran]
}

func (p *fp) snapshot(height uint64) Peer {
	if height == 0 {
		return p.peers.bestPeer()
	}

	ps := p.peers.pick(height)
	if len(ps) == 0 {
		return p.peers.bestPeer()
	}

	ran := rand.Intn(len(ps))
	return ps[ran]
}

// fetch filter
const maxMark = 3       // times
const timeThreshold = 3 // second
const expiration = 60

type record struct {
	id     p2p.MsgId
	addAt  int64
	doneAt int64
	mark   int
	st     reqState
}

func (r *record) inc() {
	r.mark += 1
}

func (r *record) reset() {
	r.mark = 0
	r.st = reqPending
	r.addAt = time.Now().Unix()
}

func (r *record) done() {
	r.st = reqDone
	r.doneAt = time.Now().Unix()
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
				return &record{}
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
func (f *filter) hold(hash types.Hash) (id p2p.MsgId, hold bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().Unix()

	var r *record
	var ok bool

	if r, ok = f.records[hash]; ok {
		if r.st == reqError {
			r.reset()
			return r.id, false
		}

		if r.st == reqDone {
			if r.mark >= maxMark && (now-r.doneAt) >= timeThreshold {
				r.reset()
				return r.id, false
			}
		} else {
			if r.mark >= maxMark*2 && (now-r.addAt) >= timeThreshold*2 {
				r.reset()
				return r.id, false
			}
		}

		r.inc()
		return r.id, true
	}

	r = f.pool.Get().(*record)
	r.addAt = now
	r.mark = 0
	r.id = f.idGen.MsgID()
	r.st = reqPending

	f.records[hash] = r
	f.idToHash[r.id] = hash

	return r.id, false
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

func (f *filter) fail(id p2p.MsgId) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if hash, ok := f.idToHash[id]; ok {
		var r *record
		if r, ok = f.records[hash]; ok {
			r.st = reqError
		}
	}
}

type fetcher struct {
	filter *filter

	st       SyncState
	receiver blockReceiver

	policy fetchTargets

	log log15.Logger

	term chan struct{}
}

func newFetcher(peers *peerSet, receiver blockReceiver) *fetcher {
	return &fetcher{
		filter:   newFilter(),
		policy:   &fp{peers},
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

func (f *fetcher) codes() []code {
	return []code{SnapshotBlocksCode, AccountBlocksCode, ExceptionCode}
}

func (f *fetcher) handle(msg p2p.Msg, sender Peer) (err error) {
	switch code(msg.Code) {
	case SnapshotBlocksCode:
		bs := new(message.SnapshotBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			return err
		}

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

	case AccountBlocksCode:
		bs := new(message.AccountBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			return err
		}

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

	case ExceptionCode:
		f.filter.fail(msg.Id)
	}

	return nil
}

func (f *fetcher) FetchSnapshotBlocks(start types.Hash, count uint64) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	// been suppressed
	id, hold := f.filter.hold(start)
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetSnapshotBlocks[hash %s, count %d]", start, count))
		return
	}

	if p := f.policy.snapshot(0); p != nil {
		m := &message.GetSnapshotBlocks{
			From:    ledger.HashHeight{Hash: start},
			Count:   count,
			Forward: false,
		}

		if err := p.send(GetSnapshotBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetSnapshotBlocks[hash %s, count %d] to %s: %v", start, count, p, err))
			f.filter.fail(id)
		} else {
			f.log.Info(fmt.Sprintf("send GetSnapshotBlocks[hash %s, count %d] to %s", start, count, p))
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
		f.filter.fail(id)
	}
}

// FetchSnapshotBlocksWithHeight fetch blocks:
//  ... count blocks ... {hash, height}
func (f *fetcher) FetchSnapshotBlocksWithHeight(hash types.Hash, height uint64, count uint64) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	id, hold := f.filter.hold(hash)
	// been suppressed
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetSnapshotBlocks[hash %s, count %d]", hash, count))
		return
	}

	if p := f.policy.snapshot(0); p != nil {
		m := &message.GetSnapshotBlocks{
			From:    ledger.HashHeight{Hash: hash},
			Count:   count,
			Forward: false,
		}

		if err := p.send(GetSnapshotBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetSnapshotBlocks[hash %s, count %d] to %s: %v", hash, count, p, err))
			f.filter.fail(id)
		} else {
			f.log.Info(fmt.Sprintf("send GetSnapshotBlocks[hash %s, count %d] to %s", hash, count, p))
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
		f.filter.fail(id)
	}
}

func (f *fetcher) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	id, hold := f.filter.hold(start)
	// been suppressed
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetAccountBlocks[hash %s, count %d]", start, count))
		return
	}

	if p := f.policy.account(0); p != nil {
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

		if err := p.send(GetAccountBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetAccountBlocks[hash %s, count %d] to %s: %v", start, count, p, err))
			f.filter.fail(id)
		} else {
			f.log.Info(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s", start, count, p))
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
		f.filter.fail(id)
	}
}

func (f *fetcher) FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64) {
	if !f.st.syncExited() {
		f.log.Debug("in syncing flow, cannot fetch")
		return
	}

	id, hold := f.filter.hold(start)
	// been suppressed
	if hold {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetAccountBlocks[hash %s, count %d]", start, count))
		return
	}

	if p := f.policy.account(sHeight); p != nil {
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

		if err := p.send(GetAccountBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("failed to send GetAccountBlocks[hash %s, count %d] to %s: %v", start, count, p, err))
			f.filter.fail(id)
		} else {
			f.log.Info(fmt.Sprintf("send GetAccountBlocks[hash %s, count %d] to %s", start, count, p))
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
		f.filter.fail(id)
	}
}
