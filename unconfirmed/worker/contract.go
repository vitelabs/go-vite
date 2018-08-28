package worker

import (
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"runtime"
	"sync"
)

var POMAXPROCS = runtime.NumCPU()

type BlockQueue struct {
	items        []*ledger.AccountBlock
	lock         sync.RWMutex
	pullListener chan struct{}
}

type ContractWorker struct {
	vite    Vite
	address types.Address
	gid     string
	log     log15.Logger

	status     int

	breaker             chan struct{}
	newContractListener chan struct{}
	stopListener        chan struct{} // make sure we can sync stop the worker

	contractTasks map[int]*ContractTask
	queue         *BlockQueue

	statusMutex sync.Mutex
}

func NewContractWorker(vite Vite, address types.Address, gid string) *ContractWorker {
	return &ContractWorker{
		vite:          vite,
		address:       types.Address{},
		log:           log15.New("ContractWorker addr", address.String(), "gid", gid),
		status:        Create,
		contractTasks: make(map[int]*ContractTask),
	}
}

func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *ContractWorker) Start(timestamp timestamp.Timestamp) {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	w.status = Start
	w.log.Info("worker startWork is called")

	addressList, err := w.vite.Ledger().Ac().GetAddressListByGid(w.gid)
	if err != nil || addressList == nil || len(addressList) < 0 {
		// todo: consider directing to stop or just waiting
		w.Stop()
	} else {
		w.queue.pullListener = make(chan struct{})
		go w.FillTxMem(addressList, 4*POMAXPROCS)
		for k := 0; k < 2*POMAXPROCS; k++ {
			task := NewContractTask()
			w.contractTasks[k] = task
			go task.Start(w.queue)
		}
	}
}

func (w *ContractWorker) Stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	w.status = Stop
}

func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}

func (w *ContractWorker) FillTxMem(addressList []*types.Address, num int) {
	//turn := 0
	//for {
	//	// todo:  need to add rotation condition
	//	if v:= <-w.queue.pullListener{
	//		turn = w.GetTx(addressList, turn, num)
	//	}
	//}
}

func (w *ContractWorker) GetTx(addressList []*types.Address, index int, num int) (turn int) {
	var i int
	for i = 0; (i+index)%len(addressList) < num; i++ {
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

func NewBlockQueue() *BlockQueue {
	return nil
}

func (q *BlockQueue) PullFromMem() error {
	q.lock.Lock()
	// todo:  need to add rotation condition
	q.pullListener <- struct{}{}
	q.lock.Unlock()
	return nil
}

func (q *BlockQueue) Dequeue() *ledger.AccountBlock {
	q.lock.Lock()
	item := q.items[0]
	q.items = q.items[1:len(q.items)]
	q.lock.Unlock()
	return item
}

func (q *BlockQueue) Enqueue(block *ledger.AccountBlock) {
	q.lock.Lock()
	q.items = append(q.items, block)
	q.lock.Unlock()
}

func (q *BlockQueue) Front() *ledger.AccountBlock {
	q.lock.Lock()
	item := q.items[0]
	q.lock.Unlock()
	return item
}

func (q *BlockQueue) Size() int {
	return len(q.items)
}

func (q *BlockQueue) IsEmpty() bool {
	return len(q.items) == 0
}