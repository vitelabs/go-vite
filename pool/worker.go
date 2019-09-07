package pool

import (
	"math/rand"
	"time"

	"github.com/vitelabs/go-vite/common"
)

type poolEventBus struct {
	snapshotForkChecker *time.Ticker
	accountCompactT     *time.Ticker
	snapshotCompactT    *time.Ticker
	broadcasterT        *time.Ticker
	clearT              *time.Ticker
	accDestroyT         *time.Ticker
	irreversibleT       *time.Ticker

	accContext      *poolContext
	snapshotContext *poolContext

	wait *common.CondTimer
}

func (bus *poolEventBus) newABlockEvent() {
	bus.accContext.setCompactDirty(true)
	bus.wait.Signal()
}
func (bus *poolEventBus) newSBlockEvent() {
	bus.snapshotContext.setCompactDirty(true)
	bus.wait.Signal()
}

type worker struct {
	p   *pool
	bus *poolEventBus

	closed chan struct{}
}

func (w *worker) work() {
	bus := w.bus

	bus.wait.Start(time.Millisecond * 40)
	defer bus.wait.Stop()
	for {
		sum := 0
		if bus.accContext.compactDirty {
			w.p.log.Debug("account compacts")
			bus.accContext.setCompactDirty(false)
			sum += w.p.accountsCompact(true)
		}

		if bus.snapshotContext.compactDirty {
			bus.snapshotContext.setCompactDirty(false)
			sum += w.p.snapshotCompact()
		}

		select {
		case <-bus.accountCompactT.C:
			sum += w.p.accountsCompact(false)
		case <-w.closed:
			return
		default:
		}

		select {
		case <-bus.snapshotCompactT.C:
			sum += w.p.snapshotCompact()
		case <-w.closed:
			return
		default:
		}

		select {
		case <-bus.snapshotForkChecker.C:
			w.p.checkFork()
		case <-bus.broadcasterT.C:
			w.p.broadcastUnConfirmedBlocks()
		case <-bus.accDestroyT.C:
			w.p.destroyAccounts()
		case <-bus.clearT.C:
			w.p.delUseLessChains()
		case <-bus.irreversibleT.C:
			w.p.updateIrreversibleBlock()
		case <-w.closed:
			return
		default:
		}

		chunks := w.p.ReadDownloadedChunks()
		if chunks != nil {
			result := w.p.insertChunks(chunks)
			if result {
				continue
			}
		}

		if sum > 0 || rand.Intn(10) > 6 {
			w.p.insert()
			continue
		}
		if bus.accContext.compactDirty || bus.snapshotContext.compactDirty {
			continue
		}
		bus.wait.Wait()
	}
}

func (w *worker) init() {
	checkForkT := time.NewTicker(time.Second * 2)
	broadcasterT := time.NewTicker(time.Second * 30)
	clearT := time.NewTicker(time.Minute)
	accDestroyT := time.NewTicker(time.Minute * 10)
	irreversibleT := time.NewTicker(time.Minute * 3)
	accountCompactT := time.NewTicker(time.Second * 6)
	snapshotCompactT := time.NewTicker(time.Second * 1)

	w.bus = &poolEventBus{
		snapshotForkChecker: checkForkT,
		broadcasterT:        broadcasterT,
		accountCompactT:     accountCompactT,
		snapshotCompactT:    snapshotCompactT,
		accContext:          &poolContext{},
		snapshotContext:     &poolContext{},
		clearT:              clearT,
		accDestroyT:         accDestroyT,
		irreversibleT:       irreversibleT,
		wait:                common.NewCondTimer(),
	}
}
