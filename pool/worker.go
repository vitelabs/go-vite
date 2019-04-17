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

func (self *poolEventBus) newABlockEvent() {
	select {
	case self.newABlock <- struct{}{}:
	default:
	}
	self.wait.Signal()
}
func (self *poolEventBus) newSBlockEvent() {
	select {
	case self.newSBlock <- struct{}{}:
	default:
	}
	self.wait.Signal()
}

type worker struct {
	p   *pool
	bus *poolEventBus

	closed chan struct{}
}

func (self *worker) work() {
	bus := self.bus

	bus.wait.Start(time.Millisecond * 40)
	defer bus.wait.Stop()
	for {
		sum := 0
		select {
		case <-bus.newABlock:
			sum += self.p.accountsCompact()
		case <-self.closed:
			return
		default:
		}

		select {
		case <-bus.newSBlock:
			sum += self.p.snapshotCompact()
		case <-self.closed:
			return
		default:
		}

		select {
		case <-bus.snapshotForkChecker.C:
			self.p.pendingSc.checkFork()
		case <-bus.broadcasterT.C:
			self.p.broadcastUnConfirmedBlocks()
		case <-bus.clearT.C:
			//self.p.delUseLessChains()
		case <-self.closed:
			return
		default:
		}
		if sum > 0 {
			self.p.insert()
		} else {
			self.p.compact()
			self.p.insert()
		}
		bus.wait.Wait()
	}
}

func (self *worker) init() {
	checkForkT := time.NewTicker(time.Second * 2)
	broadcasterT := time.NewTicker(time.Second * 30)
	clearT := time.NewTicker(time.Minute)

	self.bus = &poolEventBus{
		newABlock:           make(chan struct{}, 1),
		newSBlock:           make(chan struct{}, 1),
		snapshotForkChecker: checkForkT,
		broadcasterT:        broadcasterT,
		clearT:              clearT,
		wait:                common.NewCondTimer(),
	}
}
