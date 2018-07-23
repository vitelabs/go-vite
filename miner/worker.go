package miner

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
	"time"
)

// Work is the workers current environment and holds
// all of the current state information
type worker struct {
	types.LifecycleStatus
	workChan *chan time.Time
	chain    SnapshotChainRW
	coinbase types.Address
	mu       sync.Mutex
}

func (self *worker) Init() {
	self.PreInit()
	defer self.PostInit()
}

func (self *worker) Start() {
	self.PreStart()
	defer self.PostStart()
	go self.update()
}

func (self *worker) Stop() {
	self.PreStop()
	defer self.PostStop()
}

func (self *worker) update() {
	for !self.Stopped() {
		// A real event arrived, process interesting content
		select {
		// Handle ChainHeadEvent
		case t := <-*self.workChan:
			println("start working once.")
			self.genAndInsert(t)
		}
	}
}
func (self *worker) genAndInsert(t time.Time) {
	self.mu.Lock()
	defer self.mu.Unlock()
	self.chain.WriteMiningBlock(generateSnapshot(t, self.coinbase))
}
func generateSnapshot(t time.Time, coinbase types.Address) *ledger.SnapshotBlock {
	block := ledger.SnapshotBlock{Producer: &coinbase, Timestamp: uint64(t.Unix())}
	return &block
}
func (self *worker) setWorkCh(newWorkCh *chan time.Time) {
	self.workChan = newWorkCh
}
