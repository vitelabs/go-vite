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
			if result := tp.w.addContractIntoWorkingList(&task.Addr); !result {
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
		//tp.log.Info(fmt.Sprintf("tp %v start to sleep", tp.taskId))
		tp.w.newBlockCond.WaitTimeout(time.Millisecond * time.Duration(tp.taskId*2+500))
		//tp.log.Info(fmt.Sprintf("tp %v start to wakeup", tp.taskId))
	}
	tp.log.Info("tp end work")
}

func (tp *testProcessor) process(task *contractTask) {
	tp.log.Info("process start")
	tplog := tp.log.New("contract", &task.Addr)
	tplog.Info(fmt.Sprintf("current quota %v", task.Quota))
	rand.Seed(time.Now().UnixNano())
	sBlock := tp.w.getPendingOnroadBlock(&task.Addr)
	if sBlock == nil {
		return
	}
	blog := tplog.New("onroad", fmt.Sprintf("(%v %v)", sBlock.AccountAddress, sBlock.Hash))

	if err := tp.w.chain.InsertIntoChain(&task.Addr, &sBlock.Hash); err != nil {
		tp.w.addContractCallerToInferiorList(&task.Addr, &sBlock.AccountAddress, RETRY)
		blog.Info("addContractCallerToInferiorList, cause InsertIntoChain failed")
		return
	}
	tp.w.deletePendingOnroadBlock(&task.Addr, sBlock)
	blog.Info("deletePendingOnroadBlock")

	if !tp.w.chain.CheckQuota(&task.Addr) {
		tp.w.addContractIntoBlackList(&task.Addr)
		blog.Info("addContractCallerToInferiorList, cause quota is sufficient")
		return
	}
	tp.log.Info("process end")
}
