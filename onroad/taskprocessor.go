package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vm/quota"
	"time"
)

type ContractTaskProcessor struct {
	taskId int
	worker *ContractWorker

	log log15.Logger
}

func NewContractTaskProcessor(worker *ContractWorker, index int) *ContractTaskProcessor {
	return &ContractTaskProcessor{
		taskId: index,
		worker: worker,

		log: slog.New("tp", index),
	}
}

func (tp *ContractTaskProcessor) work() {
	tp.worker.wg.Add(1)
	defer tp.worker.wg.Done()
	tp.log.Info("work() t")

	for {
		//tp.isSleeping = false
		if tp.worker.isCancel.Load() {
			tp.log.Info("found cancel true")
			break
		}
		tp.log.Debug("pre popContractTask")
		task := tp.worker.popContractTask()
		tp.log.Debug("after popContractTask")

		if task != nil {
			// tp.worker.uBlocksPool.AcquireOnroadSortedContractCache(task.Addr)
			tp.log.Debug("pre processOneAddress " + task.Addr.String())
			tp.processOneAddress(task)
			tp.log.Debug("after processOneAddress " + task.Addr.String())
			continue
		}
		//tp.isSleeping = false
		if tp.worker.isCancel.Load() {
			tp.log.Info("found cancel true")
			break
		}
		tp.worker.newBlockCond.WaitTimeout(time.Millisecond * time.Duration(tp.taskId*2+500))
	}
	tp.log.Info("work end t")
}

func (tp *ContractTaskProcessor) accEvent() *producerevent.AccountStartEvent {
	return tp.worker.getAccEvent()
}

func (tp *ContractTaskProcessor) processOneAddress(task *contractTask) {
	plog := tp.log.New("method", "processOneAddress", "worker", task.Addr)

	//sBlock := tp.worker.uBlocksPool.GetNextContractTx(task.Addr)
	sBlock, err := tp.worker.manager.getOnRoadBlockByAddr(&task.Addr)
	if err != nil {
		plog.Error(fmt.Sprintf("getOnRoadBlockByAddr failed, err:%v", err))
		return
	}
	if sBlock == nil {
		return
	}
	plog.Info(fmt.Sprintf("onroad processing: addr=%v,height=%v,hash=%v", sBlock.AccountAddress, sBlock.Height, sBlock.Hash))

	if tp.worker.manager.checkExistInPool(sBlock.ToAddress, sBlock.Hash) {
		plog.Info("checkExistInPool true")
		// Don't deal with it for the time being
		tp.worker.addIntoBlackList(task.Addr)
		return
	}
	latestSb := tp.worker.manager.Chain().GetLatestSnapshotBlock()
	if latestSb == nil {
		plog.Error("failed to get latestSnapshotBlock")
		return
	}
	var prevHash types.Hash
	prevAb, err := tp.worker.manager.Chain().GetLatestAccountBlock(&task.Addr)
	if err != nil {
		plog.Error(fmt.Sprintf("failed to get prevAccountBlock,err:%v", err))
		return
	}
	if prevAb != nil {
		prevHash = prevAb.Hash
	}
	states, err := tp.worker.manager.Chain().GetContractRandomGlobalStatus(&task.Addr, &sBlock.Hash)
	if err != nil {
		plog.Error(fmt.Sprintf("failed to get contract random global status, err:%v", err))
		return
	}
	gen, err := generator.NewGenerator2(tp.worker.manager.Chain(), task.Addr, &latestSb.Hash, &prevHash, states)
	if err != nil {
		plog.Error(fmt.Sprintf("NewGenerator failed, err:%v", err))
		tp.worker.addIntoBlackList(task.Addr)
		return
	}

	genResult, err := gen.GenerateWithOnroad(*sBlock, &task.Addr,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			_, key, _, err := tp.worker.manager.wallet.GlobalFindAddr(addr)
			if err != nil {
				return nil, nil, err
			}
			return key.SignData(data)
		}, nil)
	if err != nil {
		plog.Error(fmt.Sprintf("GenerateWithOnroad failed, err:%v", err))
		return
	}
	if genResult == nil {
		plog.Error("result of generator is nil")
		tp.worker.addIntoBlackList(task.Addr)
		return
	}
	if genResult.Err != nil {
		plog.Error(fmt.Sprintf("vm.Run error, can ignore, err:%v", genResult.Err))
	}
	if genResult.VmBlock != nil {
		if err := tp.worker.manager.insertBlockToPool(genResult.VmBlock); err != nil {
			plog.Error(fmt.Sprintf("insertContractBlocksToPool, err:%v", err))
			tp.worker.addIntoBlackList(task.Addr)
			return
		}
		if genResult.IsRetry {
			plog.Error("impossible situation: block and retry")
			tp.worker.addIntoBlackList(task.Addr)
			return
		}
	} else {
		if genResult.IsRetry {
			// retry it in next turn
			plog.Error("genResult.IsRetry true")
			// fixme: optimize the quota utilization
			q, err := tp.worker.manager.Chain().GetPledgeQuota(&task.Addr)
			if err != nil {
				plog.Error(fmt.Sprintf("failed to get pledge quota, err:%v", err))
				return
			}
			if q == nil {
				plog.Error("pledge quota is nil, failed to judge next round")
				tp.worker.addIntoBlackList(task.Addr)
				return
			}
			if canRetryDuringNextSnapshot := quota.CheckQuota(gen.GetVmDb(), task.Addr, *q); !canRetryDuringNextSnapshot {
				plog.Info("Check quota is gone to be insufficient")
				tp.worker.addIntoBlackList(task.Addr)
				return
			}
		} else {
			// no block no retry in condition that fail to create contract
			if err := tp.worker.manager.DeleteDirect(sBlock); err != nil {
				plog.Error(fmt.Sprintf("manager.DeleteDirect, err:%v", err))
				tp.worker.addIntoBlackList(task.Addr)
				return
			}
		}
	}
}
