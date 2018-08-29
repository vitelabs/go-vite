package worker

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"runtime"
	"sync"
)

var (
	POMAXPROCS = runtime.NumCPU()
	TASK_SIZE  = 2 * POMAXPROCS
	FETCH_SIZE = 4 * POMAXPROCS
	CACHE_SIZE = 2 * POMAXPROCS
)

type ContractWorker struct {
	vite    Vite
	address types.Address
	gid     string
	log     log15.Logger

	status int

	newContractListener       chan struct{}
	newUnconfirmedGidListener chan struct{}

	contractTasks []*ContractTask
	queue         *BlockQueue

	statusMutex sync.Mutex
}

func NewContractWorker(vite Vite, address types.Address, gid string) *ContractWorker {
	return &ContractWorker{
		vite:          vite,
		address:       types.Address{},
		log:           log15.New("ContractWorker addr", address.String(), "gid", gid),
		status:        Create,
		contractTasks: make([]*ContractTask, POMAXPROCS),
	}
}

func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *ContractWorker) Start(timestamp uint64) {
	w.log.Info("worker startWork is called")
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()

	// todo  1. addNewContractListener to LYD
	w.newUnconfirmedGidListener = make(chan struct{}, 100)

	addressList, err := w.vite.Ledger().Ac().GetAddressListByGid(w.gid)
	if err != nil || addressList == nil || len(addressList) < 0 {
		w.Stop()
	} else {
		w.newUnconfirmedGidListener <- struct{}{}
		for _, v := range w.contractTasks {
			go v.Start(timestamp)
		}
		go w.DispatchTask(addressList)
	}

	w.status = Start
}

func (w *ContractWorker) Stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	w.queue.Clear()

	// todo 1. rmNewContractListener to LYD
	// todo 2. Stop all task
	for _, v := range w.contractTasks {
		v.Stop()
	}

	//stop all listener
	close(w.newUnconfirmedGidListener)
	close(w.newContractListener)

	w.status = Stop
}

func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}

func (w *ContractWorker) DispatchTask(addressList []*types.Address) {
	turn := 0
LOOP:
	for {
		for _, v := range w.contractTasks {
			if w.Status() == Stop {
				w.queue.Clear()
				break LOOP
			}
			if w.queue.Size() < 1 {
				goto WAIT
			}

			block := w.queue.Dequeue()

			if v.Status() == Idle {
				v.subQueue <- block
			}
		}

	WAIT:
		<-w.newUnconfirmedGidListener
		if w.queue.Size() < FETCH_SIZE {
			turn = w.GetTx(addressList, turn)
		} else {
			continue
		}
	}
}

func (w *ContractWorker) GetTx(addressList []*types.Address, index int) (turn int) {
	var i int
	for i = 0; (i+index)%len(addressList) < FETCH_SIZE; i++ {
		hashList, err := w.vite.Ledger().Ac().GetUnconfirmedTxHashs(0, 1, 1, addressList[i])
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
			w.queue.Enqueue(block)
		}
	}
	turn = (i + index) % len(addressList)
	return turn
}
