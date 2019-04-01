package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"math/rand"
	"time"
)

type testProcessor struct {
	taskId int
	w      *testContractWoker
	log    log15.Logger
}

func newTestProcessor(w *testContractWoker, i int) *testProcessor {
	return &testProcessor{
		taskId: i,
		w:      w,
		log:    testlog.New("tp", i),
	}
}

func (tp *testProcessor) work() {
	tp.log.Info("tp start work")
	tp.w.wg.Add(1)
	defer tp.w.wg.Done()
	for {
		//tp.isSleeping = false
		if tp.w.isCancel.Load() {
			break
		}
		task := tp.w.popContractTask()
		if task != nil {
			signal.Info(fmt.Sprintf("tp %v wakeup addr %v quota %v", tp.taskId, task.Addr, task.Quota))
			if tp.w.isContractInBlackList(&task.Addr) || !tp.w.addContractIntoWorkingList(&task.Addr) {
				continue
			}
			tp.w.acquireNewOnroadBlocks(&task.Addr)
			tp.process(task)
			tp.w.removeContractFromWorkingList(&task.Addr)
			continue
		}

		//tp.isSleeping = false
		if tp.w.isCancel.Load() {
			break
		}
		tp.w.newBlockCond.WaitTimeout(time.Millisecond * time.Duration(tp.taskId*2+500))
	}
	tp.log.Info("tp end work")
}

func (tp *testProcessor) process(task *contractTask) {
	tp.log.Info("process start", "contract", &task.Addr)
	rand.Seed(time.Now().UnixNano())
	sBlock := tp.w.getPendingOnroadBlock(&task.Addr)
	if sBlock == nil {
		return
	}
	blog := tp.log.New("onroad", fmt.Sprintf("(%v %v)", sBlock.AccountAddress, sBlock.Hash))

	if err := tp.w.chain.InsertIntoChain(&task.Addr, &sBlock.Hash); err != nil {
		tp.w.addContractCallerToInferiorList(&task.Addr, &sBlock.AccountAddress, RETRY)
		blog.Info("addContractCallerToInferiorList, cause InsertIntoChain failed")
		return
	}
	tp.w.deletePendingOnroadBlock(&task.Addr, sBlock)
	blog.Info("deletePendingOnroadBlock")

	if !tp.w.chain.CheckQuota(&task.Addr) {
		tp.w.addContractIntoBlackList(&task.Addr)
		blog.Info("addContractIntoBlackList, cause quota is sufficient")
		return
	}
	tp.log.Info("process end")
}
