package protocols

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
)

var errShouldUnsubscribe = errors.New("Should Unsubscribe")

// snapshotblockfeed

type snapshotBlockFeed struct {
	chans   []chan<- *ledger.SnapshotBlock
	newChan chan chan<- *ledger.SnapshotBlock
	delChan chan chan<- *ledger.SnapshotBlock
	event   chan *ledger.SnapshotBlock
	stop    chan struct{}
}

func NewSnapshotBlockFeed() *snapshotBlockFeed {
	f := &snapshotBlockFeed{
		chans:   make([]chan<- *ledger.SnapshotBlock, 0, 1),
		newChan: make(chan chan<- *ledger.SnapshotBlock),
		delChan: make(chan chan<- *ledger.SnapshotBlock),
		event:   make(chan *ledger.SnapshotBlock),
		stop:    make(chan struct{}),
	}

	f.loop()

	return f
}

func (f *snapshotBlockFeed) loop() {
loop:
	for {
		select {
		case <-f.stop:
			for _, c := range f.chans {
				close(c)
			}
			break loop
		case ch := <-f.newChan:
			f.chans = append(f.chans, ch)
		case ch := <-f.delChan:
			for i, c := range f.chans {
				if c == ch {
					f.chans = append(f.chans[:i], f.chans[i+1:]...)
				}
			}
			close(ch)
		case block := <-f.event:
			for _, c := range f.chans {
				c <- block
			}
		}
	}
}

func (f *snapshotBlockFeed) Subscribe() *snapshotBlockSub {
	ch := make(chan *ledger.SnapshotBlock)

	return &snapshotBlockSub{
		feed:    f,
		channel: ch,
	}
}

func (f *snapshotBlockFeed) emit(block *ledger.SnapshotBlock) {
	f.event <- block
}

func (f *snapshotBlockFeed) remove(s *snapshotBlockSub) {
	select {
	case <-f.stop:
	default:
		f.delChan <- s.channel
	}
}

func (f *snapshotBlockFeed) destroy() {
	select {
	case <-f.stop:
	default:
		close(f.stop)
	}
}

type snapshotBlockSub struct {
	feed    *snapshotBlockFeed
	channel chan *ledger.SnapshotBlock
	once    sync.Once
}

func (s *snapshotBlockSub) Chan() <-chan *ledger.SnapshotBlock {
	return s.channel
}

func (s *snapshotBlockSub) Unsubscribe() {
	s.once.Do(func() {
		s.feed.remove(s)
	})
}

// accountblockfeed
type accountBlockFeed struct {
	chans   []chan<- *ledger.AccountBlock
	newChan chan chan<- *ledger.AccountBlock
	delChan chan chan<- *ledger.AccountBlock
	event   chan *ledger.AccountBlock
	stop    chan struct{}
}

func NewAccountBlockFeed() *accountBlockFeed {
	f := &accountBlockFeed{
		chans:   make([]chan<- *ledger.AccountBlock, 0, 1),
		newChan: make(chan chan<- *ledger.AccountBlock),
		delChan: make(chan chan<- *ledger.AccountBlock),
		event:   make(chan *ledger.AccountBlock),
		stop:    make(chan struct{}),
	}

	f.loop()

	return f
}

func (f *accountBlockFeed) loop() {
loop:
	for {
		select {
		case <-f.stop:
			for _, c := range f.chans {
				close(c)
			}
			break loop
		case ch := <-f.newChan:
			f.chans = append(f.chans, ch)
		case ch := <-f.delChan:
			for i, c := range f.chans {
				if c == ch {
					f.chans = append(f.chans[:i], f.chans[i+1:]...)
				}
			}
			close(ch)
		case block := <-f.event:
			for _, c := range f.chans {
				c <- block
			}
		}
	}
}

func (f *accountBlockFeed) Subscribe() *accountBlockSub {
	ch := make(chan *ledger.AccountBlock)

	return &accountBlockSub{
		feed:    f,
		channel: ch,
	}
}

func (f *accountBlockFeed) emit(block *ledger.AccountBlock) {
	f.event <- block
}

func (f *accountBlockFeed) remove(s *accountBlockSub) {
	select {
	case <-f.stop:
	default:
		f.delChan <- s.channel
	}
}

func (f *accountBlockFeed) destroy() {
	select {
	case <-f.stop:
	default:
		close(f.stop)
	}
}

type accountBlockSub struct {
	feed    *accountBlockFeed
	channel chan *ledger.AccountBlock
	once    sync.Once
}

func (s *accountBlockSub) Chan() <-chan *ledger.AccountBlock {
	return s.channel
}

func (s *accountBlockSub) Unsubscribe() {
	s.once.Do(func() {
		s.feed.remove(s)
	})
}
