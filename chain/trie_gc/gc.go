package trie_gc

import (
	"fmt"
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

	ledgerGcRetain uint64
	log            log15.Logger

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

func (gc *collector) Check() (bool, error) {
	checkSnapshotBlockNum := gc.marker.RetainSnapshotHeight()
	const (
		numPerCheck = 100
	)

	latestSnapshotBlock := gc.chain.GetLatestSnapshotBlock()
	startSnapshotBlockHeight := uint64(1)

	if latestSnapshotBlock.Height > checkSnapshotBlockNum {
		startSnapshotBlockHeight = latestSnapshotBlock.Height - checkSnapshotBlockNum + 1
	}

	fmt.Printf("check from %d to %d\n", startSnapshotBlockHeight, latestSnapshotBlock.Height)

	current := startSnapshotBlockHeight

	for current <= latestSnapshotBlock.Height {
		next := current + numPerCheck

		if next > latestSnapshotBlock.Height {
			next = latestSnapshotBlock.Height
		}
		count := next - current + 1
		sbList, err := gc.chain.GetSnapshotBlocksByHeight(current, count, true, false)
		if err != nil {
			return false, err
		}

		for _, sb := range sbList {
			result, err := gc.chain.ShallowCheckStateTrie(&sb.StateHash)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}

		current = next + 1
	}

	return true, nil
}

func (gc *collector) randomCheckInterval() time.Duration {
	minCheckInterval := int64(gc.minCheckInterval)
	maxCheckInterval := int64(gc.maxCheckInterval)
	return time.Duration(minCheckInterval + rand.Int63n(maxCheckInterval-minCheckInterval))
}

func (gc *collector) Start() {
	gc.statusLock.Lock()
	if gc.status >= STATUS_STARTED {
		gc.log.Error("gc is started, don't start again")
		gc.statusLock.Unlock()
		return
	}

	gc.status = STATUS_STARTED

	gc.ticker = time.NewTicker(gc.randomCheckInterval())
	gc.taskTerminal = make(chan struct{}, 1)
	gc.terminal = make(chan struct{}, 1)

	gc.wg.Add(1)
	gc.statusLock.Unlock()

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
	gc.log.Info("gc started.")
}

func (gc *collector) Stop() {
	gc.statusLock.Lock()
	if gc.status < STATUS_STARTED {
		gc.statusLock.Unlock()
		return
	}
	gc.status = STATUS_STOPPED

	gc.log.Info("gc stopping.")

	gc.ticker.Stop()
	close(gc.taskTerminal)
	close(gc.terminal)

	gc.statusLock.Unlock()

	gc.wg.Wait()
	gc.log.Info("gc stopped.")
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
	gc.log.Info("gc run task.")
	if err := gc.marker.MarkAndClean(gc.taskTerminal); err != nil {
		gc.log.Error("gc.Marker.Mark failed, error is "+err.Error(), "method", "runTask")
		return
	}
}
