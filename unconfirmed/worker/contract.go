package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed"
	"runtime"
	"sync"
)

var (
	POMAXPROCS = runtime.NumCPU()
	TASK_SIZE  = 2 * POMAXPROCS
	FETCH_SIZE = 4 * POMAXPROCS
	CACHE_SIZE = 2 * POMAXPROCS
)

type fromMap map[types.Address]*BlockQueue

type ContractWorker struct {
	vite    Vite
	address types.Address
	gid     string
	log     log15.Logger

	status          int
	pullSign        bool
	dispatcherSleep bool
	dispatcherAlarm chan struct{}

	newContractListener chan struct{}

	contractTasks   []*ContractTask
	priorityToQueue PriorityToQueue
	blackList       map[types.Hash]bool

	statusMutex sync.Mutex
}

func NewContractWorker(vite Vite, address types.Address, gid string) *ContractWorker {
	return &ContractWorker{
		vite:            vite,
		address:         types.Address{},
		log:             log15.New("ContractWorker addr", address.String(), "gid", gid),
		status:          Create,
		contractTasks:   make([]*ContractTask, POMAXPROCS),
		dispatcherAlarm: make(chan struct{}),
		blackList:       make(map[types.Hash]bool),
	}
}

// sign that the queue can pull new Tx with the gid which listener add from ledger
// and alarm the queue to pull only when dispatcher is under Sleep state
type PullSignFuncType func() bool

func (w *ContractWorker) SetSignPull() {
	if w.dispatcherSleep {
		w.dispatcherAlarm <- struct{}{}
	}
}

func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *ContractWorker) Start(args *unconfirmed.RightEventArgs) {
	w.log.Info("worker startWork is called")
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()

	// todo  1. addNewContractListener to LYD

	addressList, err := w.vite.Ledger().Ac().GetAddressListByGid(w.gid)
	if err != nil || addressList == nil || len(addressList) < 0 {
		w.Stop()
	} else {
		for _, v := range w.contractTasks {
			v.InitContractTask(w.vite, args)
			go v.Start(&w.blackList)
		}
		go w.DispatchTask(addressList)
	}

	w.status = Start
}

func (w *ContractWorker) Stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	// todo: to clear tomap

	// todo 1. rmNewContractListener to LYD
	// todo 2. Stop all task
	for _, v := range w.contractTasks {
		v.Stop()
	}

	//stop all listener
	close(w.dispatcherAlarm)
	close(w.newContractListener)

	w.status = Stop
}

func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}

func (w *ContractWorker) DispatchTask(addressList []*types.Address) {
	//todo add mutex
	turn := 0
	turn = w.GetTx(addressList, turn)
	for {
		for _, v := range w.contractTasks {
			if w.Status() == Stop {
				// todo: to clear tomap
				goto END
			}
			if w.queue.Size() < 1 {
				goto WAIT
			}

			block := w.queue.Dequeue()

			if v.Status() == Idle {
				v.subQueue <- block
			}
		}
		continue

	WAIT:
		w.dispatcherSleep = true
		<-w.dispatcherAlarm
		w.dispatcherSleep = false
		turn = w.GetTx(addressList, turn)
	}

END:
	w.log.Info("DispatchTask end")
}

func (w *ContractWorker) GetTx(addressList []*types.Address, index int) (turn int) {
	var i = 0
	for j := 0; (i+index)%len(addressList) < FETCH_SIZE; i++ {
		hashList, err := w.vite.Ledger().Ac().GetUnconfirmedTxHashs(j, 1, 1, addressList[i])
		if err != nil {
			w.log.Error("FillMemTx.GetUnconfirmedTxHashs error")
			continue
		}
		for j := range hashList {
			block, err := w.vite.Ledger().Ac().GetBlockByHash(hashList[j])
			if err != nil || block == nil {
				w.log.Error("FillMemTx.GetBlockByHash error")
				continue
			}
			if _, ok := w.blackList[*block.Hash]; !ok {
				w.queue.Enqueue(block)
			} else {
				// move the dbGet iterator of the start index to the next
				j++
			}
		}
	}
	turn = (i + index) % len(addressList)
	return turn
}

func (w *ContractWorker) FetchNew(addressList []*types.Address, turn int) (newTurn int) {
	var i int
	for i = 0; i < len(addressList); i++ {
		newTurn := (i + turn) % len(addressList)
		hashList, err := w.vite.Ledger().Ac().GetUnconfirmedTxHashs(0, 1, FETCH_SIZE, addressList[newTurn])
		if err != nil {
			w.log.Error("FillMemTx.GetUnconfirmedTxHashs error")
			continue
		}
		for _, v := range hashList {
			block, err := w.vite.Ledger().Ac().GetBlockByHash(v)
			if err != nil || block == nil {
				w.log.Error("FillMemTx.GetBlockByHash error")
				continue
			}
			if _, ok := w.blackList[*block.Hash]; !ok {
				if fm, ok := w.toMap[*block.To]; ok {
					if bq, ok := fm[*block.From]; ok {
						bq.InsertNew(block)
					} else {
						var q *BlockQueue
						q.InsertNew(block)
						fm[*block.From] = q
					}
				} else {
					var q *BlockQueue
					q.InsertNew(block)
					fMap := make(fromMap)
					fMap[*block.From] = q
					w.toMap[*block.To] = fMap
				}
			}
		}
	}
	return newTurn + 1
}
