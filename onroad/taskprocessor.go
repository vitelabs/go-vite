package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/log15"
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
		task := tp.worker.popContractTask()
		if task != nil {
			signalLog.Info(fmt.Sprintf("tp %v wakeup, pop addr %v quota %v", tp.taskId, task.Addr, task.Quota))
			if tp.worker.isContractInBlackList(task.Addr) || !tp.worker.addContractIntoWorkingList(task.Addr) {
				continue
			}
			canContinue := tp.processOneAddress(task)
			tp.worker.removeContractFromWorkingList(task.Addr)
			if canContinue {
				task.Quota = tp.worker.GetPledgeQuota(task.Addr)
				tp.worker.pushContractTask(task)
			}
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

func (tp *ContractTaskProcessor) processOneAddress(task *contractTask) (canContinue bool) {
	tp.log.Info("process", "contract", &task.Addr)

	sBlock := tp.worker.acquireOnRoadBlocks(task.Addr)
	if sBlock == nil {
		return false
	}
	blog := tp.log.New("l", sBlock.Hash, "caller", sBlock.AccountAddress, "contract", task.Addr)

	// fixme checkReceivedSuccess
	// fixme: check confirmedTimes, consider sb trigger
	if err := tp.worker.VerifyConfirmedTimes(&task.Addr, &sBlock.Hash); err != nil {
		blog.Info(fmt.Sprintf("VerifyConfirmedTimes failed, err:%v", err))
		// tp.worker.addContractCallerToInferiorList(&task.Addr, &sBlock.AccountAddress, RETRY)
		return true
	}

	addrState, err := generator.GetAddressStateForGenerator(tp.worker.manager.Chain(), &task.Addr)
	if err != nil || addrState == nil {
		blog.Error(fmt.Sprintf("failed to get contract state for generator, err:%v", err))
		return true
	}
	tp.log.Info(fmt.Sprintf("contract-prev: addr=%v hash=%v height=%v", task.Addr, addrState.LatestAccountHash, addrState.LatestAccountHeight))

	gen, err := generator.NewGenerator(tp.worker.manager.Chain(), tp.worker.manager.Consensus(), task.Addr, addrState.LatestSnapshotHash, addrState.LatestAccountHash)
	if err != nil {
		blog.Error(fmt.Sprintf("NewGenerator failed, err:%v", err))
		return true
	}
	genResult, err := gen.GenerateWithOnRoad(sBlock, &tp.worker.address,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			_, key, _, err := tp.worker.manager.wallet.GlobalFindAddr(addr)
			if err != nil {
				return nil, nil, err
			}
			return key.SignData(data)
		}, nil)
	if err != nil {
		blog.Error(fmt.Sprintf("GenerateWithOnRoad failed, err:%v", err))
		return true
	}
	if genResult == nil {
		blog.Info("result of generator is nil")
		return true
	}
	if genResult.Err != nil {
		blog.Info(fmt.Sprintf("vm.Run error, can ignore, err:%v", genResult.Err))
	}
	if genResult.VmBlock != nil {
		if err := tp.worker.manager.insertBlockToPool(genResult.VmBlock); err != nil {
			blog.Error(fmt.Sprintf("insertContractBlocksToPool failed, err:%v", err))
			tp.worker.addContractCallerToInferiorList(task.Addr, sBlock.AccountAddress, OUT)
			return true
		}

		if genResult.IsRetry {
			blog.Info("impossible situation: vmBlock and vmRetry")
			tp.worker.addContractIntoBlackList(task.Addr)
			return false
		}
	} else {
		if genResult.IsRetry {
			// vmRetry it in next turn
			blog.Info("genResult.IsRetry true")
			if !types.IsBuiltinContractAddrInUseWithoutQuota(task.Addr) {
				q, err := tp.worker.manager.Chain().GetPledgeQuota(task.Addr)
				if err != nil {
					blog.Error(fmt.Sprintf("failed to get pledge quota, err:%v", err))
					return true
				}
				if q == nil {
					blog.Info("pledge quota is nil, to judge it in next round")
					tp.worker.addContractIntoBlackList(task.Addr)
					return false
				}
				if canRetryDuringNextSnapshot := quota.CheckQuota(gen.GetVmDb(), *q); !canRetryDuringNextSnapshot {
					blog.Info("Check quota is gone to be insufficient",
						"quota", fmt.Sprintf("(u:%v c:%v t:%v sb:%v)", q.Used(), q.Current(), q.Total(), addrState.LatestSnapshotHash))
					tp.worker.addContractIntoBlackList(task.Addr)
					return false
				}
			}
		} else {
			// no vmBlock no vmRetry in condition that fail to create contract
			blog.Info(fmt.Sprintf("manager.DeleteDirect, contract %v hash %v", task.Addr, sBlock.Hash))
			tp.worker.manager.deleteDirect(sBlock)
			tp.worker.addContractIntoBlackList(task.Addr)
			return false
		}
	}
	return true
}
