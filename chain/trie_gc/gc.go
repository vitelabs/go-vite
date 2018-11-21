package trie_gc

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/log15"
	"sync"
	"time"
)

const (
	status_stopped = 0
	status_started = 1
)

type collector struct {
	terminal     chan struct{}
	taskTerminal chan struct{}

	statusLock sync.Mutex
	status     int8 // 0 is stopped, 1 is started

	wg sync.WaitGroup

	checkInterval time.Duration
	ticker        *time.Ticker

	chain Chain

	log log15.Logger

	minRetain uint64

	marker *marker
}

func NewCollector(chain Chain, gcDbPath string) (Collector, error) {
	gc := &collector{
		checkInterval: time.Hour,
		chain:         chain,
		log:           log15.New("module", "trie_gc"),

		minRetain: 3600 * 24, // retain trie for about the most recent day
	}

	gcDb, err := leveldb.OpenFile(gcDbPath, nil)
	if err != nil {
		return nil, err
	}

	if mk, err := newMarker(chain, gcDb); err != nil {
		return nil, err
	} else {
		gc.marker = mk
	}

	return gc, nil
}

func (gc *collector) Start() {
	gc.statusLock.Lock()
	defer gc.statusLock.Unlock()
	if gc.status == status_started {
		gc.log.Error("gc is started, don't start again")
		return
	}
	gc.status = status_started

	gc.ticker = time.NewTicker(gc.checkInterval)
	gc.taskTerminal = make(chan struct{})
	gc.terminal = make(chan struct{})

	gc.wg.Add(1)
	go func() {
		defer gc.wg.Done()

		gc.checkAndRunTask()
		for {
			select {
			case <-gc.ticker.C:
				gc.checkAndRunTask()
			case <-gc.terminal:
				return
			}
		}

	}()
}

func (gc *collector) Stop() {
	gc.statusLock.Lock()
	defer gc.statusLock.Unlock()
	if gc.status == status_stopped {
		gc.log.Error("gc is stopped, don't stop again")
		return
	}
	gc.status = status_stopped

	gc.ticker.Stop()
	gc.taskTerminal <- struct{}{}
	gc.terminal <- struct{}{}

	gc.wg.Wait()
}
func (gc *collector) checkAndRunTask() {
	if gc.check() {
		gc.runTask()
	}
}

func (gc *collector) check() bool {
	needGc := true
	targetHeight := gc.getTargetHeight()
	if targetHeight <= 1 ||
		targetHeight <= gc.marker.ClearedHeight() {
		needGc = false
	}

	return needGc
}

func (gc *collector) runTask() {
	targetHeight := gc.getTargetHeight()
	if targetHeight <= 1 {
		return
	}

	if err := gc.marker.Mark(targetHeight, gc.taskTerminal); err != nil {
		gc.log.Error("gc.marker.Mark failed, error is "+err.Error(), "method", "runTask")
		return
	}
	if err := gc.marker.FilterMarked(); err != nil {
		gc.log.Error("gc.marker.FilterMarked failed, error is "+err.Error(), "method", "runTask")
		return
	}
	if err := gc.marker.Clean(gc.taskTerminal); err != nil {
		gc.log.Error("gc.marker.clean failed, error is "+err.Error(), "method", "runTask")
		return
	}
}

func (gc *collector) getTargetHeight() uint64 {
	latestHeight := gc.chain.GetLatestSnapshotBlock().Height
	if latestHeight < gc.minRetain {
		return 0
	}

	return latestHeight - gc.minRetain + 1
}
