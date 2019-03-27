package producer

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/hashicorp/golang-lru"

	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
)

var wLog = log15.New("module", "miner/worker")

// worker
type worker struct {
	producerLifecycle
	tools     *tools
	coinbase  *AddressContext
	mu        sync.Mutex
	wg        sync.WaitGroup
	seedCache *lru.Cache
}

func newWorker(chain *tools, coinbase *AddressContext) *worker {
	cache, err := lru.New(1000)
	if err != nil {
		panic(err)
	}
	return &worker{tools: chain, coinbase: coinbase, seedCache: cache}
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
	err := self.tools.checkAddressLock(e.Address, self.coinbase)
	if err != nil {
		mLog.Error("coinbase must be unlock.", "addr", e.Address.String(), "err", err)
		return
	}
	tmpE := &e
	common.Go(func() {
		self.genAndInsert(tmpE)
	})
}

func (self *worker) genAndInsert(e *consensus.Event) {
	wLog.Info("genAndInsert start.", "event", e)
	defer wLog.Info("genAndInsert end.", "event", e)
	defer monitor.LogTime("producer", "snapshotGenInsert", time.Now())
	defer self.wg.Done()
	self.mu.Lock()
	defer self.mu.Unlock()
	// lock pool
	self.tools.ledgerLock()
	// unlock pool
	defer self.tools.ledgerUnLock()

	seed := self.randomSeed()

	// generate snapshot block
	b, err := self.tools.generateSnapshot(e, self.coinbase, seed, self.getSeedByHash)
	if err != nil {
		wLog.Error("produce snapshot block fail[generate].", "err", err)
		return
	}

	// insert snapshot block
	err = self.tools.insertSnapshot(b)
	if err != nil {
		wLog.Error("produce snapshot block fail[insert].", "err", err)
		return
	}

	// todo
	self.storeSeedHash(seed, b.SeedHash)
}

func (self *worker) randomSeed() uint64 {
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	return r.Uint64()
}

func (self *worker) getSeedByHash(hash *types.Hash) uint64 {
	if hash == nil {
		// default-> zero
		return 0
	}
	fmt.Printf("query seed, hash:%s\n", hash)
	value, ok := self.seedCache.Get(*hash)
	if ok {
		fmt.Printf("query seed, hash:%s, seed:%d\n", hash, value.(uint64))
		return value.(uint64)
	} else {
		// default-> zero
		return 0
	}
}
func (self *worker) storeSeedHash(seed uint64, hash *types.Hash) {
	if seed == 0 {
		return
	}
	if hash == nil {
		return
	}
	fmt.Printf("store seed, hash:%s, seed:%d\n", hash, seed)
	self.seedCache.Add(*hash, seed)
}
