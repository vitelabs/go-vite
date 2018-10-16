package net

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sync/atomic"
)

var errNoSuitablePeer = errors.New("no suitable peer")

type MsgIder interface {
	MsgID() uint64
}

type fetcher struct {
	filter Filter
	peers  *peerSet
	pool   MsgIder
	ready  int32 // atomic
	log    log15.Logger
}

func newFetcher(filter Filter, peers *peerSet, pool MsgIder) *fetcher {
	return &fetcher{
		filter: filter,
		peers:  peers,
		pool:   pool,
		log:    log15.New("module", "net/fetcher"),
	}
}

func (f *fetcher) FetchSnapshotBlocks(start types.Hash, count uint64) {
	monitor.LogEvent("net/fetch", "s")

	// been suppressed
	if f.filter.hold(start) {
		f.log.Warn(fmt.Sprintf("fetch suppressed getSnapshotBlocks: %s %d", start, count))
		return
	}

	if atomic.LoadInt32(&f.ready) == 0 {
		f.log.Warn("not ready")
		return
	}

	m := &message.GetSnapshotBlocks{
		From:    ledger.HashHeight{Hash: start},
		Count:   count,
		Forward: false,
	}

	p := f.peers.BestPeer()
	if p != nil {
		id := f.pool.MsgID()
		err := p.Send(GetSnapshotBlocksCode, id, m)
		if err != nil {
			f.log.Error(fmt.Sprintf("send %s to %s error: %v", GetSnapshotBlocksCode, p, err))
		} else {
			f.log.Info(fmt.Sprintf("send %s to %s done", GetSnapshotBlocksCode, p))
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
	}
}

func (f *fetcher) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	monitor.LogEvent("net/fetch", "a")

	// been suppressed
	if f.filter.hold(start) {
		f.log.Warn(fmt.Sprintf("fetch suppressed getAccountBlocks: %s %d", start, count))
		return
	}

	if atomic.LoadInt32(&f.ready) == 0 {
		f.log.Warn("not ready")
		return
	}

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

	p := f.peers.BestPeer()
	if p != nil {
		id := f.pool.MsgID()
		err := p.Send(GetAccountBlocksCode, id, m)
		if err != nil {
			f.log.Error(fmt.Sprintf("send %s to %s error: %v", GetAccountBlocksCode, p, err))
		} else {
			f.log.Info(fmt.Sprintf("send %s to %s done", GetAccountBlocksCode, p))
		}
	} else {
		f.log.Error(errNoSuitablePeer.Error())
	}
}

func (f *fetcher) listen(st SyncState) {
	if st == Syncdone || st == SyncDownloaded {
		f.log.Info(fmt.Sprintf("ready: %s", st))
		atomic.StoreInt32(&f.ready, 1)
	} else if st == Syncing {
		f.log.Warn(fmt.Sprintf("silence: %s", st))
		atomic.StoreInt32(&f.ready, 0)
	}
}
