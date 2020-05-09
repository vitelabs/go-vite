package miner

import (
	"sync"
	"time"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
)

// worker
type worker struct {
	MinerLifecycle
	workChan <-chan time.Time
	chain    SnapshotChainRW
	coinbase common.Address
	mu       sync.Mutex
	updateWg sync.WaitGroup
	updateCh chan int // update goroutine closed event chan
}

func (w *worker) Init() {
	w.PreInit()
	defer w.PostInit()
}

func (w *worker) Start() {
	w.PreStart()
	defer w.PostStart()
	w.updateCh = make(chan int)
	go w.update(w.updateCh)
}

func (w *worker) Stop() {
	w.PreStop()
	defer w.PostStop()
	close(w.updateCh) // close update goroutine
	w.updateWg.Wait()
}

// get workChan and insert snapshot block chain
func (w *worker) update(ch chan int) {
	w.updateWg.Add(1)
	defer w.updateWg.Done()
	for !w.Stopped() {
		// A real event arrived, process interesting content
		select {
		// Handle ChainHeadEvent
		case t, ok := <-w.workChan:
			if !ok {
				log.Warn("channel closed.")
				if !w.Stopped() {
					time.Sleep(time.Second)
				}
			} else {
				log.Info("start working once.")
				w.genAndInsert(t)
			}
		case <-ch: // closed event chan
			log.Info("worker.update closed.")
			return
		}
	}
}

func (w *worker) genAndInsert(t time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.chain.MiningSnapshotBlock(w.coinbase, t.Unix())
}

func (w *worker) setWorkCh(newWorkCh <-chan time.Time) {
	w.workChan = newWorkCh
}
