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

const DefaultPullCount uint8 = 10

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
			if tp.worker.isContractInBlackList(task.Addr) || !tp.worker.addContractIntoWorkingList(task.Addr) {
				continue
			}
			tp.worker.acquireNewOnroadBlocks(&task.Addr)
			tp.log.Debug("pre processOneAddress " + task.Addr.String())
			tp.processOneAddress(task)
			tp.worker.removeContractFromWorkingList(task.Addr)
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
	plog := tp.log.New("processAddr", task.Addr)

	sBlock := tp.worker.getPendingOnroadBlock(&task.Addr)
	if sBlock == nil {
		return
	}
	blog := plog.New("onroad", sBlock.Hash, "caller", sBlock.AccountAddress)

	//fixme checkReceivedSuccess
	// fixme: check confirmedTimes, consider sb trigger
	if err := tp.worker.VerifyConfirmedTimes(&task.Addr, &sBlock.Hash); err != nil {
		blog.Info(fmt.Sprintf("VerifyConfirmedTimes failed, err%v:", err))
		tp.worker.addContractCallerToInferiorList(&task.Addr, &sBlock.AccountAddress, RETRY)
		return
	}

	randomSeedStates, err := tp.worker.manager.Chain().GetRandomGlobalStatus(&task.Addr, &sBlock.Hash)
	if err != nil {
		blog.Error(fmt.Sprintf("failed to get contract random global status, err:%v", err))
		return
	}
	if randomSeedStates != nil {
		blog.Info(fmt.Sprintf("seed=%v sb(%v,%v)", randomSeedStates.Seed, randomSeedStates.SnapshotBlock.Height, randomSeedStates.SnapshotBlock.Hash))
	}

	addrState, err := generator.GetAddressStateForGenerator(tp.worker.manager.Chain(), &task.Addr)
	if err != nil || addrState == nil {
		blog.Error(fmt.Sprintf("failed to get contract state for generator, err:%v", err))
		return
	}
	plog.Info(fmt.Sprintf("contract-prev: hash=%v height=%v", addrState.LatestAccountHash, addrState.LatestAccountHeight))

	gen, err := generator.NewGenerator2(tp.worker.manager.Chain(), task.Addr,
		addrState.LatestSnapshotHash, addrState.LatestAccountHash,
		randomSeedStates)
	if err != nil {
		blog.Error(fmt.Sprintf("NewGenerator failed, err:%v", err))
		return
	}
	genResult, err := gen.GenerateWithOnroad(sBlock, &task.Addr,
		func(addr types.Address, data []byte) (signedData, pubkey []byte, err error) {
			_, key, _, err := tp.worker.manager.wallet.GlobalFindAddr(addr)
			if err != nil {
				return nil, nil, err
			}
			return key.SignData(data)
		}, nil)
	if err != nil {
		blog.Error(fmt.Sprintf("GenerateWithOnroad failed, err:%v", err))
		return
	}
	if genResult == nil {
		blog.Info("result of generator is nil")
		return
	}
	if genResult.Err != nil {
		blog.Info(fmt.Sprintf("vm.Run error, can ignore, err:%v", genResult.Err))
	}
	if genResult.VmBlock != nil {
		if err := tp.worker.manager.insertBlockToPool(genResult.VmBlock); err != nil {
			blog.Error(fmt.Sprintf("insertContractBlocksToPool failed, err:%v", err))
			tp.worker.addContractCallerToInferiorList(&task.Addr, &sBlock.AccountAddress, OUT)
			return
		}
		// succeed in handling a onroad block, if it's a inferior-caller,than set it free.
		tp.worker.deletePendingOnroadBlock(&task.Addr, genResult.VmBlock.AccountBlock)

		if genResult.IsRetry {
			blog.Info("impossible situation: vmBlock and vmRetry")
			tp.worker.addContractIntoBlackList(task.Addr)
			return
		}
	} else {
		if genResult.IsRetry {
			// vmRetry it in next turn
			blog.Info("genResult.IsRetry true")
			if !types.IsBuiltinContractAddrInUseWithoutQuota(task.Addr) {
				q, err := tp.worker.manager.Chain().GetPledgeQuota(task.Addr)
				if err != nil {
					blog.Error(fmt.Sprintf("failed to get pledge quota, err:%v", err))
					return
				}
				if q == nil {
					blog.Info("pledge quota is nil, to judge it in next round")
					tp.worker.addContractIntoBlackList(task.Addr)
					return
				}
				if canRetryDuringNextSnapshot := quota.CheckQuota(gen.GetVmDb(), task.Addr, *q); !canRetryDuringNextSnapshot {
					blog.Info("Check quota is gone to be insufficient",
						"quota", fmt.Sprintf("(u:%v c:%v t:%v sb:%v)", q.Used(), q.Current(), q.Total(), addrState.LatestSnapshotHash))
					tp.worker.addContractIntoBlackList(task.Addr)
					return
				}
			}
		} else {
			// no vmBlock no vmRetry in condition that fail to create contract
			if err := tp.worker.manager.deleteDirect(sBlock); err != nil {
				blog.Error(fmt.Sprintf("manager.DeleteDirect, err:%v", err))
				tp.worker.addContractIntoBlackList(task.Addr)
				return
			}
		}
	}
}
