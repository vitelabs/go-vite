package producer

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	"github.com/vitelabs/go-vite/v2/ledger/consensus"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/monitor"
)

var wLog = log15.New("module", "miner/worker")

// worker
type worker struct {
	producerLifecycle
	tools     *tools
	coinbase  interfaces.Account
	mu        sync.Mutex
	wg        sync.WaitGroup
	seedCache *lru.Cache
	log       log15.Logger
}

func newWorker(chain *tools, coinbase interfaces.Account) *worker {
	cache, err := lru.New(1000)
	if err != nil {
		panic(err)
	}
	return &worker{tools: chain, coinbase: coinbase, seedCache: cache, log: log15.New("module", "producer/worker")}
}

func (w *worker) Init() error {
	if !w.PreInit() {
		return errors.New("pre init worker fail")
	}
	defer w.PostInit()
	return nil
}

func (w *worker) Start() error {
	if !w.PreStart() {
		return errors.New("pre start fail")
	}
	defer w.PostStart()
	return nil
}

func (w *worker) Stop() error {
	if !w.PreStop() {
		return errors.New("pre stop fail")
	}
	defer w.PostStop()
	w.wg.Wait()
	return nil
}

func (w *worker) produceSnapshot(e consensus.Event) {
	if e.Address != w.coinbase.Address() {
		mLog.Error("coinbase must be equal.", "addr", e.Address.String())
		return
	}
	tmpE := &e
	common.Go(func() {
		w.genAndInsert(tmpE)
	})
}

func (w *worker) genAndInsert(e *consensus.Event) {
	wLog.Info("genAndInsert start.", "event", e)
	defer wLog.Info("genAndInsert end.", "event", e)
	defer monitor.LogTime("producer", "snapshotGenInsert", time.Now())
	w.wg.Add(1)
	defer w.wg.Done()
	w.mu.Lock()
	defer w.mu.Unlock()
	// lock pool
	w.tools.pool.LockInsert()
	// unlock pool
	defer w.tools.pool.UnLockInsert()

	seed := w.randomSeed()

	// generate snapshot block
	b, err := w.tools.generateSnapshot(e, w.coinbase, seed, w.getSeedByHash)
	if err != nil {
		wLog.Error("produce snapshot block fail[generate].", "err", err)
		return
	}

	// insert snapshot block
	err = w.tools.insertSnapshot(b)
	if err != nil {
		wLog.Error("produce snapshot block fail[insert].", "err", err)
		return
	}

	// todo
	w.storeSeedHash(seed, b.SeedHash)
}

func (w *worker) randomSeed() uint64 {
	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	return r.Uint64()
}

func (w *worker) getSeedByHash(hash *types.Hash) uint64 {
	if hash == nil {
		// default-> zero
		return 0
	}
	value, ok := w.seedCache.Get(*hash)
	if ok {
		w.log.Info(fmt.Sprintf("query seed, hash:%s, seed:%d\n", hash, value.(uint64)))
		return value.(uint64)
	} else {
		// default-> zero
		return 0
	}
}
func (w *worker) storeSeedHash(seed uint64, hash *types.Hash) {
	if seed == 0 {
		return
	}
	if hash == nil {
		return
	}
	w.log.Info(fmt.Sprintf("store seed, hash:%s, seed:%d\n", hash, seed))
	w.seedCache.Add(*hash, seed)
}
