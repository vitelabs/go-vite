package onroad

import (
	"fmt"
	"sync/atomic"
	"time"

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

	log log15.Logger
}

func NewContractTaskProcessor(worker *ContractWorker, index int) *ContractTaskProcessor {
	task := &ContractTaskProcessor{
		taskId:     index,
		worker:     worker,
		blocksPool: worker.uBlocksPool,

		log: slog.New("tp", index),
	}

	return task
}

func (tp *ContractTaskProcessor) work() {
	tp.worker.wg.Add(1)
	defer tp.worker.wg.Done()
	tp.log.Info("work() t")

	for {
		//tp.isSleeping = false
		if atomic.LoadUint32(&tp.worker.isCancel) == 1 {
			tp.log.Info("found cancel true")
			break
		}
		tp.log.Debug("pre popContractTask")
		task := tp.worker.popContractTask()
		tp.log.Debug("after popContractTask")

		if task != nil {
			result := tp.worker.addIntoWorkingList(task.Addr)
			if !result {
				continue
			}
			tp.worker.uBlocksPool.AcquireOnroadSortedContractCache(task.Addr)
			tp.log.Debug("pre processOneAddress " + task.Addr.String())
			tp.processOneAddress(task)
			tp.worker.removeFromWorkingList(task.Addr)
			tp.log.Debug("after processOneAddress " + task.Addr.String())
			continue
		}
		//tp.isSleeping = false
		if atomic.LoadUint32(&tp.worker.isCancel) == 1 {
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
	defer monitor.LogTime("onroad", "processOneAddress", time.Now())
	plog := tp.log.New("processAddr", task.Addr, "remainQuota", task.Quota, "sbHash", tp.worker.currentSnapshotHash)

	sBlock := tp.worker.uBlocksPool.GetNextContractTx(task.Addr)
	if sBlock == nil {
		return
	}
	plog.Info(fmt.Sprintf("block processing: accAddr=%v,height=%v,hash=%v,remainQuota=%d", sBlock.AccountAddress, sBlock.Height, sBlock.Hash, task.Quota))
	if latestBlock, _ := tp.worker.manager.Chain().GetLatestAccountBlock(&task.Addr); latestBlock != nil {
		plog.Info(fmt.Sprintf("contract-prev: hash=%v,height=%v,sbHash:%v", latestBlock.Hash, latestBlock.Height, latestBlock.SnapshotHash))
	}

	blog := plog.New("onroad", sBlock.Hash, "caller", sBlock.AccountAddress)
	if tp.worker.manager.checkExistInPool(sBlock.ToAddress, sBlock.Hash) {
		blog.Info("checkExistInPool true")
		// Don't deal with it for the time being
		tp.worker.addIntoBlackList(task.Addr)
		return
	}

	if tp.worker.manager.chain.IsSuccessReceived(&sBlock.ToAddress, &sBlock.Hash) {
		plog.Error("had receive for account block", "addr", sBlock.AccountAddress, "hash", sBlock.Hash)
		return
	}

	receiveErrHeightList, err := tp.worker.manager.chain.GetReceiveBlockHeights(&sBlock.Hash)
	if err != nil {
		blog.Error(fmt.Sprintf("GetReceiveBlockHeights failed,err %v", err))
		return
	}
	if len(receiveErrHeightList) > 0 {
		highestHeight := receiveErrHeightList[len(receiveErrHeightList)-1]
		blog.Info(fmt.Sprintf("receiveErrBlock highest height %v", highestHeight))
		receiveErrBlock, hErr := tp.worker.manager.chain.GetAccountBlockByHeight(&sBlock.ToAddress, highestHeight)
		if hErr != nil || receiveErrBlock == nil {
			blog.Error(fmt.Sprintf("GetAccountBlockByHeight failed, err:%v", hErr))
			return
		}
		if task.Quota < receiveErrBlock.Quota {
			blog.Info("contractAddr still out of quota")
			tp.worker.addIntoBlackList(task.Addr)
			return
		}
	}

	consensusMessage, err := tp.packConsensusMessage(sBlock)
	if err != nil {
		blog.Info("packConsensusMessage failed", "error", err)
		return
	}

	blog.Info(fmt.Sprintf("fittestSbHash:%v", consensusMessage.SnapshotHash))

	gen, err := generator.NewGenerator(tp.worker.manager.Chain(), &consensusMessage.SnapshotHash, nil, &sBlock.ToAddress)
	if err != nil {
		blog.Error("NewGenerator failed", "error", err)
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
		blog.Error("GenerateWithOnroad failed", "error", err)
		return
	}

	if genResult.Err != nil {
		blog.Error("vm.Run error, ignore", "error", genResult.Err)
	}

	blog.Info(fmt.Sprintf("len(genResult.BlockGenList) = %v", len(genResult.BlockGenList)))
	if len(genResult.BlockGenList) > 0 {
		if err := tp.worker.manager.insertContractBlocksToPool(genResult.BlockGenList); err != nil {
			blog.Error("insertContractBlocksToPool", "error", err)
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		if genResult.IsRetry {
			blog.Info("genResult.IsRetry true")
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		for _, v := range genResult.BlockGenList {
			if v != nil && v.AccountBlock != nil {
				if task.Quota <= v.AccountBlock.Quota {
					plog.Error(fmt.Sprintf("addr %v out of quota expected during snapshotTime %v.", task.Addr, tp.worker.currentSnapshotHash))
					tp.worker.addIntoBlackList(task.Addr)
					return
				}
				task.Quota -= v.AccountBlock.Quota
			}
		}

		if task.Quota > 0 {
			blog.Info(fmt.Sprintf("task.Quota remain %v", task.Quota))
			tp.worker.pushContractTask(task)
		}

	} else {
		if genResult.IsRetry {
			// retry it in next turn
			blog.Info("genResult.IsRetry true")
			tp.worker.addIntoBlackList(task.Addr)
			return
		}

		if err := tp.blocksPool.DeleteDirect(sBlock); err != nil {
			blog.Error("blocksPool.DeleteDirect", "error", err)
			tp.worker.addIntoBlackList(task.Addr)
			return
		}
	}

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
