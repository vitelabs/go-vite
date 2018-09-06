package worker

import (
	"container/heap"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed"
	"runtime"
	"sync"
)

var (
	POMAXPROCS = uint64(runtime.NumCPU())
	TASK_SIZE  = uint64(2 * POMAXPROCS)
	FETCH_SIZE = uint64(2 * POMAXPROCS)
	CACHE_SIZE = uint64(2 * POMAXPROCS)
)

type ContractWorker struct {
	vite     Vite
	log      log15.Logger
	dbAccess *unconfirmed.UnconfirmedAccess

	gid                 []byte
	addresses           *types.Address
	contractAddressList []*types.Address

	status                 int
	dispatcherSleep        bool
	dispatcherAlarm        chan struct{}
	breaker                chan struct{}
	stopDispatcherListener chan struct{}

	contractTasks   []*ContractTask
	priorityToQueue *PriorityToQueue
	blackList       map[string]bool // map[(toAddress+fromAddress).String]

	statusMutex sync.Mutex
}

func NewContractWorker(vite Vite, dbAccess *unconfirmed.UnconfirmedAccess, gid []byte, address *types.Address, addressList []*types.Address) *ContractWorker {
	return &ContractWorker{
		vite:                   vite,
		dbAccess:               dbAccess,
		gid:                    gid,
		addresses:              address,
		contractAddressList:    addressList,
		status:                 Create,
		dispatcherSleep:        false,
		dispatcherAlarm:        make(chan struct{}, 1),
		breaker:                make(chan struct{}, 1),
		stopDispatcherListener: make(chan struct{}, 1),
		contractTasks:          make([]*ContractTask, POMAXPROCS),
		blackList:              make(map[string]bool),
		log:                    log15.New("ContractWorker addr", address.String(), "gid", gid),
	}
}

func (w *ContractWorker) Start(event *unconfirmed.RightEvent) {
	w.log.Info("worker startWork is called")
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()

	// todo  1. addNewContractListener to LYD

	for _, v := range w.contractTasks {
		v.InitContractTask(w.vite, w.dbAccess, event)
		go v.Start(&w.blackList)
	}
	go w.DispatchTask()

	w.status = Start
}

func (w *ContractWorker) Stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {

		w.breaker <- struct{}{}
		// todo: to clear tomap

		// 1. CallBack
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

func (w *ContractWorker) DispatchTask() {
	//todo add mutex
	w.FetchNew()
	for {
		for i := 0; i < w.priorityToQueue.Len(); i++ {
			tItem := heap.Pop(w.priorityToQueue).(*toItem)
			priorityFromQueue := tItem.value
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

				fItem := heap.Pop(priorityFromQueue).(*fromItem)
				w.contractTasks[freeTaskIndex].subQueue <- fItem
			}
		}

		w.dispatcherSleep = true

		select {
		case <-w.breaker:
			goto END
		case <-w.dispatcherAlarm:
			w.dispatcherSleep = false
			w.FetchNew()
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

func (w *ContractWorker) FetchNew() {
	for i := 0; i < len(w.contractAddressList); i++ {
		blockList, err := w.dbAccess.GetUnconfirmedBlocks(0, 1, FETCH_SIZE, w.contractAddressList[i])
		if err != nil {
			w.log.Error("ContractWorker.FetchNew.GetUnconfirmedBlocks", "error", err)
			continue
		}
		for _, v := range blockList {
			// when a to-from pair  was added into blackList,
			// the other block which under the same to-from pair won't fetch any more during the same block-out period
			var blKey = (*v).To.String() + (*v).From.String()
			if _, ok := w.blackList[blKey]; !ok {
				w.priorityToQueue.InsertNew(v)
			}
		}
	}
}
