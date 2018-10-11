package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"sync/atomic"
	"time"
)

type Receiver interface {
	ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock)
	ReceiveAccountBlocks(blocks []*ledger.AccountBlock)

	ReceiveNewSnapshotBlock(block *ledger.SnapshotBlock)
	ReceiveNewAccountBlock(block *ledger.AccountBlock)
}

// receive blocks and record them, construct skeleton to filter subsequent fetch
type receiver struct {
	ready       int32 // atomic, can report newBlock to pool
	newSBlocks  []*ledger.SnapshotBlock
	newABlocks  []*ledger.AccountBlock
	sblocks     [][]*ledger.SnapshotBlock
	ablocks     [][]*ledger.AccountBlock
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
		sblocks:     make([][]*ledger.SnapshotBlock, 0, 10),
		ablocks:     make([][]*ledger.AccountBlock, 0, 10),
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
	return "default new snapshotblocks Handler"
}

func (s *receiver) Cmds() []cmd {
	return []cmd{NewSnapshotBlockCode, NewAccountBlockCode}
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
	case NewAccountBlockCode:
		block := new(ledger.AccountBlock)
		err := block.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		sender.SeeBlock(block.Hash)

		s.ReceiveNewAccountBlock(block)
	}

	return nil
}

func (s *receiver) mark(hash types.Hash) {
	s.filter.done(hash)
}

// implementation Receiver
func (s *receiver) ReceiveNewSnapshotBlock(block *ledger.SnapshotBlock) {
	t := time.Now()

	if s.filter.has(block.Hash) {
		monitor.LogDuration("net/receiver", "nb2", time.Now().Sub(t).Nanoseconds())
		s.log.Warn(fmt.Sprintf("has receive the same new block %s", block.Hash))
		return
	}

	// record
	s.mark(block.Hash)

	if atomic.LoadInt32(&s.ready) == 0 {
		s.newSBlocks = append(s.newSBlocks, block)
		s.log.Warn(fmt.Sprintf("not ready, store new snapshotblock %s, total %d", block.Hash, len(s.newSBlocks)))
	} else {
		s.sFeed.Notify(block)
	}

	s.broadcaster.BroadcastSnapshotBlock(block)

	monitor.LogDuration("net/receiver", "nb", time.Now().Sub(t).Nanoseconds())
}

func (s *receiver) ReceiveNewAccountBlock(block *ledger.AccountBlock) {
	t := time.Now()

	if s.filter.has(block.Hash) {
		monitor.LogDuration("net/receiver", "nb2", time.Now().Sub(t).Nanoseconds())
		s.log.Warn(fmt.Sprintf("has receive the same new block %s", block.Hash))
		return
	}

	// record
	s.mark(block.Hash)

	if atomic.LoadInt32(&s.ready) == 0 {
		s.newABlocks = append(s.newABlocks, block)
		s.log.Warn(fmt.Sprintf("not ready, store new accountblock %s, total %d", block.Hash, len(s.newABlocks)))
	} else {
		s.aFeed.Notify(block)
	}

	s.broadcaster.BroadcastAccountBlock(block)

	monitor.LogDuration("net/receiver", "nb", time.Now().Sub(t).Nanoseconds())
}

func (s *receiver) ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	t := time.Now()

	if atomic.LoadInt32(&s.ready) == 0 {
		s.sblocks = append(s.sblocks, blocks)
		s.log.Warn(fmt.Sprintf("not ready, store %d snapshotblocks", len(s.sblocks)))

		for _, block := range blocks {
			s.mark(block.Hash)
		}
	} else {
		for _, block := range blocks {
			s.mark(block.Hash)
			s.sFeed.Notify(block)
		}
	}

	monitor.LogDuration("net/receiver", "bs", time.Now().Sub(t).Nanoseconds())
}

func (s *receiver) ReceiveAccountBlocks(ablocks []*ledger.AccountBlock) {
	t := time.Now()

	if atomic.LoadInt32(&s.ready) == 0 {
		s.ablocks = append(s.ablocks, ablocks)

		for _, block := range ablocks {
			s.mark(block.Hash)
		}

		s.log.Warn(fmt.Sprintf("not ready, store %d accountblocks", len(ablocks)))
	} else {
		for _, block := range ablocks {
			// verify
			if s.verifier.VerifyforP2P(block) {
				s.aFeed.Notify(block)
				s.mark(block.Hash)
			}
		}
	}

	monitor.LogDuration("net/receiver", "abs", time.Now().Sub(t).Nanoseconds())
}

func (s *receiver) listen(st SyncState) {
	if atomic.LoadInt32(&s.ready) == 1 {
		return
	}

	if st == Syncdone || st == SyncDownloaded {
		s.log.Info(fmt.Sprintf("sync status: %s", st))

		// caution: s.blocks and s.mblocks is mutating concurrently
		// so we keep waterMark, after ready, handle rest blocks
		sblockMark := len(s.sblocks)
		ablockMark := len(s.ablocks)

		// use for log
		var sblockCount, ablockCount uint64

		for i := 0; i < sblockMark; i++ {
			sblocks := s.sblocks[i]
			for _, block := range sblocks {
				s.sFeed.Notify(block)

				sblockCount++
			}
		}

		for i := 0; i < ablockMark; i++ {
			ablocks := s.ablocks[i]
			for _, block := range ablocks {
				s.aFeed.Notify(block)

				ablockCount++
			}
		}

		atomic.StoreInt32(&s.ready, 1)

		// rest blocks
		for i := sblockMark; i < len(s.sblocks); i++ {
			for _, block := range s.sblocks[i] {
				s.sFeed.Notify(block)

				sblockCount++
			}
		}

		for i := ablockMark; i < len(s.ablocks); i++ {
			for _, block := range s.ablocks[i] {
				s.aFeed.Notify(block)

				ablockCount++
			}
		}

		// new blocks
		for _, block := range s.newSBlocks {
			s.sFeed.Notify(block)
		}

		for _, block := range s.newABlocks {
			s.aFeed.Notify(block)
		}

		s.log.Info(fmt.Sprintf("notify %d sblocks, %d ablocks, %d newSBlocks, %d newABlocks", sblockCount, ablockCount, len(s.newSBlocks), len(s.newABlocks)))

		// clear job
		s.sblocks = s.sblocks[:0]
		s.ablocks = s.ablocks[:0]
		s.newSBlocks = s.newSBlocks[:0]
		s.newABlocks = s.newABlocks[:0]
	}
}
