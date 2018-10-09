package onroad

import (
	"container/heap"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad/model"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/verifier"
	"sync"
)

type ContractWorker struct {
	manager *Manager

	uBlocksPool *model.OnroadBlocksPool
	verifier    *verifier.AccountVerifier

	gid      types.Gid
	address  types.Address
	accEvent producerevent.AccountStartEvent

	status      int
	statusMutex sync.Mutex

	isSleep                bool
	newOnroadTxAlarm       chan struct{}
	breaker                chan struct{}
	stopDispatcherListener chan struct{}

	contractTaskProcessors []*ContractTaskProcessor
	contractAddressList    []types.Address

	contractTaskPQueue contractTaskPQueue
	ctpMutex           sync.RWMutex

	blackList      map[types.Address]bool
	blackListMutex sync.RWMutex

	log log15.Logger
}

func NewContractWorker(manager *Manager, accEvent producerevent.AccountStartEvent) *ContractWorker {
	return &ContractWorker{
		manager:     manager,
		uBlocksPool: manager.onroadBlocksPool,
		verifier:    verifier.NewAccountVerifier(manager.vite.Chain(), manager.vite),

		gid:      accEvent.Gid,
		address:  accEvent.Address,
		accEvent: accEvent,

		status:  Create,
		isSleep: false,

		contractTaskProcessors: make([]*ContractTaskProcessor, ContractTaskProcessorSize),
		blackList:              make(map[types.Address]bool),
		log:                    slog.New("worker", "c", "addr", accEvent.Address, "gid", accEvent.Gid),
	}
}

func (w *ContractWorker) Start() {
	w.log.Info("Start()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {

		// 1. get gid`s all contract address if error happened return immediately
		addressList, err := w.manager.uAccess.GetContractAddrListByGid(&w.gid)
		if err != nil {
			w.log.Error("GetAddrListByGid ", "err", err)
			return
		}
		if len(addressList) == 0 {
			w.log.Info("newContractWorker addressList nil")
			return
		}
		w.contractAddressList = addressList

		// 2. get getAndSortAllAddrQuota it is a heavy operation so we call it only once in Start
		if err = w.getAndSortAllAddrQuota(); err != nil {
			w.log.Error("getAndSortAllAddrQuota ", "err", err)
		}

		// 3. init some local variables
		w.newOnroadTxAlarm = make(chan struct{})
		w.breaker = make(chan struct{})
		w.stopDispatcherListener = make(chan struct{})

		w.uBlocksPool.AddContractLis(w.gid, func(address types.Address) {
			if w.isInBlackList(address) {
				return
			}

			q := w.manager.vite.Chain().GetPledgeQuota(w.accEvent.SnapshotHash, address)
			c := &contractTask{
				Addr:  address,
				Quota: q,
			}

			w.ctpMutex.Lock()
			heap.Push(&w.contractTaskPQueue, c)
			w.ctpMutex.Unlock()

			w.NewOnroadTxAlarm()
		})

		for i, v := range w.contractTaskProcessors {
			v = NewContractTaskProcessor(w, i)
			v.Start()
		}

		go w.waitingNewBlock()

		w.status = Start
	} else {
		// awake it in order to run at least once
		w.NewOnroadTxAlarm()
	}
	w.log.Info("end start")
}

func (w *ContractWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status == Start {

		w.breaker <- struct{}{}
		close(w.breaker)

		w.uBlocksPool.RemoveContractLis(w.gid)
		w.isSleep = true
		close(w.newOnroadTxAlarm)

		<-w.stopDispatcherListener
		close(w.stopDispatcherListener)

		w.log.Info("stop all task")
		wg := new(sync.WaitGroup)
		for _, v := range w.contractTaskProcessors {
			wg.Add(1)
			go func() {
				v.Stop()
				wg.Done()
			}()
		}
		wg.Wait()
		w.log.Info("all task stopped")
		w.status = Stop
		w.log.Info("stopped")
	}
}

func (w *ContractWorker) waitingNewBlock() {
	w.log.Info("waitingNewBlock")
LOOP:
	for {
		w.isSleep = false
		if w.Status() == Stop {
			break
		}

		w.ctpMutex.RLock()
		if w.contractTaskPQueue.Len() != 0 {
			w.ctpMutex.RUnlock()
			for _, v := range w.contractTaskProcessors {
				v.WakeUp()
			}
		}

		w.isSleep = true
		select {
		case <-w.newOnroadTxAlarm:
			w.log.Info("newOnroadTxAlarm start awake")
		case <-w.breaker:
			w.log.Info("worker broken")
			break LOOP
		}
	}

	w.log.Info("waitingNewBlock end called")
	w.stopDispatcherListener <- struct{}{}
	w.log.Info("waitingNewBlock end")
}

func (w *ContractWorker) getAndSortAllAddrQuota() error {
	w.contractTaskPQueue = make([]*contractTask, len(w.contractAddressList))

	quotas := w.manager.vite.Chain().GetPledgeQuotas(w.accEvent.SnapshotHash, w.contractAddressList)
	i := 0
	for key, value := range quotas {
		w.contractTaskPQueue[i] = &contractTask{
			Addr:  key,
			Index: i,
			Quota: value,
		}
		i++
	}
	heap.Init(&w.contractTaskPQueue)
	return nil
}

func (w *ContractWorker) NewOnroadTxAlarm() {
	if w.isSleep {
		w.newOnroadTxAlarm <- struct{}{}
	}
}

func (w *ContractWorker) pushContractTask(t *contractTask) {
	w.ctpMutex.Lock()
	defer w.ctpMutex.Unlock()
	heap.Push(&w.contractTaskPQueue, t)
}

func (w *ContractWorker) popContractTask() *contractTask {
	w.ctpMutex.Lock()
	defer w.ctpMutex.Unlock()
	if w.contractTaskPQueue.Len() > 0 {
		return heap.Pop(&w.contractTaskPQueue).(*contractTask)
	}
	return nil
}

func (w *ContractWorker) addIntoBlackList(addr types.Address) {
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList[addr] = true
}

func (w *ContractWorker) isInBlackList(addr types.Address) bool {
	w.blackListMutex.RLock()
	defer w.blackListMutex.RUnlock()
	_, ok := w.blackList[addr]
	if ok {
		w.log.Info("isInBlackList", "addr", addr, "in", ok)
	}
	return ok
}

func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}

func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}
