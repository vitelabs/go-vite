package net

import (
	"errors"
	"fmt"
	"math/rand"
	"sync/atomic"

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

type fetchPolicy struct {
	peers *peerSet
}

func (p *fetchPolicy) pickAccount(height uint64) []Peer {
	var l, taller []Peer

	peers := p.peers.Peers()
	total := len(peers)

	if total == 0 {
		return l
	}

	// best
	var peer Peer
	var maxHeight uint64
	for _, p := range peers {
		peerHeight := p.Height()

		if peerHeight > maxHeight {
			maxHeight = peerHeight
			peer = p
		}

		if peerHeight >= height {
			taller = append(taller, p)
		}
	}

	l = append(l, peer)

	// random
	ran := rand.Intn(total)
	if peer = peers[ran]; peer != l[0] {
		l = append(l, peer)
	}

	// taller
	if len(taller) > 0 {
		ran = rand.Intn(len(taller))
		peer = taller[ran]

		for _, p := range l {
			if peer == p {
				return l
			}
		}

		l = append(l, peer)
	}

	return l
}

func (p *fetchPolicy) pickSnap() Peer {
	return p.peers.BestPeer()
}

type fPolicy interface {
	pickAccount(height uint64) (l []Peer)
	pickSnap() Peer
}

type fetcher struct {
	filter Filter

	st       SyncState
	verifier Verifier
	bn       *blockNotifier

	policy fPolicy
	pool   MsgIder
	ready  int32 // atomic
	log    log15.Logger
}

func newFetcher(filter Filter, peers *peerSet, pool MsgIder) *fetcher {
	return &fetcher{
		filter: filter,
		policy: &fetchPolicy{peers},
		pool:   pool,
		log:    log15.New("module", "net/fetcher"),
	}
}

func (f *fetcher) subSyncState(st SyncState) {
	f.st = st
}

func (f *fetcher) canFetch() bool {
	return f.st != Syncing && f.st != SyncNotStart
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
			f.bn.NotifySnapshotBlock(block, types.RemoteFetch)
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
			f.bn.NotifyAccountBlock(block, types.RemoteFetch)
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
