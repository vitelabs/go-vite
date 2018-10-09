package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"sync/atomic"
	"time"
)

type Receiver interface {
	ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock)
	ReceiveAccountBlocks(mblocks map[types.Address][]*ledger.AccountBlock)
	ReceiveNewSnapshotBlock(block *ledger.SnapshotBlock)
}

type broadNewSnapshotBlock interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
}

// receive blocks and record them, construct skeleton to filter subsequent fetch
type receiver struct {
	ready       int32 // atomic, can report newBlock to pool
	newBlocks   []*ledger.SnapshotBlock
	sblocks     [][]*ledger.SnapshotBlock
	mblocks     map[types.Address][]*ledger.AccountBlock
	sFeed       *snapshotBlockFeed
	aFeed       *accountBlockFeed
	verifier    Verifier
	broadcaster broadNewSnapshotBlock
	filter      Filter
}

func newReceiver(verifier Verifier, broadcaster broadNewSnapshotBlock, filter Filter) *receiver {
	return &receiver{
		newBlocks:   make([]*ledger.SnapshotBlock, 0, 100),
		sblocks:     make([][]*ledger.SnapshotBlock, 0, 10),
		mblocks:     make(map[types.Address][]*ledger.AccountBlock, 100),
		sFeed:       newSnapshotBlockFeed(),
		aFeed:       newAccountBlockFeed(),
		verifier:    verifier,
		broadcaster: broadcaster,
		filter:      filter,
	}
}

// implementation MsgHandler
func (s *receiver) ID() string {
	return "default new snapshotblocks Handler"
}

func (s *receiver) Cmds() []cmd {
	return []cmd{NewSnapshotBlockCode}
}

func (s *receiver) Handle(msg *p2p.Msg, sender *Peer) error {
	block := new(ledger.SnapshotBlock)
	err := block.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	sender.SeeBlock(block.Hash)

	s.ReceiveNewSnapshotBlock(block)

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
		return
	}

	// record
	s.mark(block.Hash)

	if atomic.LoadInt32(&s.ready) == 0 {
		s.newBlocks = append(s.newBlocks, block)
	} else {
		s.sFeed.Notify(block)
		s.broadcaster.BroadcastSnapshotBlock(block)
	}

	monitor.LogDuration("net/receiver", "nb", time.Now().Sub(t).Nanoseconds())
}

func (s *receiver) ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	t := time.Now()

	if atomic.LoadInt32(&s.ready) == 0 {
		s.sblocks = append(s.sblocks, blocks)

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

func (s *receiver) ReceiveAccountBlocks(mblocks map[types.Address][]*ledger.AccountBlock) {
	t := time.Now()

	if atomic.LoadInt32(&s.ready) == 0 {
		for addr, blocks := range mblocks {
			s.mblocks[addr] = append(s.mblocks[addr], blocks...)

			for _, block := range blocks {
				s.mark(block.Hash)
			}
		}
	} else {
		for _, blocks := range mblocks {
			for _, block := range blocks {
				// verify
				if s.verifier.VerifyforP2P(block) {
					s.aFeed.Notify(block)
					s.mark(block.Hash)
				}
			}
		}
	}

	monitor.LogDuration("net/receiver", "abs", time.Now().Sub(t).Nanoseconds())
}

func (s *receiver) listen(st SyncState) {
	if st == Syncdone || st == SyncDownloaded {
		// caution: s.blocks and s.mblocks is mutating concurrently
		// so we keep waterMark, after ready, handle rest blocks
		sblockMark := len(s.sblocks)
		mblockMark := make(map[types.Address]int)
		for addr, ablocks := range s.mblocks {
			mblockMark[addr] = len(ablocks)
		}

		for i := 0; i < sblockMark; i++ {
			sblocks := s.sblocks[i]
			for _, block := range sblocks {
				s.sFeed.Notify(block)
			}
		}

		for addr, length := range mblockMark {
			for i := 0; i < length; i++ {
				block := s.mblocks[addr][i]
				s.aFeed.Notify(block)
			}
		}

		atomic.StoreInt32(&s.ready, 1)

		// rest blocks
		for i := sblockMark; i < len(s.sblocks); i++ {
			sblocks := s.sblocks[i]
			for _, block := range sblocks {
				s.sFeed.Notify(block)
			}
		}

		for addr, length := range mblockMark {
			for i := length; i < len(s.mblocks[addr]); i++ {
				block := s.mblocks[addr][i]
				s.aFeed.Notify(block)
			}
		}

		// new blocks
		for _, block := range s.newBlocks {
			s.sFeed.Notify(block)
		}

		// clear job
		s.sblocks = s.sblocks[:0]
		s.mblocks = nil
	}
}
