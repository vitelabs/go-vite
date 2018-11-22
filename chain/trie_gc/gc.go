package trie_gc

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/log15"
	"sync"
	"time"
)

const (
	STATUS_STOPPED       = 1
	STATUS_STARTED       = 2
	STATUS_MARKING       = 3
	STATUS_FILTER_MARKED = 4
	STATUS_CLEANING      = 5
)

type collector struct {
	terminal     chan struct{}
	taskTerminal chan struct{}

	statusLock sync.Mutex
	status     uint8 // 0 is stopped, 1 is started

	wg sync.WaitGroup

	checkInterval time.Duration
	ticker        *time.Ticker

	chain Chain

	log log15.Logger

	minRetain uint64

	marker *Marker
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
		gc.log.Error("gcDb open failed, error is "+err.Error(), "method", "NewCollector")
		return nil, err
	}

	if mk, err := NewMarker(chain, gcDb); err != nil {
		gc.log.Error("newMarker failed, error is "+err.Error(), "method", "NewCollector")
		return nil, err
	} else {
		gc.marker = mk
	}

	return gc, nil
}

func (gc *collector) Start() {
	gc.statusLock.Lock()
	defer gc.statusLock.Unlock()
	if gc.status >= STATUS_STARTED {
		gc.log.Error("gc is started, don't start again")
		return
	}

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
	gc.taskTerminal <- struct{}{}
	gc.terminal <- struct{}{}

	gc.wg.Wait()
	gc.status = STATUS_STOPPED
}

func (gc *collector) Status() uint8 {
	return gc.status
}

func (gc *collector) ClearedHeight() uint64 {
	return gc.marker.clearedHeight
}

func (gc *collector) MarkedHeight() uint64 {
	return gc.marker.markedHeight
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

	defer func() {
		gc.statusLock.Lock()
		gc.status = STATUS_STARTED
		gc.statusLock.Unlock()
	}()

	gc.statusLock.Lock()
	gc.status = STATUS_MARKING
	gc.statusLock.Unlock()
	if isTerminal, err := gc.marker.Mark(targetHeight, gc.taskTerminal); err != nil {
		gc.log.Error("gc.Marker.Mark failed, error is "+err.Error(), "method", "runTask")
		return
	} else if isTerminal {
		return
	}

	gc.statusLock.Lock()
	gc.status = STATUS_FILTER_MARKED
	gc.statusLock.Unlock()
	if err := gc.marker.FilterMarked(); err != nil {
		gc.log.Error("gc.Marker.FilterMarked failed, error is "+err.Error(), "method", "runTask")
		return
	}

	gc.statusLock.Lock()
	gc.status = STATUS_CLEANING
	gc.statusLock.Unlock()

	if isTerminal, err := gc.marker.Clean(gc.taskTerminal); err != nil {
		gc.log.Error("gc.Marker.clean failed, error is "+err.Error(), "method", "runTask")
		return
	} else if isTerminal {
		return
	}

}

func (gc *collector) getTargetHeight() uint64 {
	latestSnapshotBlock := gc.chain.GetLatestSnapshotBlock()
	if latestSnapshotBlock == nil {
		return 0
	}

	latestHeight := gc.chain.GetLatestSnapshotBlock().Height
	if latestHeight < gc.minRetain {
		return 0
	}

	return latestHeight - gc.minRetain + 1
}
