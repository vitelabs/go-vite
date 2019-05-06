package pool

import (
	"time"

	"github.com/vitelabs/go-vite/common"
)

type poolEventBus struct {
	newABlock           chan struct{}
	newSBlock           chan struct{}
	snapshotForkChecker *time.Ticker
	broadcasterT        *time.Ticker
	clearT              *time.Ticker

	wait *common.CondTimer
}

func (bus *poolEventBus) newABlockEvent() {
	select {
	case bus.newABlock <- struct{}{}:
	default:
	}
	bus.wait.Signal()
}
func (bus *poolEventBus) newSBlockEvent() {
	select {
	case bus.newSBlock <- struct{}{}:
	default:
	}
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
		select {
		case <-bus.newABlock:
			sum += w.p.accountsCompact()
		case <-w.closed:
			return
		default:
		}

		select {
		case <-bus.newSBlock:
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
		case <-bus.clearT.C:
			w.p.delUseLessChains()
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

		if sum > 0 {
			w.p.insert()
		} else {
			w.p.compact()
			w.p.insert()
		}
		bus.wait.Wait()
	}
}

func (w *worker) init() {
	checkForkT := time.NewTicker(time.Second * 2)
	broadcasterT := time.NewTicker(time.Second * 30)
	clearT := time.NewTicker(time.Minute)

	w.bus = &poolEventBus{
		newABlock:           make(chan struct{}, 1),
		newSBlock:           make(chan struct{}, 1),
		snapshotForkChecker: checkForkT,
		broadcasterT:        broadcasterT,
		clearT:              clearT,
		wait:                common.NewCondTimer(),
	}
}
