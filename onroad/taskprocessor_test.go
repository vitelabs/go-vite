package onroad

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"math/rand"
	"testing"
	"time"
)

func TestRangeNil(t *testing.T) {
	var l []*testProcessor
	for k, v := range l {
		t.Log("lalallalalal", "k", k, "v", v)
	}
	l2 := make([]*testProcessor, 0)
	for k, v := range l2 {
		t.Log("babababababab", "k", k, "v", v)
	}

	t.Log("heheheh", l2[:0])
}

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
			tp.process(&task.Addr)
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

func (tp *testProcessor) process(addr *types.Address) {
	tp.log.Info("start process")
	tplog := tp.log.New("contract", addr)
	rand.Seed(time.Now().UnixNano())
	block := tp.w.getPendingOnroadBlock(addr)
	if block == nil {
		return
	}
	blog := tplog.New("onroad", fmt.Sprintf("(%v %v)", block.AccountAddress, block.Hash))
	switch rand.Intn(math.MaxUint8) % 10 {
	case 0:
		tp.w.addContractCallerToInferiorList(addr, &block.AccountAddress, RETRY)
		blog.Info("addContractCallerToInferiorList")
	case 1:
		tp.w.addContractCallerToInferiorList(addr, &block.AccountAddress, OUT)
		blog.Info("addContractCallerToInferiorList")
	case 2:
		tp.w.addContractIntoBlackList(addr)
		blog.Info("addContractIntoBlackList")
	default:
		blog.Info("deletePendingOnroadBlock")
		tp.w.deletePendingOnroadBlock(addr, block)
	}
	tplog.Info("end process")
}
