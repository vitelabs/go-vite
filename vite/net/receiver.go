package net

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

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
}

func newReceiver(verifier Verifier, broadcaster Broadcaster, filter Filter) *receiver {
	return &receiver{
		newSBlocks:  make([]*ledger.SnapshotBlock, 0, 100),
		newABlocks:  make([]*ledger.AccountBlock, 0, 100),
		sFeed:       newSnapshotBlockFeed(),
		aFeed:       newAccountBlockFeed(),
		verifier:    verifier,
		broadcaster: broadcaster,
		filter:      filter,
		log:         log15.New("module", "net/receiver"),
	}
}

// implementation MsgHandler
func (s *receiver) ID() string {
	return "receiver"
}

func (s *receiver) Cmds() []cmd {
	return []cmd{NewSnapshotBlockCode, NewAccountBlockCode, SnapshotBlocksCode, AccountBlocksCode}
}

func (s *receiver) Handle(msg *p2p.Msg, sender *Peer) error {
	switch cmd(msg.Cmd) {
	case NewSnapshotBlockCode:
		block := new(ledger.SnapshotBlock)
		err := block.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		sender.SeeBlock(block.Hash)

		s.ReceiveNewSnapshotBlock(block)

		s.log.Info(fmt.Sprintf("receive new snapshotblock %s/%s/%d", sender.ID, block.Hash, block.Height))
	case NewAccountBlockCode:
		block := new(ledger.AccountBlock)
		err := block.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		sender.SeeBlock(block.Hash)

		s.ReceiveNewAccountBlock(block)

		s.log.Info(fmt.Sprintf("receive new accountblock %s/%s/%d/%s", sender.ID, block.Hash, block.Height, block.AccountAddress))
	case SnapshotBlocksCode:
		bs := new(message.SnapshotBlocks)
		err := bs.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		s.ReceiveSnapshotBlocks(bs.Blocks)

	case AccountBlocksCode:
		bs := new(message.AccountBlocks)
		err := bs.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		s.ReceiveAccountBlocks(bs.Blocks)
	}

	return nil
}

func (s *receiver) mark(hash types.Hash) {
	s.filter.done(hash)
}

