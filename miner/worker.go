package miner

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"sync"
	"time"
	"github.com/inconshreveable/log15"
)

var wLog = log15.New("module", "miner/worker")

// worker
type worker struct {
	MinerLifecycle
	workChan <-chan time.Time
	chain    SnapshotChainRW
	coinbase types.Address
	mu       sync.Mutex
	updateWg sync.WaitGroup
	updateCh chan int  // update goroutine closed event chan
}

func (self *worker) Init() {
	self.PreInit()
	defer self.PostInit()
}

func (self *worker) Start() {
	self.PreStart()
	defer self.PostStart()
	self.updateCh = make(chan int)
	go self.update(self.updateCh)
}

func (self *worker) Stop() {
	self.PreStop()
	defer self.PostStop()
	close(self.updateCh)  // close update goroutine
	self.updateWg.Wait()
}

// get workChan and insert snapshot block chain
func (self *worker) update(ch chan int) {
	self.updateWg.Add(1)
	defer self.updateWg.Done()
	for !self.Stopped() {
		// A real event arrived, process interesting content
		select {
		// Handle ChainHeadEvent
		case t, ok := <-self.workChan:
			if !ok {
				wLog.Warn("channel closed.")
				if !self.Stopped() {
					time.Sleep(time.Second)
				}
			} else {
				wLog.Info("start working once.")
				self.genAndInsert(t)
			}
		case <-ch: // closed event chan
			wLog.Info("worker.update closed.")
		 	return
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
func (self *worker) setWorkCh(newWorkCh <-chan time.Time) {
	self.workChan = newWorkCh
}
