package onroad

import (
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/onroad/model"
	"github.com/vitelabs/go-vite/producer/producerevent"
)

type ContractTaskProcessor struct {
	taskId int
	worker *ContractWorker

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
		blocksPool: worker.uBlocksPool,
		status:     Create,
		isCancel:   false,
		isSleeping: false,
		log:        slog.New("tp", index),
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

		common.Go(tp.work)

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
	}
	tp.log.Info("stopped t")
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
			tp.log.Info("found cancel true")
			break
		}
		tp.log.Debug("pre popContractTask")
		task := tp.worker.popContractTask()
		tp.log.Debug("after popContractTask")

		if task != nil {
			// todo: sort addr fromAddress's height
			//fmt.Println("contractAddr", task.Addr)

			tp.log.Debug("pre processOneAddress " + task.Addr.String())
			tp.processOneAddress(task)
			tp.log.Debug("after processOneAddress " + task.Addr.String())
			continue
		}

		tp.isSleeping = true
		tp.log.Debug("start sleep t")
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

func (tp *ContractTaskProcessor) accEvent() *producerevent.AccountStartEvent {
	return tp.worker.getAccEvent()
}

func (tp *ContractTaskProcessor) processOneAddress(task *contractTask) {
	defer monitor.LogTime("onroad", "processOneAddress", time.Now())
	plog := tp.log.New("method", "processOneAddress", "worker", task.Addr)

	sBlock := tp.worker.manager.onroadBlocksPool.GetNextContractTx(task.Addr)
	if sBlock == nil {
		return
	}
	plog.Info(fmt.Sprintf("block processing: hash %v height %v accAddr:", sBlock.Hash, sBlock.Height, sBlock.AccountAddress))

	//blockList, e := tp.worker.manager.uAccess.GetOnroadBlocks(0, 1, 1, &task.Addr)
	//if e != nil {
	//	plog.Error("GetOnroadBlocks ", "e", e)
	//	return
	//}
	//if len(blockList) == 0 {
	//	return
	//}
	//sBlock := blockList[0]

	//plog.Info(fmt.Sprintf("get %v blocks, the first Hash is %v", len(blockList), sBlock.Hash), "addr", sBlock.ToAddress)

	if tp.worker.manager.checkExistInPool(sBlock.ToAddress, sBlock.Hash) {
		plog.Info("checkExistInPool true")
		// Don't deal with it for the time being
		tp.worker.addIntoBlackList(task.Addr)
		return
	}

	consensusMessage, err := tp.packConsensusMessage(sBlock)
	if err != nil {
		plog.Info("packConsensusMessage failed", "error", err)
		return
	}

	gen, err := generator.NewGenerator(tp.worker.manager.Chain(), &consensusMessage.SnapshotHash, nil, &sBlock.ToAddress)
	if err != nil {
		plog.Error("NewGenerator failed", "error", err)
		tp.worker.addIntoBlackList(task.Addr)
		return
	}

	genResult, err := gen.GenerateWithOnroad(*sBlock, consensusMessage,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			_, key, _, err := tp.worker.manager.wallet.GlobalFindAddr(addr)
			if err != nil {
				return nil, nil, err
			}
			return key.SignData(data)
		}, nil)
	if err != nil {
		plog.Error("GenerateWithOnroad failed", "error", err)
		return
	}

	if genResult.Err != nil {
		plog.Error("vm.Run error, ignore", "error", genResult.Err)
	}

	plog.Info(fmt.Sprintf("len(genResult.BlockGenList) = %v", len(genResult.BlockGenList)))
	if len(genResult.BlockGenList) > 0 {
		if err := tp.worker.manager.insertContractBlocksToPool(genResult.BlockGenList); err != nil {
			plog.Error("insertContractBlocksToPool", "error", err)
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		if genResult.IsRetry {
			plog.Error("genResult.IsRetry true")
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		for _, v := range genResult.BlockGenList {
			if v != nil && v.AccountBlock != nil {
				task.Quota -= v.AccountBlock.Quota
			}
		}

		if task.Quota > 0 {
			plog.Info(fmt.Sprintf("task.Quota remain %v", task.Quota))
			tp.worker.pushContractTask(task)
		}

	} else {
		if genResult.IsRetry {
			// retry it in next turn
			plog.Error("genResult.IsRetry true")
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		if err := tp.blocksPool.DeleteDirect(sBlock); err != nil {
			plog.Error("blocksPool.DeleteDirect", "error", err)
			tp.worker.addIntoBlackList(task.Addr)
			return
		}
	}

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

func (tp *ContractTaskProcessor) packConsensusMessage(sendBlock *ledger.AccountBlock) (*generator.ConsensusMessage, error) {
	consensusMessage := &generator.ConsensusMessage{
		SnapshotHash: tp.accEvent().SnapshotHash,
		Timestamp:    tp.accEvent().Timestamp,
		Producer:     tp.accEvent().Address,
	}
	var referredSnapshotHashList []types.Hash
	referredSnapshotHashList = append(referredSnapshotHashList, sendBlock.SnapshotHash, consensusMessage.SnapshotHash)
	_, fitestHash, err := generator.GetFittestGeneratorSnapshotHash(tp.worker.manager.chain, &sendBlock.ToAddress, referredSnapshotHashList, true)
	if err != nil {
		return nil, err
	}
	if fitestHash != nil {
		consensusMessage.SnapshotHash = *fitestHash
	}
	return consensusMessage, nil
}
