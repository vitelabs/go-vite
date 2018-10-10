package producer

import (
	"sync/atomic"

	"fmt"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
)

// Package producer implements vite block creation

var mLog = log15.New("module", "producer")

type Producer interface {
	SetAccountEventFunc(func(producerevent.AccountEvent))
}

// Backend wraps all methods required for mining.
type SnapshotChainRW interface {
	WriteMiningBlock(block *ledger.SnapshotBlock) error
}

type DownloaderRegister func(chan<- int) // 0 represent success, not 0 represent failed.

/**

0->1->2->3->4->5->6->7->8
		 ^|_______\
*/
// 0:origin 1: initing 2:inited 3:starting 4:started 5:stopping 6:stopped 7:destroying 8:destroyed
type producerLifecycle struct {
	common.LifecycleStatus
}

func (self *producerLifecycle) PreDestroy() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 6, 7)
}
func (self *producerLifecycle) PostDestroy() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 7, 8)
}

func (self *producerLifecycle) PreStart() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 2, 3) || atomic.CompareAndSwapInt32(&self.Status, 6, 3)
}
func (self *producerLifecycle) PostStart() bool {
	return atomic.CompareAndSwapInt32(&self.Status, 3, 4)
}

type producer struct {
	producerLifecycle
	tools                *tools
	mining               int32
	coinbase             types.Address // address
	worker               *worker
	cs                   consensus.Subscriber
	downloaderRegister   DownloaderRegister
	downloaderRegisterCh chan int
	dwlFinished          bool
	accountFn            func(producerevent.AccountEvent)
}

// todo syncDone
func NewProducer(rw chain.Chain,
	downloaderRegister DownloaderRegister,
	coinbase types.Address,
	cs consensus.Subscriber,
	verifier *verifier.SnapshotVerifier,
	wt *wallet.Manager,
	p pool.SnapshotProducerWriter) *producer {
	chain := newChainRw(rw, verifier, wt, p)
	miner := &producer{tools: chain, coinbase: coinbase}

	miner.cs = cs
	miner.worker = newWorker(chain, coinbase)
	miner.downloaderRegister = downloaderRegister
	miner.downloaderRegisterCh = make(chan int)
	miner.dwlFinished = false
	return miner
}
func (self *producer) Init() error {
	if !self.PreInit() {
		return errors.New("pre init fail.")
	}
	defer self.PostInit()

	if !self.tools.checkAddressLock(self.coinbase) {
		return errors.New(fmt.Sprintf("coinbase[%s] must be unlock.", self.coinbase.String()))
	}

	if err := self.worker.Init(); err != nil {
		return err
	}

	return nil
}

func (self *producer) Start() error {
	if !self.PreStart() {
		return errors.New("pre start fail.")
	}
	defer self.PostStart()

	err := self.worker.Start()
	if err != nil {
		return err
	}

	snapshotId := self.coinbase.String() + "_snapshot"
	contractId := self.coinbase.String() + "_contract"

	self.cs.Subscribe(types.SNAPSHOT_GID, snapshotId, &self.coinbase, self.worker.produceSnapshot)
	self.cs.Subscribe(types.DELEGATE_GID, contractId, &self.coinbase, self.producerContract)

	return nil
}

func (self *producer) Stop() error {
	if !self.PreStop() {
		return errors.New("pre stop fail.")
	}
	defer self.PostStop()

	snapshotId := self.coinbase.String() + "_snapshot"
	contractId := self.coinbase.String() + "_contract"

	self.cs.UnSubscribe(types.SNAPSHOT_GID, snapshotId)
	self.cs.UnSubscribe(types.DELEGATE_GID, contractId)

	err := self.worker.Stop()
	if err != nil {
		return err
	}
	return nil
}

func (self *producer) producerContract(e consensus.Event) {
	fn := self.accountFn

	if fn != nil {
		if !self.tools.checkAddressLock(e.Address) {
			mLog.Error("coinbase must be unlock.", "addr", e.Address.String())
			return
		}
		go fn(producerevent.AccountStartEvent{
			Gid:            e.Gid,
			Address:        e.Address,
			Stime:          e.Stime,
			Etime:          e.Etime,
			Timestamp:      e.Timestamp,
			SnapshotHeight: e.SnapshotHeight,
			SnapshotHash:   e.SnapshotHash,
		})
	}
}

func (self *producer) SetAccountEventFunc(accountFn func(producerevent.AccountEvent)) {
	self.accountFn = accountFn
}
