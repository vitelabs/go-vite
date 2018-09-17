package unconfirmed

import (
	"container/heap"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"sync"
)

type ContractWorker struct {
	manager *Manager

	uBlocksPool *model.UnconfirmedBlocksPool

	gid                 types.Gid
	address             types.Address
	accEvent            producer.AccountStartEvent
	contractAddressList []*types.Address

	status      int
	statusMutex sync.Mutex

	isSleep               bool
	newUnconfirmedTxAlarm chan struct{}

	breaker                chan struct{}
	stopDispatcherListener chan struct{}

	contractTasks []*ContractTask

	priorityToQueue      *model.PriorityToQueue
	priorityToQueueMutex sync.RWMutex

	blackList      map[types.Hash]bool // map[(toAddress+fromAddress).String]
	blackListMutex sync.RWMutex

	log log15.Logger
}

func NewContractWorker(manager *Manager, accEvent producer.AccountStartEvent) (*ContractWorker, error) {

	addressList, err := manager.unconfirmedBlocksPool.GetAddrListByGid(accEvent.Gid)
	if err != nil {
		return nil, err
	}
	if len(addressList) <= 0 {
		return nil, errors.New("newContractWorker addressList nil")
	}

	return &ContractWorker{
		manager:                manager,
		uBlocksPool:            manager.unconfirmedBlocksPool,
		gid:                    accEvent.Gid,
		address:                accEvent.Address,
		accEvent:               accEvent,
		contractAddressList:    addressList,
		status:                 Create,
		isSleep:                false,
		newUnconfirmedTxAlarm:  make(chan struct{}),
		breaker:                make(chan struct{}),
		stopDispatcherListener: make(chan struct{}),
		contractTasks:          make([]*ContractTask, CONTRACT_TASK_SIZE),
		blackList:              make(map[types.Hash]bool),
		log:                    log15.New("worker", "c", "addr", accEvent.Address, "gid", accEvent.Gid),
	}, nil

}

func (w *ContractWorker) Start() {
	w.log.Info("Start()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {

		w.uBlocksPool.AddContractLis(w.gid, func() {
			w.NewUnconfirmedTxAlarm()
		})

		for i, v := range w.contractTasks {
			v = NewContractTask(w, i, w.dispatchTask)
			v.Start()
		}

		go w.waitingNewBlock()

		w.status = Start
	} else {
		// awake it in order to run at least once
		w.NewUnconfirmedTxAlarm()
	}
	w.log.Info("end start")
}

func (w *ContractWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {

		w.breaker <- struct{}{}
		// todo: to clear tomap

		w.uBlocksPool.RemoveContractLis(w.gid)
		w.isSleep = true
		close(w.newUnconfirmedTxAlarm)

		<-w.stopDispatcherListener
		close(w.stopDispatcherListener)

		// todo 2. Stop all task
		for _, v := range w.contractTasks {
			v.Stop()
		}
		w.status = Stop
	}
	w.log.Info("stopped")
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

func (w *ContractWorker) dispatchTask(index int) *model.FromItem {
	w.log.Info("dispatchTask", "index", index)
	w.priorityToQueueMutex.Lock()
	defer w.priorityToQueueMutex.Unlock()

	if w.priorityToQueue.Len() == 0 {
		w.log.Info("priorityToQueue empty now get from db")
		w.FetchNewFromDb()
		if w.priorityToQueue.Len() == 0 {
			w.log.Info("priorityToQueue db empty")
			return nil
		}
		return nil
	}

	tItem := heap.Pop(w.priorityToQueue).(*model.ToItem)
	priorityFromQueue := tItem.Value
	for j := 0; j < priorityFromQueue.Len(); j++ {
		fItem := heap.Pop(priorityFromQueue).(*model.FromItem)
		return fItem
	}
	return nil
}

func (w *ContractWorker) NewUnconfirmedTxAlarm() {
	if w.isSleep {
		w.newUnconfirmedTxAlarm <- struct{}{}
	}
}

func (w *ContractWorker) waitingNewBlock() {
	w.log.Info("waitingNewBlock")
	for {
		w.isSleep = false
		if w.Status() == Stop {
			goto END
		}

		for _, v := range w.contractTasks {
			v.WakeUp()
		}

		w.isSleep = true
		select {
		case <-w.newUnconfirmedTxAlarm:
			w.log.Info("start awake")
		case <-w.breaker:
			w.log.Info("worker broken")
			goto END
		}
	}
END:
	w.log.Info("waitingNewBlock end called")
	w.stopDispatcherListener <- struct{}{}
	w.log.Info("waitingNewBlock end")
}

func (w *ContractWorker) FetchNewFromDb() {
	snapshotHash := w.accEvent.SnapshotHash
	for i := 0; i < len(w.contractAddressList); i++ {
		blockList, err := w.uBlocksPool.GetUnconfirmedBlocks(0, 1, CONTRACT_FETCH_SIZE, w.contractAddressList[i])
		if err != nil {
			w.log.Error("FetchNewFromDb.GetUnconfirmedBlocks", "error", err)
			continue
		}
		for _, v := range blockList {
			// when a to-from pair  was added into blackList,
			// the other block which under the same to-from pair won't fetch any more during the same block-out period
			if w.isInBlackList(v.AccountAddress, v.ToAddress) {
				fromQuota := w.manager.uAccess.GetAccountQuota(v.AccountAddress, snapshotHash)
				toQuota := w.manager.uAccess.GetAccountQuota(v.ToAddress, snapshotHash)
				w.priorityToQueue.InsertNew(v, toQuota, fromQuota)
			}
		}
	}
}

func (w *ContractWorker) addIntoBlackList(from types.Address, to types.Address) {
	w.log.Info("addIntoBlackList", "from", from, "to", to)
	key := types.DataListHash(from.Bytes(), to.Bytes())
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList[key] = true
}

func (w *ContractWorker) deleteBlackListItem(from types.Address, to types.Address) {
	w.log.Info("deleteBlackListItem", "from", from, "to", to)
	key := types.DataListHash(from.Bytes(), to.Bytes())
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	delete(w.blackList, key)
}

func (w *ContractWorker) isInBlackList(from types.Address, to types.Address) bool {
	key := types.DataListHash(from.Bytes(), to.Bytes())
	w.blackListMutex.RLock()
	defer w.blackListMutex.RUnlock()
	_, ok := w.blackList[key]
	w.log.Info("isInBlackList", "from", from, "to", to, "in", ok)
	return ok
}
