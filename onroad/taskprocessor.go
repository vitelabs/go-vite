package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/generator"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vm/quota"
	"time"
)

const DefaultPullCount uint8 = 10

type ContractTaskProcessor struct {
	taskId int
	worker *ContractWorker

	currentTaskAddr *types.Address
	pendingMap      *callerPendingMap

	log log15.Logger
}

func NewContractTaskProcessor(worker *ContractWorker, index int) *ContractTaskProcessor {
	return &ContractTaskProcessor{
		taskId:     index,
		worker:     worker,
		pendingMap: newCallerPendingMap(),

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
		if tp.currentTaskAddr == nil || *tp.currentTaskAddr != task.Addr {
			tp.ResetPendingMap()
		}
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
	tp.ClearPendingMap()
	tp.log.Info("work end t")
}

func (tp *ContractTaskProcessor) accEvent() *producerevent.AccountStartEvent {
	return tp.worker.getAccEvent()
}

func (tp *ContractTaskProcessor) processOneAddress(task *contractTask) {
	plog := tp.log.New("method", "processOneAddress", "worker", task.Addr)

	//sBlock := tp.worker.uBlocksPool.GetNextContractTx(task.Addr)
	sBlock := tp.acquireNewOnroad(&task.Addr)
	if sBlock == nil {
		return
	}
	plog.Info(fmt.Sprintf("onroad processing: addr=%v,height=%v,hash=%v", sBlock.AccountAddress, sBlock.Height, sBlock.Hash))
	// fixme: check confirmedTimes, consider sb trigger
	if err := tp.worker.VerifierConfirmedTimes(&task.Addr, &sBlock.Hash); err != nil {
		plog.Info(fmt.Sprintf("VerifierConfirmedTimes failed, err%v:", err))
		tp.pendingMap.addCallerIntoRetryList(&sBlock.AccountAddress)
		return
	}
	randomSeedStates, err := tp.worker.manager.Chain().GetRandomGlobalStatus(&task.Addr, &sBlock.Hash)
	if err != nil {
		plog.Error(fmt.Sprintf("failed to get contract random global status, err:%v", err))
		return
	}
	addrState, err := generator.GetAddressStateForGenerator(tp.worker.manager.Chain(), &task.Addr)
	if err != nil {
		plog.Error(fmt.Sprintf("failed to get contract state for generator, err:%v", err))
		return
	}
	gen, err := generator.NewGenerator2(tp.worker.manager.Chain(), task.Addr,
		addrState.LatestSnapshotHash, addrState.LatestAccountHash,
		randomSeedStates)
	if err != nil {
		plog.Error(fmt.Sprintf("NewGenerator failed, err:%v", err))
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
		plog.Error(fmt.Sprintf("GenerateWithOnroad failed, err:%v", err))
		return
	}
	if genResult == nil {
		plog.Error("result of generator is nil")
		return
	}
	if genResult.Err != nil {
		plog.Error(fmt.Sprintf("vm.Run error, can ignore, err:%v", genResult.Err))
	}
	if genResult.VmBlock != nil {
		if err := tp.worker.manager.insertBlockToPool(genResult.VmBlock); err != nil {
			plog.Error(fmt.Sprintf("insertContractBlocksToPool, err:%v", err))
			// fixme @shan consider the err situation
			tp.pendingMap.addCallerIntoBlackList(&sBlock.AccountAddress)
			return
		}
		tp.succeedHandleOnroad(genResult.VmBlock.AccountBlock)
		if genResult.IsRetry {
			plog.Error("impossible situation: block and retry")
			tp.worker.addContractIntoBlackList(task.Addr)
			return
		}
	} else {
		if genResult.IsRetry {
			// retry it in next turn
			plog.Info("genResult.IsRetry true")
			if !types.IsBuiltinContractAddrInUseWithoutQuota(task.Addr) {
				q, err := tp.worker.manager.Chain().GetPledgeQuota(task.Addr)
				if err != nil {
					plog.Error(fmt.Sprintf("failed to get pledge quota, err:%v", err))
					return
				}
				if q == nil {
					plog.Error("pledge quota is nil, failed to judge next round")
					tp.worker.addContractIntoBlackList(task.Addr)
					return
				}
				if canRetryDuringNextSnapshot := quota.CheckQuota(gen.GetVmDb(), task.Addr, *q); !canRetryDuringNextSnapshot {
					plog.Info("Check quota is gone to be insufficient")
					tp.worker.addContractIntoBlackList(task.Addr)
					return
				}
			}
		} else {
			// no block no retry in condition that fail to create contract
			if err := tp.worker.manager.deleteDirect(sBlock); err != nil {
				plog.Error(fmt.Sprintf("manager.DeleteDirect, err:%v", err))
				tp.worker.addContractIntoBlackList(task.Addr)
				return
			}
		}
	}
}

func (tp *ContractTaskProcessor) acquireNewOnroad(contractAddr *types.Address) *ledger.AccountBlock {
	var pageNum uint8 = 1
	for tp.pendingMap.isPendingMapNotSufficient() {
		blocks, _ := tp.worker.manager.GetOnRoadBlockByAddr(contractAddr, uint64(pageNum), uint64(DefaultPullCount))
		if blocks == nil {
			break
		}
		for _, v := range blocks {
			if !tp.pendingMap.existInInferiorList(&v.AccountAddress) {
				tp.pendingMap.addPendingMap(v)
			}
		}
		pageNum++
	}
	return tp.pendingMap.getPendingOnroad()
}

func (tp *ContractTaskProcessor) succeedHandleOnroad(sendBlock *ledger.AccountBlock) {
	tp.pendingMap.deletePendingMap(&sendBlock.AccountAddress, &sendBlock.Hash)
}

func (tp *ContractTaskProcessor) ResetPendingMap() {
	tp.pendingMap = newCallerPendingMap()
}

func (tp *ContractTaskProcessor) ClearPendingMap() {
	tp.pendingMap = nil
	tp.currentTaskAddr = nil
}
