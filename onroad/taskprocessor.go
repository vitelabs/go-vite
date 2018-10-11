package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/onroad/model"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"sync"
	"time"
)

type ContractTaskProcessor struct {
	taskId   int
	worker   *ContractWorker
	accEvent producerevent.AccountStartEvent

	blocksPool *model.OnroadBlocksPool

	status      int
	statusMutex sync.Mutex

	isSleeping   bool
	wakeup       chan struct{}
	breaker      chan struct{}
	stopListener chan struct{}

	log log15.Logger
}

func NewContractTaskProcessor(worker *ContractWorker, index int) *ContractTaskProcessor {
	task := &ContractTaskProcessor{
		taskId:     index,
		worker:     worker,
		accEvent:   worker.accEvent,
		blocksPool: worker.uBlocksPool,
		status:     Create,
		log:        worker.log.New("class", "tp", "taskid", index),
	}

	return task
}

func (tp *ContractTaskProcessor) Start() {
	tp.log.Info("Start()", "current status", tp.status)
	tp.statusMutex.Lock()
	defer tp.statusMutex.Unlock()
	if tp.status != Start {
		tp.stopListener = make(chan struct{})
		tp.breaker = make(chan struct{})
		tp.wakeup = make(chan struct{})

		tp.isSleeping = false

		go tp.work()

		tp.status = Start
	}
	tp.log.Info("end start")
}

func (tp *ContractTaskProcessor) Stop() {
	tp.log.Info("Stop()", "current status", tp.status)
	tp.statusMutex.Lock()
	defer tp.statusMutex.Unlock()
	if tp.status == Start {
		tp.breaker <- struct{}{}
		close(tp.breaker)

		<-tp.stopListener
		close(tp.stopListener)

		close(tp.wakeup)

		tp.status = Stop
		tp.log.Info("stopped")
	}
}

func (tp *ContractTaskProcessor) WakeUp() {
	if tp.isSleeping {
		tp.wakeup <- struct{}{}
	}
}

func (tp *ContractTaskProcessor) work() {
	tp.log.Info("work()")
LOOP:
	for {
		if tp.isTimeout() {
			break
		}

		tp.isSleeping = false
		if tp.status == Stop {
			break
		}

		task := tp.worker.popContractTask()
		if task != nil {
			tp.log.Debug("task in work " + task.Addr.String())
			tp.processOneAddress(task)
			continue
		}

		tp.isSleeping = true
		select {
		case <-tp.wakeup:
			tp.log.Info("start awake")
		case <-tp.breaker:
			tp.log.Info("worker broken")
			break LOOP
		}
	}

	tp.log.Info("work end called ")
	tp.stopListener <- struct{}{}
	tp.log.Info("work end")
}

func (tp *ContractTaskProcessor) processOneAddress(task *contractTask) {

	blockList, e := tp.worker.manager.uAccess.GetOnroadBlocks(0, 0, 1, &task.Addr)
	if e != nil {
		tp.log.Error("GetOnroadBlocks ", "e", e)
		return
	}

	if len(blockList) == 0 {
		return
	}

	sBlock := blockList[0]

	tp.log.Info("Process to make the receiveBlock, its'sendBlock detail:", "hash", sBlock.Hash)

	if tp.worker.manager.checkExistInPool(sBlock.ToAddress, sBlock.Hash) {
		// Don't deal with it for the time being
		tp.worker.addIntoBlackList(task.Addr)
		return
	}

	gen, err := generator.NewGenerator(tp.worker.manager.Chain(),
		&tp.accEvent.SnapshotHash, nil, &sBlock.ToAddress)
	if err != nil {
		tp.log.Error("NewGenerator failed", "error", err)
		tp.worker.addIntoBlackList(task.Addr)
		return
	}

	consensusMessage := &generator.ConsensusMessage{
		SnapshotHash: tp.accEvent.SnapshotHash,
		Timestamp:    tp.accEvent.Timestamp,
		Producer:     tp.accEvent.Address,
	}

	genResult, err := gen.GenerateWithOnroad(*sBlock, consensusMessage,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			return tp.worker.manager.keystoreManager.SignData(addr, data)
		})
	if err != nil {
		tp.log.Error("GenerateWithOnroad failed", "error", err)
		return
	}

	if genResult.Err != nil {
		tp.log.Error("vm.Run error, ignore", "err", genResult.Err)
	}

	if len(genResult.BlockGenList) > 0 {
		if err := tp.worker.manager.insertContractBlocksToPool(genResult.BlockGenList); err != nil {
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		if genResult.IsRetry {
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		for _, v := range genResult.BlockGenList {
			if v != nil && v.AccountBlock != nil {
				task.Quota -= v.AccountBlock.Quota
			}
		}

		if task.Quota > 0 {
			tp.worker.pushContractTask(task)
		}

	} else {
		if genResult.IsRetry {
			// retry it in next turn
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		if err := tp.blocksPool.DeleteDirect(sBlock); err != nil {
			tp.worker.addIntoBlackList(task.Addr)
			return
		}
	}

}

func (tp *ContractTaskProcessor) isTimeout() bool {
	return time.Now().After(tp.accEvent.Etime)
}

func (tp *ContractTaskProcessor) Close() error {
	tp.Stop()
	return nil
}

func (tp *ContractTaskProcessor) Status() int {
	tp.statusMutex.Lock()
	defer tp.statusMutex.Unlock()
	return tp.status
}
