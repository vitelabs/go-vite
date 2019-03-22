package onroad

import (
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad/model"
	"math/big"
	"sync"
)

type SimpleAutoReceiveFilterPair struct {
	tti      types.TokenTypeId
	minValue big.Int
}

type AutoReceiveWorker struct {
	log           log15.Logger
	address       types.Address
	powDifficulty *big.Int
	entropystore  string

	manager          *Manager
	onroadBlocksPool *model.OnroadBlocksPool

	status     int
	isSleeping bool
	isCancel   bool

	breaker          chan struct{}
	stopListener     chan struct{}
	newOnroadTxAlarm chan struct{}

	filters map[types.TokenTypeId]big.Int

	statusMutex sync.Mutex
}

func NewAutoReceiveWorker(manager *Manager, entropystore string, address types.Address, filters map[types.TokenTypeId]big.Int, powDifficulty *big.Int) *AutoReceiveWorker {
	return &AutoReceiveWorker{
		manager:          manager,
		entropystore:     entropystore,
		onroadBlocksPool: manager.onroadBlocksPool,
		address:          address,
		status:           Create,
		isSleeping:       false,
		isCancel:         false,
		filters:          filters,
		powDifficulty:    powDifficulty,
		log:              slog.New("worker", "chain", "addr", address),
	}
}

func (w AutoReceiveWorker) ResetPowDifficulty(powDifficulty *big.Int) {
	w.powDifficulty = powDifficulty
}

func (w AutoReceiveWorker) GetEntropystore() string {
	return w.entropystore
}

func (w *AutoReceiveWorker) Start() {

	w.log.Info("Start()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {

		w.isCancel = false

		w.breaker = make(chan struct{})
		w.newOnroadTxAlarm = make(chan struct{})
		w.stopListener = make(chan struct{})

		w.onroadBlocksPool.AddCommonTxLis(w.address, func() {
			w.NewOnroadTxAlarm()
		})

		w.onroadBlocksPool.AcquireFullOnroadBlocksCache(w.address)

		common.Go(w.startWork)

		w.status = Start

	} else {
		// awake it in order to run at least once
		w.NewOnroadTxAlarm()
	}
	w.log.Info("end start()")
}

func (w *AutoReceiveWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status == Start {

		w.isCancel = true

		w.onroadBlocksPool.ReleaseFullOnroadBlocksCache(w.address)

		w.breaker <- struct{}{}
		close(w.breaker)

		w.onroadBlocksPool.RemoveCommonTxLis(w.address)
		close(w.newOnroadTxAlarm)

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
	w.onroadBlocksPool.ResetCacheCursor(w.address)
}

func (w *AutoReceiveWorker) startWork() {
	w.log.Info("startWork")
LOOP:
	for {
		w.isSleeping = false
		if w.isCancel {
			w.log.Info("found cancel true")
			break
		}

		entropyStoreManager, e := w.manager.wallet.GetEntropyStoreManager(w.entropystore)
		if e != nil {
			w.log.Error("startWork ", "err", e)
			continue
		}

		if !entropyStoreManager.IsAddrUnlocked(w.address) {
			w.log.Error("startWork address locked", "addr", w.address)
			continue
		}

		tx := w.onroadBlocksPool.GetNextCommonTx(w.address)
		if tx != nil {
			if len(w.filters) == 0 {
				w.ProcessOneBlock(tx)
				continue
			}
			minAmount, ok := w.filters[tx.TokenId]
			if !ok || tx.Amount.Cmp(&minAmount) < 0 {
				continue
			}
			w.ProcessOneBlock(tx)
			continue
		}

		w.isSleeping = true
		w.log.Debug("start sleep")
		select {
		case <-w.newOnroadTxAlarm:
			w.log.Info("start awake")
		case <-w.breaker:
			w.log.Info("worker broken")
			break LOOP
		}
	}

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

func (w *AutoReceiveWorker) NewOnroadTxAlarm() {
	if w.isSleeping {
		w.newOnroadTxAlarm <- struct{}{}
	}
}

func (w *AutoReceiveWorker) ProcessOneBlock(sendBlock *ledger.AccountBlock) {
	if w.manager.checkExistInPool(sendBlock.ToAddress, sendBlock.FromBlockHash) {
		w.log.Info("ProcessOneBlock.checkExistInPool failed")
		return
	}

	var referredSnapshotHashList []types.Hash
	referredSnapshotHashList = append(referredSnapshotHashList, sendBlock.SnapshotHash)
	_, fitestSnapshotBlockHash, err := generator.GetFittestGeneratorSnapshotHash(w.manager.Chain(), &sendBlock.ToAddress, referredSnapshotHashList, true)
	if err != nil {
		w.log.Info("GetFittestGeneratorSnapshotHash failed", "error", err)
		return
	}
	gen, err := generator.NewGenerator(w.manager.Chain(), fitestSnapshotBlockHash, nil, &sendBlock.ToAddress)
	if err != nil {
		w.log.Error("NewGenerator failed", "error", err)
		return
	}

	genResult, err := gen.GenerateWithOnroad(*sendBlock, nil,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			manager, e := w.manager.wallet.GetEntropyStoreManager(w.entropystore)
			if e != nil {
				return nil, nil, e
			}
			return manager.SignData(addr, data)
		}, w.powDifficulty)
	if err != nil {
		w.log.Error("GenerateWithOnroad failed", "error", err)
		return
	}
	if genResult.Err != nil {
		w.log.Error("vm.Run error, ignore", "error", genResult.Err)
	}
	if len(genResult.BlockGenList) == 0 {
		w.log.Error("GenerateWithOnroad failed, BlockGenList is nil")
		return
	}

	poolErr := w.manager.insertCommonBlockToPool(genResult.BlockGenList)
	if poolErr != nil {
		w.log.Error("insertCommonBlockToPool failed, ", "error", poolErr)
		return
	}
}