// implementation Receiver
func (s *receiver) ReceiveNewSnapshotBlock(block *ledger.SnapshotBlock) {
	if block == nil {
		return
	}

	defer monitor.LogTime("net/receive", "NewSnapshotBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "NewSnapshotBlock_Event")

	if s.filter.has(block.Hash) {
		return
	}

	if s.verifier != nil {
		verify_b := time.Now()
		if err := s.verifier.VerifyNetSb(block); err != nil {
			monitor.LogDuration("net/verifier", "SnapshotBlock", time.Now().Sub(verify_b).Nanoseconds())
			s.log.Error(fmt.Sprintf("verify new snapshotblock %s/%d fail: %v", block.Hash, block.Height, err))
			return
		}
		monitor.LogDuration("net/verifier", "SnapshotBlock", time.Now().Sub(verify_b).Nanoseconds())
	}

	// record
	s.mark(block.Hash)

	if atomic.LoadInt32(&s.ready) == 0 {
		s.newSBlocks = append(s.newSBlocks, block)
		s.log.Warn(fmt.Sprintf("not ready, store new snapshotblock %s, total %d", block.Hash, len(s.newSBlocks)))
	} else {
		notify_b := time.Now()
		s.sFeed.Notify(block)
		monitor.LogDuration("net/notify", "NewSnapshotBlock", time.Now().Sub(notify_b).Nanoseconds())
		s.log.Info(fmt.Sprintf("notify new snapshotblock %s done", block.Hash))
	}

	s.broadcaster.BroadcastSnapshotBlock(block)
}

func (s *receiver) ReceiveNewAccountBlock(block *ledger.AccountBlock) {
	if block == nil {
		return
	}

	defer monitor.LogTime("net/receive", "NewAccountBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "NewAccountBlock_Event")

	if s.filter.has(block.Hash) {
		return
	}

	if s.verifier != nil {
		verify_b := time.Now()
		if err := s.verifier.VerifyNetAb(block); err != nil {
			monitor.LogDuration("net/verifier", "AccountBlock", time.Now().Sub(verify_b).Nanoseconds())
			s.log.Error(fmt.Sprintf("verify new accountblock %s/%d fail: %v", block.Hash, block.Height, err))
			return
		}
		monitor.LogDuration("net/verifier", "AccountBlock", time.Now().Sub(verify_b).Nanoseconds())
	}

	// record
	s.mark(block.Hash)

	if atomic.LoadInt32(&s.ready) == 0 {
		s.newABlocks = append(s.newABlocks, block)
		s.log.Warn(fmt.Sprintf("not ready, store new accountblock %s, total %d", block.Hash, len(s.newABlocks)))
	} else {
		notify_b := time.Now()
		s.aFeed.Notify(block)
		monitor.LogDuration("net/notify", "NewAccountBlock", time.Now().Sub(notify_b).Nanoseconds())
		s.log.Info(fmt.Sprintf("notify new accountblock %s done", block.Hash))
	}

	s.broadcaster.BroadcastAccountBlock(block)
}

func (s *receiver) ReceiveSnapshotBlock(block *ledger.SnapshotBlock) {
	if block == nil {
		return
	}

	defer monitor.LogTime("net/receive", "SnapshotBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "SnapshotBlock_Event")

	if s.filter.has(block.Hash) {
		return
	}

	if s.verifier != nil {
		verify_b := time.Now()
		if err := s.verifier.VerifyNetSb(block); err != nil {
			monitor.LogDuration("net/verifier", "SnapshotBlock", time.Now().Sub(verify_b).Nanoseconds())
			s.log.Error(fmt.Sprintf("verify snapshotblock %s/%d fail: %v", block.Hash, block.Height, err))
			return
		}
		monitor.LogDuration("net/verifier", "SnapshotBlock", time.Now().Sub(verify_b).Nanoseconds())
	}

	s.mark(block.Hash)

	notify_b := time.Now()
	s.sFeed.Notify(block)
	monitor.LogDuration("net/notify", "SnapshotBlock", time.Now().Sub(notify_b).Nanoseconds())
}

func (s *receiver) ReceiveAccountBlock(block *ledger.AccountBlock) {
	if block == nil {
		return
	}

	defer monitor.LogTime("net/receive", "AccountBlock_Time", time.Now())
	monitor.LogEvent("net/receive", "AccountBlock_Event")

	if s.filter.has(block.Hash) {
		return
	}

	if s.verifier != nil {
		verify_b := time.Now()
		if err := s.verifier.VerifyNetAb(block); err != nil {
			monitor.LogDuration("net/verifier", "AccountBlock", time.Now().Sub(verify_b).Nanoseconds())
			s.log.Error(fmt.Sprintf("verify accountblock %s/%d fail: %v", block.Hash, block.Height, err))
			return
		}
		monitor.LogDuration("net/verifier", "AccountBlock", time.Now().Sub(verify_b).Nanoseconds())
	}

	s.mark(block.Hash)

	notify_b := time.Now()
	s.aFeed.Notify(block)
	monitor.LogDuration("net/notify", "AccountBlock", time.Now().Sub(notify_b).Nanoseconds())
}

func (s *receiver) ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	for _, block := range blocks {
		s.ReceiveSnapshotBlock(block)
	}
}

func (s *receiver) ReceiveAccountBlocks(blocks []*ledger.AccountBlock) {
	for _, block := range blocks {
		s.ReceiveAccountBlock(block)
	}
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

	if st == Syncdone || st == SyncDownloaded {
		s.log.Info(fmt.Sprintf("ready: %s", st))
		atomic.StoreInt32(&s.ready, 1)

		// new blocks
		for _, block := range s.newSBlocks {
			s.sFeed.Notify(block)
		}

		for _, block := range s.newABlocks {
			s.aFeed.Notify(block)
		}

		s.newSBlocks = s.newSBlocks[:0]
		s.newABlocks = s.newABlocks[:0]
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
