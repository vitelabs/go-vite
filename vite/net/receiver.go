package net

import (
	"github.com/seiflotfy/cuckoofilter"
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

type receiver struct {
	ready       int32 // atomic, can report newBlock to pool
	newBlocks   []*ledger.SnapshotBlock
	sblocks     []*ledger.SnapshotBlock
	mblocks     map[types.Address][]*ledger.AccountBlock
	sFeed       *snapshotBlockFeed
	aFeed       *accountBlockFeed
	broadcaster broadNewSnapshotBlock
	record      *cuckoofilter.CuckooFilter
}

func newReceiver(broadcaster broadNewSnapshotBlock) *receiver {
	return &receiver{
		ready:       0,
		newBlocks:   make([]*ledger.SnapshotBlock, 0, 100),
		sblocks:     make([]*ledger.SnapshotBlock, 0, 1000),
		mblocks:     make(map[types.Address][]*ledger.AccountBlock, 100),
		sFeed:       newSnapshotBlockFeed(),
		aFeed:       newAccountBlockFeed(),
		broadcaster: broadcaster,
		record:      cuckoofilter.NewCuckooFilter(10000),
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

// implementation Receiver
func (s *receiver) ReceiveNewSnapshotBlock(block *ledger.SnapshotBlock) {
	t := time.Now()

	if s.record.Lookup(block.Hash[:]) {
		monitor.LogDuration("net/receiver", "nb2", time.Now().Sub(t).Nanoseconds())
		return
	}

	s.record.Insert(block.Hash[:])

	if atomic.LoadInt32(&s.ready) == 0 {
		s.newBlocks = append(s.newBlocks, block)
	} else {
		s.sFeed.Notify(block)
		s.broadcaster.BroadcastSnapshotBlock(block)
	}

	monitor.LogDuration("net/receiver", "nb", time.Now().Sub(t).Nanoseconds())
}

// todo add record
func (s *receiver) ReceiveSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	t := time.Now()

	if atomic.LoadInt32(&s.ready) == 0 {
		s.newBlocks = append(s.newBlocks, blocks...)
	} else {
		for _, block := range blocks {
			s.sFeed.Notify(block)
		}
	}

	monitor.LogDuration("net/receiver", "bs", time.Now().Sub(t).Nanoseconds())
}

// todo add record
func (s *receiver) ReceiveAccountBlocks(mblocks map[types.Address][]*ledger.AccountBlock) {
	t := time.Now()

	if atomic.LoadInt32(&s.ready) == 0 {
		for addr, blocks := range mblocks {
			s.mblocks[addr] = append(s.mblocks[addr], blocks...)
		}
	} else {
		for _, blocks := range mblocks {
			for _, block := range blocks {
				s.aFeed.Notify(block)
			}
		}
	}

	monitor.LogDuration("net/receiver", "abs", time.Now().Sub(t).Nanoseconds())
}

func (s *receiver) listen(st SyncState) {
	if st == Syncdone || st == SyncDownloaded {
		for _, block := range s.sblocks {
			s.sFeed.Notify(block)
		}
		s.sblocks = s.sblocks[:0]

		for addr, blocks := range s.mblocks {
			for _, block := range blocks {
				s.aFeed.Notify(block)
			}

			s.mblocks[addr] = s.mblocks[addr][:0]
		}

		atomic.StoreInt32(&s.ready, 1)
	}
}
