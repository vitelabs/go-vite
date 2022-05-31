package producer

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/chain"
	"github.com/vitelabs/go-vite/v2/ledger/consensus"
	"github.com/vitelabs/go-vite/v2/ledger/pool"
	"github.com/vitelabs/go-vite/v2/log15"
	"github.com/vitelabs/go-vite/v2/net"
	"github.com/vitelabs/go-vite/v2/producer/producerevent"
)

// Package producer implements vite block creation

var mLog = log15.New("module", "producer")

// Backend wraps all methods required for mining.
type SnapshotChainRW interface {
	WriteMiningBlock(block *ledger.SnapshotBlock) error
}

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
	tools      *tools
	mining     int32
	coinbase   interfaces.Account
	worker     *worker
	cs         consensus.Subscriber
	subscriber net.Subscriber
	accountFn  func(producerevent.AccountEvent)
	syncState  net.SyncState
	netSyncId  int
}

// todo syncDone
func NewProducer(rw chain.Chain,
	subscriber net.Subscriber,
	coinbase interfaces.Account,
	cs consensus.Subscriber,
	p pool.SnapshotProducerWriter) *producer {
	chain := newChainRw(rw, p)
	miner := &producer{tools: chain, coinbase: coinbase}

	miner.cs = cs
	miner.worker = newWorker(chain, coinbase)
	miner.subscriber = subscriber
	return miner
}
func (self *producer) Init() error {
	if !self.PreInit() {
		return errors.New("pre init fail")
	}
	defer self.PostInit()

	if err := self.worker.Init(); err != nil {
		return err
	}
	wLog.Info("init.")
	return nil
}

func (self *producer) Start() error {
	if !self.PreStart() {
		return errors.New("pre start fail")
	}
	defer self.PostStart()

	err := self.worker.Start()
	if err != nil {
		return err
	}
	if self.coinbase == nil {
		return errors.New("coinbase must not be nil")
	}

	snapshotId := self.coinbase.Address().Hex() + "_snapshot"
	contractId := self.coinbase.Address().Hex() + "_contract"

	addr := self.coinbase.Address()
	self.cs.Subscribe(types.SNAPSHOT_GID, snapshotId, &addr, func(e consensus.Event) {
		mLog.Info("snapshot producer trigger.", "addr", self.coinbase.Address, "syncState", self.syncState, "e", e)
		if self.syncState == net.SyncDone {
			self.worker.produceSnapshot(e)
		}
	})
	self.cs.Subscribe(types.DELEGATE_GID, contractId, &addr, func(e consensus.Event) {
		mLog.Info("contract producer trigger.", "addr", self.coinbase.Address, "syncState", self.syncState, "e", e)
		if self.syncState == net.SyncDone {
			self.producerContract(e)
		}
	})

	self.syncState = self.subscriber.SyncState()
	id := self.subscriber.SubscribeSyncStatus(func(state net.SyncState) {
		self.syncState = state
	})
	self.netSyncId = id
	wLog.Info("started.")
	return nil
}

func (self *producer) SnapshotOnce() error {
	t := time.Now()
	e := &consensus.Event{
		Gid:         types.SNAPSHOT_GID,
		Address:     self.coinbase.Address(),
		Stime:       t,
		Etime:       t,
		Timestamp:   t,
		VoteTime:    t,
		PeriodStime: t,
		PeriodEtime: t,
	}
	self.worker.genAndInsert(e)
	return nil
}

func (self *producer) Stop() error {
	if !self.PreStop() {
		return errors.New("pre stop fail")
	}
	defer self.PostStop()

	snapshotId := self.coinbase.Address().Hex() + "_snapshot"
	contractId := self.coinbase.Address().Hex() + "_contract"

	self.cs.UnSubscribe(types.SNAPSHOT_GID, snapshotId)
	self.cs.UnSubscribe(types.DELEGATE_GID, contractId)

	self.subscriber.UnsubscribeSyncStatus(self.netSyncId)
	self.netSyncId = 0

	err := self.worker.Stop()
	if err != nil {
		return err
	}
	return nil
}

func (self *producer) producerContract(e consensus.Event) {
	fn := self.accountFn

	if fn != nil {
		if e.Address != self.coinbase.Address() {
			mLog.Error("coinbase can't match.", "addr", e.Address.String(), "coinbase", self.coinbase.Address())
			return
		}

		header := self.tools.chain.GetLatestSnapshotBlock()
		if header.Timestamp.Before(e.VoteTime) {
			mLog.Error(fmt.Sprintf("contract producer fail. snapshot is too lower. voteTime:%s, headerTime:%s, headerHeight:%d, headerHash:%s", e.VoteTime, header.Timestamp, header.Height, header.Hash))
			return
		}
		if err := self.tools.checkStableSnapshotChain(header); err != nil {
			mLog.Error(fmt.Sprintf("contract producer fail. snapshot is not stable version. voteTime:%s, startTime:%s, endTime:%s", e.VoteTime, e.Stime, e.Etime))
			return
		}

		tmpEvent := producerevent.AccountStartEvent{
			Gid:     e.Gid,
			Address: e.Address,
			Stime:   e.Stime,
			Etime:   e.Etime,
		}
		common.Go(func() {
			fn(tmpEvent)
		})
	}
}

func (self *producer) SetAccountEventFunc(accountFn func(producerevent.AccountEvent)) {
	self.accountFn = accountFn
}

func (self *producer) GetCoinBase() types.Address {
	return self.coinbase.Address()
}
