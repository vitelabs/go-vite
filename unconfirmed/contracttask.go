package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/wallet"
	"sync"
	"time"
)

type ContractTask struct {
	taskId int

	blocksPool *model.UnconfirmedBlocksPool
	pool       PoolReader
	verifier   *verifier.AccountVerifier
	genBuilder *generator.GenBuilder

	status      int
	statusMutex sync.Mutex

	stopListener chan struct{}
	breaker      chan struct{}

	isSleeping bool
	wakeup     chan struct{}

	accEvent producer.AccountStartEvent
	cworker  *ContractWorker

	getNewBlocksFunc func(index int) *model.FromItem

	log log15.Logger
}

func NewContractTask(worker *ContractWorker, index int, getNewBlocksFunc func(index int) *model.FromItem) *ContractTask {
	return &ContractTask{
		taskId:           index,
		blocksPool:       worker.uBlocksPool,
		verifier:         worker.verifier,
		genBuilder:       worker.manager.genBuilder,
		status:           Create,
		stopListener:     make(chan struct{}),
		breaker:          make(chan struct{}),
		wakeup:           make(chan struct{}),
		accEvent:         worker.accEvent,
		cworker:          worker,
		getNewBlocksFunc: getNewBlocksFunc,

		log: worker.log.New("taskid", index),
	}
}
func (task *ContractTask) WakeUp() {
	if task.isSleeping {
		task.wakeup <- struct{}{}
	}
}

func (task *ContractTask) work() {
	for {
		task.isSleeping = false
		if task.status == Stop {
			goto END
		}

		fItem := task.getNewBlocksFunc(task.taskId)
		if fItem == nil {
			goto WAIT
		}

		if task.ProcessOneQueue(fItem) {
			task.cworker.addIntoBlackList(fItem.Key, fItem.Value.Front().ToAddress)
		}

		continue

	WAIT:
		task.isSleeping = true
		select {
		case <-task.wakeup:
		case <-task.breaker:
			goto END
		default:
			break
		}
	}
END:
	task.log.Info("ContractTask send stopDispatcherListener ")
	task.stopListener <- struct{}{}
	task.log.Info("ContractTask Stop")
}

func (task *ContractTask) Start() {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Start {
		task.isSleeping = false

		go task.work()

		task.status = Start
	}
}

func (task *ContractTask) Stop() {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Stop {

		task.breaker <- struct{}{}
		close(task.breaker)

		<-task.stopListener
		close(task.stopListener)

		close(task.wakeup)

		task.status = Stop
	}
}

func (task *ContractTask) Close() error {
	task.Stop()
	return nil
}

func (task *ContractTask) Status() int {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	return task.status
}

func (task *ContractTask) ProcessOneQueue(fItem *model.FromItem) (intoBlackList bool) {
	// get db.go block from wakeup
	task.log.Info("Process the fromQueue,", "fromAddress", fItem.Key, "index", fItem.Index, "priority", fItem.Priority)

	bQueue := fItem.Value

	for i := 0; i < bQueue.Size(); i++ {
		var priorBlock *ledger.AccountBlock
		sBlock := bQueue.Dequeue()
		task.log.Info("Process to make the receiveBlock, its'sendBlock detail:", task.log.New("hash", sBlock.Hash))

		if task.pool.ExistInPool(sBlock.ToAddress, sBlock.Hash) {
			// Don't deal with it for the time being
			return true
		}

		if task.verifier.VerifyReceiveReachLimit(sBlock) {
			task.log.Info("Delete the UnconfirmedMeta: the recvBlock reach the max-limit count of existence.")
			task.blocksPool.WriteUnconfirmed(false, nil, sBlock)
			continue
		}

		genBuilder, err := task.genBuilder.PrepareVm(&task.accEvent.SnapshotHash, &block.PrevHash, &block.AccountAddress)
		if err != nil {
			task.log.Error("NewGenerator Error", err)
			return true
		}

		gen := genBuilder.Build()
		recvBlock := gen.PackUnconfirmedReceiveBlock(sBlock, &task.accEvent.SnapshotHash, &task.accEvent.Timestamp)
		genResult := gen.GenerateWithBlock(generator.SourceTypeUnconfirmed, recvBlock,
			func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
				return gen.Sign(addr, "", data)
			})
		if err != nil {
			task.log.Error("GenerateTx error ignore, ", "error", err)
		}

		if genResult.BlockGenList == nil {
			if genResult.IsRetry == true {
				return true
			} else {
				task.blocksPool.WriteUnconfirmed(false, nil, sBlock)
			}
		} else {
			if genResult.IsRetry == true {
				return true
			}

			nowTime := time.Now().Unix()
			if nowTime >= task.accEvent.Etime.Unix() {
				task.breaker <- struct{}{}
				return true
			}

			if err := task.cworker.manager.insertContractBlocksToPool(genResult.BlockGenList); err != nil {
				return true
			}

			priorBlock = genResult.BlockGenList[len(genResult.BlockGenList)-1].AccountBlock
		}
	WaitForVmDB:
		if task.verifier.VerifyUnconfirmedPriorBlockReceived(&priorBlock.Hash) == false {
			goto WaitForVmDB
		}
	}

	return false
}
