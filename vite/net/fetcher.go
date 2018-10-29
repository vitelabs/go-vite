package net

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vite/net/message"
	"math/rand"
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
	monitor.LogEvent("net/fetch", "GetSnapshotBlocks")

	// been suppressed
	if f.filter.hold(start) {
		f.log.Debug(fmt.Sprintf("fetch suppressed GetSnapshotBlocks[hash %s, count %d]", start, count))
		return
	}

	if atomic.LoadInt32(&f.ready) == 0 {
		f.log.Debug("not ready")
		return
	}

	if peerList := f.peers.Pick(0); len(peerList) != 0 {
		m := &message.GetSnapshotBlocks{
			From:    ledger.HashHeight{Hash: start},
			Count:   count,
			Forward: false,
		}

		id := f.pool.MsgID()

		p := peerList[rand.Intn(len(peerList))]
		if err := p.Send(GetSnapshotBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("send %s to %s error: %v", m, p, err))
		} else {
			f.log.Debug(fmt.Sprintf("send %s to %s done", m, p))
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

	if atomic.LoadInt32(&f.ready) == 0 {
		f.log.Warn("not ready")
		return
	}

	if peerList := f.peers.Pick(0); len(peerList) != 0 {
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

		p := peerList[rand.Intn(len(peerList))]
		if err := p.Send(GetAccountBlocksCode, id, m); err != nil {
			f.log.Error(fmt.Sprintf("send %s to %s error: %v", m, p, err))
		} else {
			f.log.Debug(fmt.Sprintf("send %s to %s done", m, p))
		}
		monitor.LogEvent("net/fetch", "GetAccountBlocks_Send")

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
