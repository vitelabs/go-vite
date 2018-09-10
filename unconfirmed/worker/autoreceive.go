package worker

import (
	"container/heap"
	"container/list"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"math/big"
	"sync"
)

type SimpleAutoReceiveFilterPair struct {
	tti      types.TokenTypeId
	minValue big.Int
}

type AutoReceiveWorker struct {
	log     log15.Logger
	address *types.Address
	uAccess *model.UAccess

	//blockQueue *BlockQueue
	priorityFromQueue *model.PriorityFromQueue

	status                int
	isSleeping            bool
	breaker               chan struct{}
	stopListener          chan struct{}
	newUnconfirmedTxAlarm chan struct{}

	filters        map[types.TokenTypeId]big.Int
	currentElement list.Element

	statusMutex sync.Mutex
}

func NewAutoReceiveWorker(address *types.Address, filters map[types.TokenTypeId]big.Int) *AutoReceiveWorker {
	return &AutoReceiveWorker{
		address:    address,
		status:     Create,
		log:        log15.New("AutoReceiveWorker addr", address),
		isSleeping: false,
		filters:    filters,
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

		w.uAccess.AddCommonTxLis(w.address, func() {
			w.NewUnconfirmedTxAlarm()
		})

		go w.startWork()
	} else {
		// awake it in order to run at least once
		w.NewUnconfirmedTxAlarm()
	}
}

func (w *AutoReceiveWorker) Restart() {
	w.log.Info("Restart()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
}

func (w *AutoReceiveWorker) SetAutoReceiveFilter(filters map[types.TokenTypeId]big.Int) {
	w.filters = filters
	w.Restart()
}

func (w *AutoReceiveWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {
		w.breaker <- struct{}{}
		close(w.breaker)

		w.uAccess.RemoveCommonTxLis(w.address)
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
	w.FetchNewOne()

	for {
		w.log.Debug("worker working")
		for i := 0; i < w.priorityFromQueue.Len(); i++ {
			if w.Status() == Stop {
				goto END
			}
			fItem := heap.Pop(w.priorityFromQueue).(*model.FromItem)
			blockQueue := fItem.Value
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

func (w *AutoReceiveWorker) FetchNewOne() {
	blockList, err := w.uAccess.GetUnconfirmedBlocks(0, 1, COMMON_FETCH_SIZE, w.address)
	if err != nil {
		w.log.Error("CommonTxWorker.FetchNew.GetUnconfirmedBlocks", "error", err)
		return
	}
	for _, v := range blockList {
		w.priorityFromQueue.InsertNew(v)
	}
}

func (w *AutoReceiveWorker) ProcessOneBlock(sendBlock *ledger.AccountBlock) {
	// todo 1.ExistInPool

	//todo 2.PackReceiveBlock
	recvBlock := w.PackReceiveBlock(sendBlock)

	//todo 3.GenerateBlocks

	//todo 4.InertBlockIntoPool
	err := w.InertBlockIntoPool(recvBlock)
	if err != nil {

	}
}

func (w *AutoReceiveWorker) PackReceiveBlock(sendBlock *ledger.AccountBlock) *ledger.AccountBlock {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Running {
		w.status = Running
	}

	w.log.Info("PackReceiveBlock", "sendBlock",
		w.log.New("sendBlock.Hash", sendBlock.Hash), w.log.New("sendBlock.To", sendBlock.ToAddress))

	// todo pack the block with w.args, comput hash, Sign,
	block := &ledger.AccountBlock{
		Meta:              nil,
		BlockType:         0,
		Hash:              nil,
		Height:            nil,
		PrevHash:          nil,
		AccountAddress:    nil,
		PublicKey:         nil,
		ToAddress:         nil,
		FromBlockHash:     nil,
		Amount:            nil,
		TokenId:           nil,
		QuotaFee:          nil,
		ContractFee:       nil,
		SnapshotHash:      nil,
		Data:              "",
		Timestamp:         0,
		StateHash:         nil,
		LogHash:           nil,
		Nonce:             nil,
		SendBlockHashList: nil,
		Signature:         nil,
	}

	hash, err := block.ComputeHash()
	if err != nil {
		w.log.Error("ComputeHash Error")
		return nil
	}
	block.Hash = hash

	return block
}

func (w *AutoReceiveWorker) InertBlockIntoPool(recvBlock *ledger.AccountBlock) error {
	return nil
}
