package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"math/big"
	"sync"
)

type SimpleAutoReceiveFilterPair struct {
	tti      types.TokenTypeId
	minValue big.Int
}

type AutoReceiveWorker struct {
	log     log15.Logger
	address types.Address

	manager     *Manager
	generator   *generator.Generator
	uBlocksPool *model.UnconfirmedBlocksPool

	status                int
	isSleeping            bool
	breaker               chan struct{}
	stopListener          chan struct{}
	newUnconfirmedTxAlarm chan struct{}

	filters map[types.TokenTypeId]big.Int

	statusMutex sync.Mutex
}

func NewAutoReceiveWorker(manager *Manager, address types.Address, filters map[types.TokenTypeId]big.Int) *AutoReceiveWorker {
	return &AutoReceiveWorker{
		manager:     manager,
		uBlocksPool: manager.unconfirmedBlocksPool,
		generator:   generator.NewGenerator(manager.vite.Chain(), manager.vite.WalletManager().KeystoreManager),
		address:     address,
		status:      Create,
		isSleeping:  false,
		filters:     filters,
		log:         log15.New("worker", "a", "addr", address),
	}
}

func (w *AutoReceiveWorker) Start() {

	w.log.Info("Start()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {

		w.breaker = make(chan struct{})
		w.newUnconfirmedTxAlarm = make(chan struct{})
		w.stopListener = make(chan struct{})

		w.uBlocksPool.AddCommonTxLis(w.address, func() {
			w.NewUnconfirmedTxAlarm()
		})

		w.uBlocksPool.AcquireAccountInfoCache(w.address)

		go w.startWork()

		w.status = Start

	} else {
		// awake it in order to run at least once
		w.NewUnconfirmedTxAlarm()
	}
	w.log.Info("end start")
}

func (w *AutoReceiveWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {

		w.uBlocksPool.ReleaseAccountInfoCache(w.address)

		w.breaker <- struct{}{}
		close(w.breaker)

		w.uBlocksPool.RemoveCommonTxLis(w.address)
		close(w.newUnconfirmedTxAlarm)

		// make sure we can stop the worker
		<-w.stopListener
		close(w.stopListener)

		w.status = Stop
	}
	w.log.Info("stopped")
}

func (w *AutoReceiveWorker) ResetAutoReceiveFilter(filters map[types.TokenTypeId]big.Int) {
	w.log.Info("ResetAutoReceiveFilter", "len", len(filters))
	w.filters = filters
	if w.Status() == Start {
		w.uBlocksPool.ResetCacheCursor(w.address)
	}
}

func (w *AutoReceiveWorker) startWork() {
	w.log.Info("startWork")

	for {
		w.isSleeping = false
		if w.Status() == Stop {
			goto END
		}

		tx := w.uBlocksPool.GetNextTx(w.address)
		if tx != nil {
			minAmount, ok := w.filters[tx.TokenId]
			if !ok || tx.Amount.Cmp(&minAmount) < 0 {
				continue
			}
			w.ProcessOneBlock(tx)
			continue
		}

		w.isSleeping = true
		select {
		case <-w.newUnconfirmedTxAlarm:
			w.log.Info("start awake")
		case <-w.breaker:
			w.log.Info("worker broken")
			goto END
		}
	}

END:
	w.log.Info("startWork end called")
	w.stopListener <- struct{}{}
	w.log.Info("startWork end")
}

func (w *AutoReceiveWorker) Close() error {
	w.Stop()
	return nil
}

func (w AutoReceiveWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *AutoReceiveWorker) NewUnconfirmedTxAlarm() {
	if w.isSleeping {
		w.newUnconfirmedTxAlarm <- struct{}{}
	}
}

func (w *AutoReceiveWorker) ProcessOneBlock(sendBlock *ledger.AccountBlock) {
	if w.manager.checkExistInPool(sendBlock.ToAddress, sendBlock.FromBlockHash) {
		w.log.Info("ProcessOneBlock.checkExistInPool failed")
		return
	}

	err := w.generator.PrepareVm(nil, nil, &sendBlock.ToAddress)
	if err != nil {
		w.log.Error("ProcessOneBlock.PrepareVm failed", "error", err)
		return
	}

	genResult, genErr := w.generator.GenerateWithUnconfirmed(*sendBlock, nil,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			return w.generator.Sign(addr, nil, data)
		})
	if genErr != nil {
		w.log.Error("GenerateTx error ignore, ", "error", genErr)
		return
	}

	poolErr := w.manager.insertCommonBlockToPool(genResult.BlockGenList)
	if poolErr != nil {
		w.log.Error("insertCommonBlockToPool failed, ", "error", poolErr)
		return
	}
}
