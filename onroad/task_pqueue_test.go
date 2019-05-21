package onroad

import (
	"container/heap"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"math/rand"
	"testing"
	"time"
)

var addrStr = [5]string{
	"vite_0f12dabefaff2e01dd0ab3bc2cbbe1c375ad253c8744bf95c9",
	"vite_5c84c3bc3c554060d17ae7c66924b379a2896ba7b3d4dd318f",
	"vite_2152892514fc6c1be2cfecdf8051250fc74fe2327bf1804f87",
	"vite_a7c3ad4ca89c1f60911f9a680946df45e5c375afe953ee24cc",
	"vite_02eb678e9122c5de50e2c1f321dd9b833e72d13f227bcd9a6b",
}

var addrStrPush = [5]string{
	"vite_9073d51752a268198217bba9ff7699551ee3a886d46640971d",
	"vite_873e3f90db60d55de0fb2a93e1e3918756b8acbaf29435028e",
	"vite_d436d6b0b72d8c4283e40c960201a344fd8122b096125922ae",
	"vite_b94b19a6272677776c7f886827bf51b8e0686da2563de8172a",
	"vite_686bb5c3479ff36ddce34ed6c7e5dc35f0a0fe99a9bbd1d124",
}

var q = [5]uint64{
	1, 2, 3, 4, 5,
}

var quotaPush = [5]uint64{
	6, 7, 8, 9, 0,
}

func TestContractTaskPQueue(t *testing.T) {

	addrPush := make([]types.Address, len(addrStr))
	for i, value := range addrStrPush {
		a, _ := types.HexToAddress(value)
		addrPush[i] = a
	}

	addr := make([]types.Address, len(addrStr))
	for i, value := range addrStr {
		a, _ := types.HexToAddress(value)
		addr[i] = a
	}

	ct := make(contractTaskPQueue, len(addr))
	for key, value := range addr {
		ct[key] = &contractTask{
			Addr:  value,
			Index: key,
			Quota: q[key],
		}
	}

	for _, t := range ct {
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}

	fmt.Println("\n")

	heap.Init(&ct)

	heap.Push(&ct, &contractTask{
		Addr:  addrPush[0],
		Quota: 1000,
	})
	for _, t := range ct {
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}

	fmt.Println("\n")

	heap.Push(&ct, &contractTask{
		Addr:  addrPush[1],
		Quota: 1200,
	})
	for _, t := range ct {
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}

	fmt.Println("\n")

	for {
		if ct.Len() == 0 {
			break
		}
		v := heap.Pop(&ct)
		t := v.(*contractTask)
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}

}

func TestContractTaskPQueue_PushDuplicates(t *testing.T) {
	addrPool := make([]types.Address, len(addrStr))
	for i, value := range addrStr {
		a, _ := types.HexToAddress(value)
		addrPool[i] = a
	}
	w := &testPQWorker{
		contractTaskPQueue: make(contractTaskPQueue, len(addrPool)),
	}
	for key, value := range addrPool {
		w.contractTaskPQueue[key] = &contractTask{
			Addr:  value,
			Index: key,
			Quota: q[key],
		}
	}
	fmt.Println("\ninit before")
	for _, t := range w.contractTaskPQueue {
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}
	heap.Init(&w.contractTaskPQueue)
	fmt.Println("\ninit after")
	for _, t := range w.contractTaskPQueue {
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}

	fmt.Println("\nstart push duplicates")
	var round = 15
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < round; i++ {
		new := &contractTask{
			Addr:  addrPool[rand.Intn(round)%len(addrPool)],
			Quota: uint64(rand.Intn(100)),
		}
		w.pushContractTask(new)
	}
	for _, t := range w.contractTaskPQueue {
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}
	fmt.Println()
	for {
		if w.contractTaskPQueue.Len() == 0 {
			break
		}
		v := w.popContractTask()
		fmt.Println("addr", v.Addr, "q", v.Quota, "index", v.Index)
	}
}

type testPQWorker struct {
	contractTaskPQueue contractTaskPQueue
}

func (w *testPQWorker) popContractTask() *contractTask {
	if w.contractTaskPQueue.Len() > 0 {
		return heap.Pop(&w.contractTaskPQueue).(*contractTask)
	}
	return nil
}

func (w *testPQWorker) pushContractTask(t *contractTask) {
	for _, v := range w.contractTaskPQueue {
		if v.Addr == t.Addr {
			v.Quota = t.Quota
			var preIdx = v.Index
			heap.Fix(&w.contractTaskPQueue, v.Index)
			fmt.Printf("heap fix, pre-idx=%v, new-idx=%v, addr=%v, quota=%v\n", preIdx, v.Index, v.Addr, v.Quota)
			return
		}
	}
	heap.Push(&w.contractTaskPQueue, t)
}
