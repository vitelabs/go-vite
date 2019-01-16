package net

import (
	"encoding/hex"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/seiflotfy/cuckoofilter"

	"github.com/vitelabs/go-vite/p2p/discovery"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

const cacheSBlockTotal = 1000
const cacheABlockTotal = 10000

var currentHeight uint64

// receive blocks and record them, construct skeleton to filter subsequent fetch
type receiver struct {
	ready       int32 // atomic, can report newBlock to pool
	newSBlocks  []*ledger.SnapshotBlock
	newABlocks  []*ledger.AccountBlock
	sFeed       *snapshotBlockFeed
	aFeed       *accountBlockFeed
	verifier    Verifier
	broadcaster Broadcaster
	filter      Filter
	log         log15.Logger
	batchSource types.BlockSource // report to pool
	p2p         p2p.Server

	mu          sync.RWMutex
	KnownBlocks *cuckoofilter.CuckooFilter
}

func newReceiver(verifier Verifier, broadcaster Broadcaster, filter Filter, p2p p2p.Server) *receiver {
	return &receiver{
		newSBlocks:  make([]*ledger.SnapshotBlock, 0, cacheSBlockTotal),
		newABlocks:  make([]*ledger.AccountBlock, 0, cacheABlockTotal),
		sFeed:       newSnapshotBlockFeed(),
		aFeed:       newAccountBlockFeed(),
		verifier:    verifier,
		broadcaster: broadcaster,
		filter:      filter,
		log:         log15.New("module", "net/receiver"),
		batchSource: types.RemoteSync,
		p2p:         p2p,
		KnownBlocks: cuckoofilter.NewCuckooFilter(filterCap),
	}
}

func (s *receiver) ID() string {
	return "receiver"
}

func (s *receiver) Cmds() []ViteCmd {
	return []ViteCmd{NewSnapshotBlockCode, NewAccountBlockCode, SnapshotBlocksCode, AccountBlocksCode}
}

func (s *receiver) Handle(msg *p2p.Msg, sender Peer) (err error) {
	switch ViteCmd(msg.Cmd) {
	case NewSnapshotBlockCode:
		block := new(ledger.SnapshotBlock)
		if err = block.Deserialize(msg.Payload); err != nil {
			return err
		}

		s.log.Info(fmt.Sprintf("receive new snapshotblock %s/%d from %s", block.Hash, block.Height, sender.RemoteAddr()))

		sender.SeeBlock(block.Hash)

		return s.ReceiveNewSnapshotBlock(block, sender)

	case NewAccountBlockCode:
		block := new(ledger.AccountBlock)
		if err = block.Deserialize(msg.Payload); err != nil {
			return err
		}

		s.log.Info(fmt.Sprintf("receive new accountblock %s from %s", block.Hash, sender.RemoteAddr()))

		sender.SeeBlock(block.Hash)

		return s.ReceiveNewAccountBlock(block, sender)

	case SnapshotBlocksCode:
		bs := new(message.SnapshotBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			return err
		}

		return s.ReceiveSnapshotBlocks(bs.Blocks, sender)

	case AccountBlocksCode:
		bs := new(message.AccountBlocks)
		if err = bs.Deserialize(msg.Payload); err != nil {
			return err
		}

		return s.ReceiveAccountBlocks(bs.Blocks, sender)
	}

	return nil
}

func (s *receiver) mark(hash types.Hash) {
	s.filter.done(hash)
}

func (s *receiver) block(peer Peer, reason p2p.DiscReason) {
	peer.Disconnect(reason)

	var id discovery.NodeID
	buf, err := hex.DecodeString(peer.ID())
	if err != nil {
		id = discovery.ZERO_NODE_ID
	} else {
		copy(id[:], buf)
	}

	s.p2p.Block(id, peer.RemoteAddr().IP, reason)
}

func (s *receiver) ReceiveNewSnapshotBlock(block *ledger.SnapshotBlock, sender Peer) (err error) {
	defer monitor.LogTime("net/receive", "NewSnapshotBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "NewSnapshotBlock_Event")

	s.mu.Lock()
	seen := s.KnownBlocks.Lookup(block.Hash[:])
	s.mu.Unlock()

	if seen {
		s.log.Debug(fmt.Sprintf("has NewSnapshotBlock %s/%d", block.Hash, block.Height))
		return
	}

	// record
	hash := block.ComputeHash()
	s.mu.Lock()
	exist := s.KnownBlocks.InsertUnique(hash[:])
	s.mu.Unlock()

	if exist {
		return
	}

	if s.verifier != nil {
		if err = s.verifier.VerifyNetSb(block); err != nil {
			s.log.Error(fmt.Sprintf("verify NewSnapshotBlock %s/%d from %s fail: %v", block.Hash, block.Height, sender.RemoteAddr(), err))
			s.block(sender, p2p.DiscProtocolError)
			return err
		}
	}

	// broadcast
	s.broadcaster.BroadcastSnapshotBlock(block)

	if s.ready == 0 {
		if len(s.newSBlocks) >= cacheSBlockTotal {
			return
		}

		s.newSBlocks = append(s.newSBlocks, block)
		s.log.Debug(fmt.Sprintf("not ready, store NewSnapshotBlock %s/%d, total %d", block.Hash, block.Height, len(s.newSBlocks)))
	} else {
		// record
		s.sFeed.Notify(block, types.RemoteBroadcast)
	}

	return nil
}

func (s *receiver) ReceiveNewAccountBlock(block *ledger.AccountBlock, sender Peer) (err error) {
	defer monitor.LogTime("net/receive", "NewAccountBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "NewAccountBlock_Event")

	s.mu.Lock()
	seen := s.KnownBlocks.Lookup(block.Hash[:])
	s.mu.Unlock()

	if seen {
		s.log.Debug(fmt.Sprintf("has NewAccountBlock %s/%d", block.Hash, block.Height))
		return
	}

	// record
	hash := block.ComputeHash()
	s.mu.Lock()
	exist := s.KnownBlocks.InsertUnique(hash[:])
	s.mu.Unlock()

	if exist {
		return
	}

	if s.verifier != nil {
		if err = s.verifier.VerifyNetAb(block); err != nil {
			s.log.Error(fmt.Sprintf("verify NewAccountBlock %s/%d from %s fail: %v", block.Hash, block.Height, sender.RemoteAddr(), err))
			s.block(sender, p2p.DiscProtocolError)
			return
		}
	}

	s.broadcaster.BroadcastAccountBlock(block)

	if s.ready == 0 {
		if len(s.newABlocks) >= cacheABlockTotal {
			return
		}

		s.newABlocks = append(s.newABlocks, block)
		s.log.Warn(fmt.Sprintf("not ready, store NewAccountBlock %s, total %d", block.Hash, len(s.newABlocks)))
	} else {
		s.aFeed.Notify(block, types.RemoteBroadcast)
	}

	return nil
}

func (s *receiver) ReceiveSnapshotBlock(block *ledger.SnapshotBlock, sender Peer) (err error) {
	if block == nil {
		return
	}

	defer monitor.LogTime("net/receive", "SnapshotBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "SnapshotBlock_Event")

	if s.filter.has(block.Hash) {
		s.log.Debug(fmt.Sprintf("has SnapshotBlock %s/%d", block.Hash, block.Height))
		return
	}

	if s.verifier != nil {
		if err = s.verifier.VerifyNetSb(block); err != nil {
			s.log.Error(fmt.Sprintf("verify SnapshotBlock %s/%d from %s fail: %v", block.Hash, block.Height, sender.RemoteAddr(), err))
			s.block(sender, p2p.DiscProtocolError)
			return err
		}
	}

	s.mark(block.Hash)

	s.sFeed.Notify(block, s.batchSource)

	return nil
}

func (s *receiver) ReceiveAccountBlock(block *ledger.AccountBlock, sender Peer) (err error) {
	if block == nil {
		return
	}

	defer monitor.LogTime("net/receive", "AccountBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "AccountBlock_Event")

	if s.filter.has(block.Hash) {
		s.log.Debug(fmt.Sprintf("has AccountBlock %s", block.Hash))
		return
	}

	if s.verifier != nil {
		if err = s.verifier.VerifyNetAb(block); err != nil {
			s.log.Error(fmt.Sprintf("verify AccountBlock %s/%d from %s fail: %v", block.Hash, block.Height, sender.RemoteAddr(), err))
			s.block(sender, p2p.DiscProtocolError)
			return
		}
	}

	s.mark(block.Hash)

	s.aFeed.Notify(block, s.batchSource)

	return nil
}

func (s *receiver) ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock, sender Peer) (err error) {
	for _, block := range blocks {
		if err = s.ReceiveSnapshotBlock(block, sender); err != nil {
			return
		}
	}

	return
}

func (s *receiver) ReceiveAccountBlocks(blocks []*ledger.AccountBlock, sender Peer) (err error) {
	for _, block := range blocks {
		if err = s.ReceiveAccountBlock(block, sender); err != nil {
			return
		}
	}

	return
}

func (s *receiver) listen(st SyncState) {
	if st == Syncing {
		s.log.Warn(fmt.Sprintf("silence: %s", st))
		atomic.StoreInt32(&s.ready, 0)
		return
	}

	if atomic.LoadInt32(&s.ready) == 1 {
		return
	}

	if st == Syncdone || st == SyncDownloaded || st == Syncerr {
		s.log.Info(fmt.Sprintf("ready: %s", st))
		atomic.StoreInt32(&s.ready, 1)

		// new blocks
		for _, block := range s.newSBlocks {
			s.sFeed.Notify(block, types.RemoteBroadcast)
		}

		for _, block := range s.newABlocks {
			s.aFeed.Notify(block, types.RemoteBroadcast)
		}

		s.newSBlocks = s.newSBlocks[:0]
		s.newABlocks = s.newABlocks[:0]

		// after synced, blocks will be transmit by fetch
		s.batchSource = types.RemoteFetch
	}
}

func (s *receiver) SubscribeAccountBlock(fn AccountblockCallback) (subId int) {
	return s.aFeed.Sub(fn)
}

func (s *receiver) UnsubscribeAccountBlock(subId int) {
	s.aFeed.Unsub(subId)
}

func (s *receiver) SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int) {
	return s.sFeed.Sub(fn)
}

func (s *receiver) UnsubscribeSnapshotBlock(subId int) {
	s.sFeed.Unsub(subId)
}
