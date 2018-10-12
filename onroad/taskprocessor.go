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
	isCancel     bool
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
		isCancel:   false,
		isSleeping: false,
		log:        worker.log.New("class", "tp", "taskid", index),
	}

	return task
}

func (tp *ContractTaskProcessor) Start() {
	tp.log.Info("Start() t", "current status", tp.status)
	tp.statusMutex.Lock()
	defer tp.statusMutex.Unlock()
	if tp.status != Start {
		tp.isCancel = false
		tp.stopListener = make(chan struct{})
		tp.breaker = make(chan struct{})
		tp.wakeup = make(chan struct{})

		tp.isSleeping = false

		go tp.work()

		tp.status = Start
	}
	tp.log.Info("end start t")
}

func (tp *ContractTaskProcessor) Stop() {
	tp.log.Info("Stop() t", "current status", tp.status)
	tp.statusMutex.Lock()
	defer tp.statusMutex.Unlock()
	if tp.status == Start {
		tp.isCancel = true
		tp.breaker <- struct{}{}
		close(tp.breaker)

		<-tp.stopListener
		close(tp.stopListener)

		close(tp.wakeup)

		tp.status = Stop
		tp.log.Info("stopped t")
	}
}

func (tp *ContractTaskProcessor) WakeUp() {
	if tp.isSleeping {
		tp.wakeup <- struct{}{}
	}
}

func (tp *ContractTaskProcessor) work() {
	tp.log.Info("work() t")
LOOP:
	for {
		tp.isSleeping = false
		if tp.isCancel {
			break
		}
		tp.log.Debug("pre popContractTask")
		task := tp.worker.popContractTask()
		tp.log.Debug("after popContractTask")
		if task != nil {
			tp.log.Debug("pre processOneAddress " + task.Addr.String())
			tp.processOneAddress(task)
			tp.log.Debug("after processOneAddress " + task.Addr.String())
			continue
		}

		tp.isSleeping = true
		tp.log.Info("start sleep t")
		select {
		case <-tp.wakeup:
			tp.log.Info("start awake t")
		case <-tp.breaker:
			tp.log.Info("worker broken t")
			break LOOP
		}
	}

	tp.log.Info("work end called t ")
	tp.stopListener <- struct{}{}
	tp.log.Info("work end t")
}

func (tp *ContractTaskProcessor) processOneAddress(task *contractTask) {

	blockList, e := tp.worker.manager.uAccess.GetOnroadBlocks(0, 1, 1, &task.Addr)
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
