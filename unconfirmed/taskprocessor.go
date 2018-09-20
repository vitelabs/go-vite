package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/unconfirmed/model"
	"github.com/vitelabs/go-vite/verifier"
	"sync"
	"time"
)

type ContractTaskProcessor struct {
	taskId   int
	worker   *ContractWorker
	accEvent producer.AccountStartEvent

	generator  *generator.Generator
	verifier   *verifier.AccountVerifier
	blocksPool *model.UnconfirmedBlocksPool

	status      int
	statusMutex sync.Mutex

	isSleeping   bool
	wakeup       chan struct{}
	breaker      chan struct{}
	stopListener chan struct{}

	getNewBlocksFunc func(index int) *model.FromItem

	log log15.Logger
}

func NewContractTaskProcessor(worker *ContractWorker, index int, getNewBlocksFunc func(index int) *model.FromItem) *ContractTaskProcessor {
	task := &ContractTaskProcessor{
		taskId:           index,
		worker:           worker,
		accEvent:         worker.accEvent,
		verifier:         worker.verifier,
		blocksPool:       worker.uBlocksPool,
		status:           Create,
		stopListener:     make(chan struct{}),
		breaker:          make(chan struct{}),
		wakeup:           make(chan struct{}),
		getNewBlocksFunc: getNewBlocksFunc,
		log:              worker.log.New("taskid", index),
	}
	task.generator = generator.NewGenerator(worker.manager.vite.Chain(),
		worker.manager.vite.WalletManager().KeystoreManager)

	return task
}

func (task *ContractTaskProcessor) Start() {
	task.log.Info("Start()", "current status", task.status)
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	if task.status != Start {
		task.isSleeping = false

		go task.work()

		task.status = Start
	}
	task.log.Info("end start")
}

func (task *ContractTaskProcessor) Stop() {
	task.log.Info("Stop()", "current status", task.status)
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
	task.log.Info("stopped")
}

func (task *ContractTaskProcessor) WakeUp() {
	if task.isSleeping {
		task.wakeup <- struct{}{}
	}
}

func (task *ContractTaskProcessor) work() {
	task.log.Info("work()")
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
			task.worker.addIntoBlackList(fItem.Key, fItem.Value.Front().ToAddress)
		}
		continue

	WAIT:
		task.isSleeping = true
		select {
		case <-task.wakeup:
			task.log.Info("start awake")
		case <-task.breaker:
			task.log.Info("worker broken")
			break
		}
	}
END:
	task.log.Info("work end called ")
	task.stopListener <- struct{}{}
	task.log.Info("work end")
}

func (task *ContractTaskProcessor) Close() error {
	task.Stop()
	return nil
}

func (task *ContractTaskProcessor) Status() int {
	task.statusMutex.Lock()
	defer task.statusMutex.Unlock()
	return task.status
}

func (task *ContractTaskProcessor) ProcessOneQueue(fItem *model.FromItem) (intoBlackList bool) {
	// get db.go block from wakeup
	task.log.Info("Process the fromQueue,", "fromAddress", fItem.Key, "index", fItem.Index, "priority", fItem.Priority)

	bQueue := fItem.Value

	for i := 0; i < bQueue.Size(); i++ {

		sBlock := bQueue.Dequeue()
		task.log.Info("Process to make the receiveBlock, its'sendBlock detail:", task.log.New("hash", sBlock.Hash))

		if task.worker.manager.checkExistInPool(sBlock.ToAddress, sBlock.Hash) {
			// Don't deal with it for the time being
			return true
		}

		err := task.generator.PrepareVm(&task.accEvent.SnapshotHash, nil, &sBlock.ToAddress)
		if err != nil {
			task.log.Error("NewGenerator Error", err)
			return true
		}

		consensusMessage := &generator.ConsensusMessage{
			SnapshotHash: task.accEvent.SnapshotHash,
			Timestamp:    task.accEvent.Timestamp,
			Producer:     task.accEvent.Address,
		}
		genResult, err := task.generator.GenerateWithUnconfirmed(*sBlock, consensusMessage, func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			return task.generator.Sign(addr, nil, data)
		})
		if err != nil {
			task.log.Error("GenerateTx error ignore, ", "error", err)
		}

		if genResult.BlockGenList == nil {
			if genResult.IsRetry {
				return true
			}
			task.blocksPool.WriteUnconfirmed(false, nil, sBlock)
		} else {
			if genResult.IsRetry {
				return true
			}

			nowTime := time.Now().Unix()
			if nowTime >= task.accEvent.Etime.Unix() {
				task.breaker <- struct{}{}
				return true
			}

			if err := task.worker.manager.insertContractBlocksToPool(genResult.BlockGenList); err != nil {
				return true
			}
		}
	}
	return false
}
