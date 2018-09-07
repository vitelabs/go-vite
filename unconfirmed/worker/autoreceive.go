package worker

import (
	"container/heap"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed"
	"math/big"
	"sync"
)

type SimpleAutoReceiveFilterPair struct {
	tti      types.TokenTypeId
	minValue big.Int
}

var (
	SimpleAutoReceiveFilters []SimpleAutoReceiveFilterPair
)

type AutoReceiveWorker struct {
	vite     Vite
	log      log15.Logger
	address  *types.Address
	dbAccess *unconfirmed.Access

	//blockQueue *BlockQueue
	priorityFromQueue *PriorityFromQueue

	status                int
	isSleeping            bool
	breaker               chan struct{}
	stopListener          chan struct{}
	newUnconfirmedTxAlarm chan struct{}

	statusMutex sync.Mutex
}

func NewAutoReceiveWorker(vite Vite, address *types.Address) *AutoReceiveWorker {
	return &AutoReceiveWorker{
		vite:       vite,
		address:    address,
		status:     Create,
		log:        log15.New("AutoReceiveWorker addr", address),
		isSleeping: false,
	}
}

func (w *AutoReceiveWorker) Start() {
	w.log.Info("Start()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {

		w.breaker = make(chan struct{}, 1)
		w.newUnconfirmedTxAlarm = make(chan struct{}, 100)
		w.stopListener = make(chan struct{})

		w.status = Start
		w.statusMutex.Unlock()

		w.dbAccess.AddCommonTxLis(w.address, func() {
			w.NewUnconfirmedTxAlarm()
		})

		go w.startWork()
	} else {
		// awake it in order to run at least once
		w.NewUnconfirmedTxAlarm()
	}
}

func (w *AutoReceiveWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {
		w.breaker <- struct{}{}
		close(w.breaker)

		w.dbAccess.RemoveCommonTxLis(w.address)
		close(w.newUnconfirmedTxAlarm)

		// make sure we can stop the worker
		<-w.stopListener
		close(w.stopListener)

		w.status = Stop
	}
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

func (w *AutoReceiveWorker) startWork() {
	w.log.Info("worker startWork is called")
	w.FetchNew()

	for {
		w.log.Debug("worker working")
		for i := 0; i < w.priorityFromQueue.Len(); i++ {
			if w.Status() == Stop {
				goto END
			}
			fItem := heap.Pop(w.priorityFromQueue).(*fromItem)
			blockQueue := fItem.value
			for j := 0; j < blockQueue.Size(); j++ {
				recvBlock := blockQueue.Dequeue()
				w.ProcessOneBlock(recvBlock)
			}
		}

		w.isSleeping = true
		w.log.Info("worker Start sleep")
		select {
		case <-w.newUnconfirmedTxAlarm:
			w.log.Info("worker Start awake")
			continue
		case <-w.breaker:
			w.log.Info("worker broken")
			break
		}
	}
END:
	w.log.Info("worker send stopDispatcherListener ")
	w.stopListener <- struct{}{}
	w.log.Info("worker end work")
}

func (w *AutoReceiveWorker) FetchNew() {
	blockList, err := w.dbAccess.GetUnconfirmedBlocks(0, 1, COMMON_FETCH_SIZE, w.address)
	if err != nil {
		w.log.Error("CommonTxWorker.FetchNew.GetUnconfirmedBlocks", "error", err)
		return
	}
	for _, v := range blockList {
		w.priorityFromQueue.InsertNew(v)
	}
}

func (w *AutoReceiveWorker) ProcessOneBlock(sendBlock *unconfirmed.AccountBlock) {
	// todo 1.ExistInPool

	//todo 2.PackReceiveBlock
	recvBlock := w.PackReceiveBlock(sendBlock)

	//todo 3.GenerateBlocks

	//todo 4.InertBlockIntoPool
	err := w.InertBlockIntoPool(recvBlock)
	if err != nil {

	}
}

func (w *AutoReceiveWorker) PackReceiveBlock(sendBlock *unconfirmed.AccountBlock) *unconfirmed.AccountBlock {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Running {
		w.status = Running
	}

	w.log.Info("PackReceiveBlock", "sendBlock",
		w.log.New("sendBlock.Hash", sendBlock.Hash), w.log.New("sendBlock.To", sendBlock.To))

	// todo pack the block with w.args, comput hash, Sign,
	block := &unconfirmed.AccountBlock{
		From:            nil,
		To:              nil,
		Height:          nil,
		Type:            0,
		PrevHash:        nil,
		FromHash:        nil,
		Amount:          nil,
		TokenId:         nil,
		CreateFee:       nil,
		Data:            nil,
		StateHash:       types.Hash{},
		SummaryHashList: nil,
		LogHash:         types.Hash{},
		SnapshotHash:    types.Hash{},
		Depth:           0,
		Quota:           0,
		Hash:            nil,
		Balance:         nil,
	}

	hash, err := block.ComputeHash()
	if err != nil {
		w.log.Error("ComputeHash Error")
		return nil
	}
	block.Hash = hash

	return block
}

func (w *AutoReceiveWorker) InertBlockIntoPool(recvBlock *unconfirmed.AccountBlock) error {
	return nil
}
