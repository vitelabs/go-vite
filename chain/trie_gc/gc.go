package trie_gc

import (
	"github.com/vitelabs/go-vite/log15"
	"math/rand"
	"sync"
	"time"
)

const (
	STATUS_STOPPED              = 1
	STATUS_STARTED              = 2
	STATUS_MARKING_AND_CLEANING = 3
)

type collector struct {
	terminal     chan struct{}
	taskTerminal chan struct{}

	statusLock sync.Mutex
	status     uint8 // 0 is stopped, 1 is started

	wg sync.WaitGroup

	minCheckInterval time.Duration
	maxCheckInterval time.Duration
	ticker           *time.Ticker

	chain Chain

	log log15.Logger

	marker *Marker
}

func NewCollector(chain Chain, ledgerGcRetain uint64) Collector {
	gc := &collector{
		minCheckInterval: time.Hour,
		maxCheckInterval: 3 * time.Hour,

		chain: chain,
		log:   log15.New("module", "trie_gc"),

		marker: NewMarker(chain, ledgerGcRetain),
	}

	return gc
}

func (gc *collector) randomCheckInterval() time.Duration {
	minCheckInterval := int64(gc.minCheckInterval)
	maxCheckInterval := int64(gc.maxCheckInterval)
	return time.Duration(minCheckInterval + rand.Int63n(maxCheckInterval-minCheckInterval))
}

func (gc *collector) Start() {
	gc.statusLock.Lock()
	defer gc.statusLock.Unlock()
	if gc.status >= STATUS_STARTED {
		gc.log.Error("gc is started, don't start again")
		return
	}

	gc.ticker = time.NewTicker(gc.randomCheckInterval())
	gc.taskTerminal = make(chan struct{})
	gc.terminal = make(chan struct{})

	gc.wg.Add(1)
	go func() {
		defer gc.wg.Done()

		gc.runTask()
		for {
			select {
			case <-gc.ticker.C:
				gc.ticker = time.NewTicker(gc.randomCheckInterval())
				gc.runTask()
			case <-gc.terminal:
				return
			}
		}

	}()
	gc.status = STATUS_STARTED
}

func (gc *collector) Stop() {
	gc.statusLock.Lock()
	defer gc.statusLock.Unlock()
	if gc.status < STATUS_STARTED {
		gc.log.Error("gc is stopped, don't stop again")
		return
	}

	gc.ticker.Stop()
	close(gc.taskTerminal)
	close(gc.terminal)

	gc.wg.Wait()
	gc.status = STATUS_STOPPED
}

func (gc *collector) Status() uint8 {
	return gc.status
}

func (gc *collector) runTask() {
	gc.statusLock.Lock()
	if gc.status > STATUS_STARTED {
		gc.log.Error("One task is already running, can't run multiple tasks in parallel.", "method", "runTask")
		gc.statusLock.Unlock()
		return
	}
	gc.status = STATUS_MARKING_AND_CLEANING
	gc.statusLock.Unlock()

	defer func() {
		gc.statusLock.Lock()
		gc.status = STATUS_STARTED
		gc.statusLock.Unlock()
	}()

	if err := gc.marker.MarkAndClean(gc.taskTerminal); err != nil {
		gc.log.Error("gc.Marker.Mark failed, error is "+err.Error(), "method", "runTask")
		return
	}
}
