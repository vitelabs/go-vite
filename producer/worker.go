package producer

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/log15"
)

var wLog = log15.New("module", "miner/worker")

// worker
type worker struct {
	producerLifecycle
	tools    *tools
	coinbase types.Address
	mu       sync.Mutex
	wg       sync.WaitGroup
}

func newWorker(chain *tools, coinbase types.Address) *worker {
	return &worker{tools: chain, coinbase: coinbase}
}

func (self *worker) Init() error {
	if !self.PreInit() {
		return errors.New("pre init worker fail.")
	}
	defer self.PostInit()
	return nil
}

func (self *worker) Start() error {
	if !self.PreStart() {
		return errors.New("pre start fail.")
	}
	defer self.PostStart()
	return nil
}

func (self *worker) Stop() error {
	if !self.PreStop() {
		return errors.New("pre stop fail")
	}
	defer self.PostStop()
	self.wg.Wait()
	return nil
}

func (self *worker) produceSnapshot(e consensus.Event) {
	self.wg.Add(1)
	if !self.tools.checkAddressLock(e.Address) {
		mLog.Error("coinbase must be unlock.", "addr", e.Address.String())
		return
	}
	go self.genAndInsert(&e)
}

func (self *worker) genAndInsert(e *consensus.Event) {
	defer self.wg.Done()
	self.mu.Lock()
	defer self.mu.Unlock()
	// lock pool
	self.tools.ledgerLock()
	// unlock pool
	defer self.tools.ledgerUnLock()

	// generate snapshot block
	b := self.tools.generateSnapshot(e)

	// insert snapshot block
	self.tools.insertSnapshot(b)
}
