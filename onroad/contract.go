package onroad

import (
	"container/heap"
	"sync"

	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad/model"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vm/contracts"
	"strconv"
)

type ContractWorker struct {
	manager *Manager

	uBlocksPool *model.OnroadBlocksPool

	gid      types.Gid
	address  types.Address
	accEvent producerevent.AccountStartEvent

	status      int
	statusMutex sync.Mutex

	isSleep                bool
	isCancel               bool
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

func NewContractWorker(manager *Manager) *ContractWorker {
	worker := &ContractWorker{
		manager:     manager,
		uBlocksPool: manager.onroadBlocksPool,

		status:   Create,
		isSleep:  false,
		isCancel: false,

		blackList: make(map[types.Address]bool),
	}

	processors := make([]*ContractTaskProcessor, ContractTaskProcessorSize)
	for i, _ := range processors {
		processors[i] = NewContractTaskProcessor(worker, i)
	}
	worker.contractTaskProcessors = processors

	return worker
}

func (w ContractWorker) getAccEvent() *producerevent.AccountStartEvent {
	return &w.accEvent
}

func (w *ContractWorker) Start(accEvent producerevent.AccountStartEvent) {
	w.gid = accEvent.Gid
	w.address = accEvent.Address
	w.accEvent = accEvent
	w.log = slog.New("worker", "c", "addr", accEvent.Address, "gid", accEvent.Gid)

	log := w.log.New("method", "start")
	log.Info("Start() current status" + strconv.Itoa(w.status))
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {
		w.isCancel = false

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
		log.Info("get addresslist", "len", len(addressList))

		// 2. get getAndSortAllAddrQuota it is a heavy operation so we call it only once in Start
		w.getAndSortAllAddrQuota()
		log.Info("getAndSortAllAddrQuota", "len", len(w.contractTaskPQueue))

		// 3. init some local variables
		w.newOnroadTxAlarm = make(chan struct{})
		w.breaker = make(chan struct{})
		w.stopDispatcherListener = make(chan struct{})

		w.uBlocksPool.AddContractLis(w.gid, func(address types.Address) {
			if w.isInBlackList(address) {
				return
			}

			q := w.manager.Chain().GetPledgeQuota(w.accEvent.SnapshotHash, address)
			c := &contractTask{
				Addr:  address,
				Quota: q,
			}

			w.ctpMutex.Lock()
			heap.Push(&w.contractTaskPQueue, c)
			w.ctpMutex.Unlock()

			w.NewOnroadTxAlarm()
		})

		log.Info("start all tp")
		for _, v := range w.contractTaskProcessors {
			v.Start()
		}
		log.Info("end start all tp")
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
		w.isCancel = true

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
		w.log.Info("end stop all task")
		w.status = Stop
	}
	w.log.Info("stopped")
}

func (w *ContractWorker) waitingNewBlock() {
	mlog := w.log.New("method", "waitingNewBlock")
	mlog.Info("im in work")
LOOP:
	for {
		w.isSleep = false
		if w.isCancel {
			mlog.Info("found cancel true")
			break
		}
		w.ctpMutex.RLock()
		if w.contractTaskPQueue.Len() == 0 {
			w.ctpMutex.RUnlock()
		} else {
			w.ctpMutex.RUnlock()
			for _, v := range w.contractTaskProcessors {
				if v == nil {
					mlog.Error("tp is nil. wakeup")
					continue
				}
				mlog.Debug("before WakeUp")
				v.WakeUp()
				mlog.Debug("after WakeUp")
			}
		}

		w.isSleep = true
		mlog.Info("start sleep c")
		select {
		case <-w.newOnroadTxAlarm:
			mlog.Info("newOnroadTxAlarm start awake")
		case <-w.breaker:
			mlog.Info("worker broken")
			break LOOP
		}
	}

	mlog.Info("end called")
	w.stopDispatcherListener <- struct{}{}
	mlog.Info("end")
}

func (w *ContractWorker) getAndSortAllAddrQuota() {
	quotas := w.manager.Chain().GetPledgeQuotas(w.accEvent.SnapshotHash, w.contractAddressList)

	w.contractTaskPQueue = make([]*contractTask, len(quotas))
	i := 0
	for addr, quota := range quotas {
		task := &contractTask{
			Addr:  addr,
			Index: i,
			Quota: quota,
		}
		if addr == contracts.AddressPledge {
			task.Quota = math.MaxUint64
		}
		w.contractTaskPQueue[i] = task
		i++
	}

	heap.Init(&w.contractTaskPQueue)
}

func (w *ContractWorker) NewOnroadTxAlarm() {
	w.log.Info("NewOnroadTxAlarm", "isSleep", w.isSleep)
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
