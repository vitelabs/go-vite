package worker

import (
	"container/heap"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/wallet"
	"sync"
)

type ContractWorker struct {
	wallet     *wallet.Manager
	blocksPool *model.UnconfirmedBlocksPool

	gid                 types.Gid
	address             types.Address
	accevent            producer.AccountStartEvent
	contractAddressList []*types.Address

	status      int
	statusMutex sync.Mutex

	dispatcherSleep        bool
	dispatcherAlarm        chan struct{}
	breaker                chan struct{}
	stopDispatcherListener chan struct{}

	contractTasks   []*ContractTask
	priorityToQueue *model.PriorityToQueue

	blackList      map[types.Hash]bool // map[(toAddress+fromAddress).String]
	blackListMutex sync.RWMutex

	log log15.Logger
}

func NewContractWorker(blocksPool *model.UnconfirmedBlocksPool, wallet *wallet.Manager, accevent producer.AccountStartEvent) (*ContractWorker, error) {

	addressList, err := blocksPool.GetAddrListByGid(accevent.Gid)

	if err != nil {
		return nil, err
	}

	if len(addressList) <= 0 {
		return nil, errors.New("newContractWorker addressList nil")
	}

	return &ContractWorker{
		blocksPool: blocksPool,
		wallet:     wallet,

		accevent:            accevent,
		gid:                 accevent.Gid,
		address:             accevent.Address,
		contractAddressList: addressList,

		status:          Create,
		dispatcherSleep: false,

		dispatcherAlarm:        make(chan struct{}),
		breaker:                make(chan struct{}),
		stopDispatcherListener: make(chan struct{}),

		contractTasks: make([]*ContractTask, CONTRACT_TASK_SIZE),
		blackList:     make(map[types.Hash]bool),

		log: log15.New("ContractWorker ", "addr", accevent.Address, "gid", accevent.Gid),
	}, nil
}

func (w *ContractWorker) addIntoBlackList(from types.Address, to types.Address) {
	key := types.DataListHash(from.Bytes(), to.Bytes())
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList[key] = true
}

func (w *ContractWorker) deleteBlackListItem(from types.Address, to types.Address) {
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
	return ok
}

func (w *ContractWorker) Start() {
	w.log.Info("worker startWork is called")
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {

		w.blocksPool.AddContractLis(w.gid, func() {
			w.NewUnconfirmedTxAlarm()
		})

		for i, v := range w.contractTasks {
			v = NewContractTask(w, i)
			go v.Start()
		}

		go w.DispatchTask(event.SnapshotHash)

		w.status = Start
	} else {
		// awake it in order to run at least once
		w.NewUnconfirmedTxAlarm()
	}

}

func (w *ContractWorker) Stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {

		w.breaker <- struct{}{}
		// todo: to clear tomap

		w.blocksPool.RemoveContractLis(w.gid)
		w.dispatcherSleep = true
		close(w.dispatcherAlarm)

		<-w.stopDispatcherListener
		close(w.stopDispatcherListener)

		// todo 2. Stop all task
		for _, v := range w.contractTasks {
			v.Stop()
		}
		w.status = Stop
	}
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

func (w *ContractWorker) NewUnconfirmedTxAlarm() {
	if w.dispatcherSleep {
		w.dispatcherAlarm <- struct{}{}
	}
}

func (w *ContractWorker) DispatchTask(snapshotHash *types.Hash) {
	//todo add mutex
	w.FetchNew(snapshotHash)
	for {
		for i := 0; i < w.priorityToQueue.Len(); i++ {
			tItem := heap.Pop(w.priorityToQueue).(*model.ToItem)
			priorityFromQueue := tItem.Value
			for j := 0; j < priorityFromQueue.Len(); j++ {
			FINDFREETASK:
				if w.Status() == Stop {
					// clear blackList
					w.blackList = nil
					// fixme: to clear priorityToQueue?
					goto END
				}

				freeTaskIndex := w.FindAFreeTask()
				if freeTaskIndex == -1 {
					goto FINDFREETASK
				}

				fItem := heap.Pop(priorityFromQueue).(*model.FromItem)
				w.contractTasks[freeTaskIndex].subQueue <- fItem
			}
		}

		w.dispatcherSleep = true

		select {
		case <-w.breaker:
			goto END
		case <-w.dispatcherAlarm:
			w.dispatcherSleep = false
			w.FetchNew(snapshotHash)
		}
	}
END:
	w.log.Info("ContractWorker send stopDispatcherListener")
	w.stopDispatcherListener <- struct{}{}
	w.log.Info("ContractWorker DispatchTask end")
}

func (w *ContractWorker) FindAFreeTask() (index int) {
	for k, v := range w.contractTasks {
		if v.status == Idle {
			return k
		}
	}
	return -1
}

func (w *ContractWorker) FetchNew(snapshotHash *types.Hash) {
	for i := 0; i < len(w.contractAddressList); i++ {
		blockList, err := w.uAccess.GetUnconfirmedBlocks(0, 1, CONTRACT_FETCH_SIZE, w.contractAddressList[i])
		if err != nil {
			w.log.Error("ContractWorker.FetchNew.GetUnconfirmedBlocks", "error", err)
			continue
		}
		for _, v := range blockList {
			// when a to-from pair  was added into blackList,
			// the other block which under the same to-from pair won't fetch any more during the same block-out period
			var blKey = (*v).ToAddress.String() + (*v).AccountAddress.String()
			if _, ok := w.blackList[blKey]; !ok {
				fromQuota := w.uAccess.Chain.GetAccountQuota(v.AccountAddress, snapshotHash)
				toQuota := w.uAccess.Chain.GetAccountQuota(v.ToAddress, snapshotHash)
				w.priorityToQueue.InsertNew(v, toQuota, fromQuota)
			}
		}
	}
}
