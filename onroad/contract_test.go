package onroad

import (
	"container/heap"
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"go.uber.org/atomic"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var (
	contract1, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	contract2, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	contract3, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})
	caller1, _   = types.BytesToAddress([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	caller2, _   = types.BytesToAddress([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2})
	caller3, _   = types.BytesToAddress([]byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 3})

	ContractList = []*types.Address{&contract1, &contract2, &contract3}
	GeneralList  = []*types.Address{&caller1, &caller2, &caller3}

	testlog = log15.New("test", "onroad")
)

func TestSelectivePendingCache(t *testing.T) {
	c := newTestChainDb()
	c.work()
	w := newTestContractWoker(c)

	var breaker chan struct{}
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	//time.Sleep(3 * time.Second)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	defer wg.Done()

	common.Go(w.work)

	time.AfterFunc(3*time.Minute, func() {
		breaker <- struct{}{}
	})
	for {
		select {
		case <-ticker.C:
			c.work()
			t.Log("new blocks add to db")
		case <-breaker:
			t.Log("break to stop worker")
			w.stop()
			t.Log("worker be stopped")
		}
	}
	wg.Wait()
}

type testChainDb struct {
	db      map[types.Address][]*ledger.AccountBlock
	blockFn func(address *types.Address)
}

func newTestChainDb() *testChainDb {
	return &testChainDb{
		db: make(map[types.Address][]*ledger.AccountBlock),
	}
}

func (c *testChainDb) work() {
	blocks := c.newTestOnroad()
	testlog.Info("chain work, get new onroad", "len", len(blocks))
	for _, v := range blocks {
		if l, ok := c.db[v.ToAddress]; ok && l != nil {
			l = append(l, v)
		} else {
			nl := make([]*ledger.AccountBlock, 0)
			nl = append(nl, v)
			c.db[v.ToAddress] = nl
		}
		if c.blockFn != nil {
			c.blockFn(&v.ToAddress)
		}
	}
}

func (c *testChainDb) setNewSignal(f func(address *types.Address)) {
	c.blockFn = f
}

func (w *testChainDb) newTestOnroad() []*ledger.AccountBlock {
	var blen int = 20
	rand.Seed(time.Now().UnixNano())
	blockList := make([]*ledger.AccountBlock, 0)
	for i := 0; i < blen; i++ {
		u8height := rand.Intn(math.MaxUint8 + 1)
		b := &ledger.AccountBlock{
			BlockType:      ledger.BlockTypeSendCall,
			Height:         uint64(u8height),
			Hash:           types.DataHash([]byte{uint8(u8height)}),
			AccountAddress: *GeneralList[rand.Intn(len(GeneralList))],
			ToAddress:      *ContractList[rand.Intn(len(ContractList))],
		}
		//fmt.Printf("height:%v,hash:%v,caller:%v, contract:%v\n", b.Height, b.Hash, b.AccountAddress, b.ToAddress)
		blockList = append(blockList, b)
	}
	return blockList
}

func (w *testChainDb) getTestOnRoadList(addr *types.Address, pageNum, countPerPage uint8) ([]*ledger.AccountBlock, error) {
	blocks := make([]*ledger.AccountBlock, 0)
	var index uint8 = 0
	for _, l := range w.db {
		for _, v := range l {
			if index >= pageNum*countPerPage && index < countPerPage*(pageNum+1) {
				blocks = append(blocks, v)
			}
			index++
		}
	}
	return blocks, nil
}

func (w *testChainDb) deleteTestOnroad(cAddr *types.Address, sendHash *types.Hash) bool {
	if l, ok := w.db[*cAddr]; ok && l != nil {
		for k, v := range l {
			if v.Hash == *sendHash {
				if k >= len(l)-1 {
					l = l[0:k]
				} else {
					l = append(l[0:k], l[k+1:]...)
				}
				return true
			}
		}
	}
	return false
}

type testContractWoker struct {
	chain          *testChainDb
	testProcessors []*testProcessor

	isCancel    *atomic.Bool
	status      int
	statusMutex sync.Mutex

	newBlockCond *common.TimeoutCond
	wg           sync.WaitGroup

	blackList      map[types.Address]bool
	blackListMutex sync.RWMutex

	workingAddrList      map[types.Address]bool
	workingAddrListMutex sync.RWMutex

	testPendingCache map[types.Address]*callerPendingMap

	contractTaskPQueue contractTaskPQueue
	ctpMutex           sync.RWMutex
}

func newTestContractWoker(c *testChainDb) *testContractWoker {
	w := &testContractWoker{
		chain:            c,
		testPendingCache: make(map[types.Address]*callerPendingMap),
		blackList:        make(map[types.Address]bool),
		workingAddrList:  make(map[types.Address]bool),

		status:       Create,
		isCancel:     atomic.NewBool(false),
		newBlockCond: common.NewTimeoutCond(),
	}
	processors := make([]*testProcessor, 2)
	for i, _ := range processors {
		processors[i] = newTestProcessor(w, i)
	}
	w.testProcessors = processors
	return w
}

func (w *testContractWoker) stop() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status == Start {

		w.isCancel.Store(true)
		w.newBlockCond.Broadcast()

		w.clearSelectiveBlocksCache()
		w.clearContractBlackList()
		w.clearWorkingAddrList()

		slog.Info("stop all task")
		w.wg.Wait()
		slog.Info("end stop all task")
		w.status = Stop
	}
}

func (w *testContractWoker) work() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {
		w.isCancel.Store(false)

		w.initContractTask()

		w.chain.setNewSignal(func(addr *types.Address) {
			if w.isContractInBlackList(addr) {
				return
			}
			rand.Seed(time.Now().UnixNano())
			c := &contractTask{
				Addr:  *ContractList[rand.Intn(100)%len(ContractList)],
				Quota: uint64(rand.Intn(100)),
			}
			w.pushContractTask(c)

			w.WakeupOneTp()
		})

		for _, v := range w.testProcessors {
			common.Go(v.work)
		}
		w.status = Start
	} else {
		w.WakeupAllTps()
	}
}
func (w *testContractWoker) WakeupOneTp() {
	testlog.Info("WakeupOneTp")
	w.newBlockCond.Signal()
}

func (w *testContractWoker) WakeupAllTps() {
	testlog.Info("WakeupAllTPs")
	w.newBlockCond.Broadcast()
}

func (w *testContractWoker) initContractTask() {
	var quota = uint64(0)
	w.contractTaskPQueue = make([]*contractTask, len(ContractList))
	for k, v := range ContractList {
		w.contractTaskPQueue[k] = &contractTask{
			Addr:  *v,
			Index: k,
			Quota: quota,
		}
		quota++
	}
	heap.Init(&w.contractTaskPQueue)

	fmt.Println("taskpqueue init")
	for _, t := range w.contractTaskPQueue {
		fmt.Println("addr", t.Addr, "q", t.Quota, "index", t.Index)
	}
}

func (w *testContractWoker) pushContractTask(t *contractTask) {
	w.ctpMutex.Lock()
	defer w.ctpMutex.Unlock()
	// be careful duplicates
	for _, v := range w.contractTaskPQueue {
		if v.Addr == t.Addr {
			testlog.Info(fmt.Sprintf("heap fix, pre-idx=%v, addr=%v, quota=%v", v.Index, t.Addr, t.Quota))
			v.Quota = t.Quota
			heap.Fix(&w.contractTaskPQueue, v.Index)
			return
		}
	}
	heap.Push(&w.contractTaskPQueue, t)
	testlog.Info("pushContractTask new", "addr", t.Addr)
}

func (w *testContractWoker) popContractTask() *contractTask {
	w.ctpMutex.Lock()
	defer w.ctpMutex.Unlock()
	if w.contractTaskPQueue.Len() > 0 {
		return heap.Pop(&w.contractTaskPQueue).(*contractTask)
	}
	return nil
}

func (w *testContractWoker) clearSelectiveBlocksCache() {
	w.testPendingCache = make(map[types.Address]*callerPendingMap)
}

func (w *testContractWoker) getPendingOnroadBlock(contractAddr *types.Address) *ledger.AccountBlock {
	if pendingCache, ok := w.testPendingCache[*contractAddr]; ok && pendingCache != nil {
		return pendingCache.getPendingOnroad()
	}
	return nil
}

func (w *testContractWoker) deletePendingOnroadBlock(contractAddr *types.Address, sendBlock *ledger.AccountBlock) {
	w.chain.deleteTestOnroad(contractAddr, &sendBlock.Hash)
	if pendingMap, ok := w.testPendingCache[*contractAddr]; ok && pendingMap != nil {
		success := pendingMap.deletePendingMap(sendBlock.AccountAddress, &sendBlock.Hash)
		if success && pendingMap.isInferiorStateRetry(sendBlock.AccountAddress) {
			pendingMap.removeFromInferiorList(sendBlock.AccountAddress)
		}
	}
}

func (w *testContractWoker) acquireNewOnroadBlocks(contractAddr *types.Address) {
	if pendingMap, ok := w.testPendingCache[*contractAddr]; ok && pendingMap != nil {
		var pageNum uint8 = 0
		for pendingMap.isPendingMapNotSufficient() {
			pageNum++
			blocks, _ := w.chain.getTestOnRoadList(contractAddr, pageNum-1, DefaultPullCount)
			if blocks == nil {
				break
			}
			for _, v := range blocks {
				if !pendingMap.existInInferiorList(v.AccountAddress) {
					pendingMap.addPendingMap(v)
				}
			}
		}
	} else {
		callerMap := newCallerPendingMap()
		blocks, _ := w.chain.getTestOnRoadList(contractAddr, 0, DefaultPullCount)
		if blocks == nil {
			return
		}
		for _, v := range blocks {
			callerMap.addPendingMap(v)
		}
		w.testPendingCache[*contractAddr] = callerMap
	}
	testlog.Info(fmt.Sprintf("pendingCache len %v", len(w.testPendingCache[*contractAddr].pmap)))
}

func (w *testContractWoker) addContractCallerToInferiorList(contract, caller *types.Address, state inferiorState) {
	if pendingCache, ok := w.testPendingCache[*contract]; ok && pendingCache != nil {
		pendingCache.addCallerIntoInferiorList(*caller, state)
	}
}

func (w *testContractWoker) isContractCallerInferiorRetry(contract, caller *types.Address) bool {
	if pendingCache, ok := w.testPendingCache[*contract]; ok && pendingCache != nil {
		return pendingCache.isInferiorStateRetry(*caller)
	}
	return false
}

func (w *testContractWoker) clearWorkingAddrList() {
	w.workingAddrListMutex.Lock()
	defer w.workingAddrListMutex.Unlock()
	w.workingAddrList = make(map[types.Address]bool)
}

func (w *testContractWoker) isContractCallerInferiorOut(contract, caller *types.Address) bool {
	if pendingCache, ok := w.testPendingCache[*contract]; ok && pendingCache != nil {
		return pendingCache.isInferiorStateOut(*caller)
	}
	return false
}

func (w *testContractWoker) addContractIntoWorkingList(addr *types.Address) bool {
	w.workingAddrListMutex.Lock()
	defer w.workingAddrListMutex.Unlock()
	result, ok := w.workingAddrList[*addr]
	if result && ok {
		return false
	}
	w.workingAddrList[*addr] = true
	return true
}

func (w *testContractWoker) removeContractFromWorkingList(addr *types.Address) {
	w.workingAddrListMutex.RLock()
	defer w.workingAddrListMutex.RUnlock()
	w.workingAddrList[*addr] = false
}

func (w *testContractWoker) clearContractBlackList() {
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList = make(map[types.Address]bool)
}

func (w *testContractWoker) addContractIntoBlackList(addr *types.Address) {
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList[*addr] = true
	delete(w.testPendingCache, *addr)
}

func (w *testContractWoker) isContractInBlackList(addr *types.Address) bool {
	w.blackListMutex.RLock()
	defer w.blackListMutex.RUnlock()
	_, ok := w.blackList[*addr]
	if ok {
		fmt.Printf("isContractInBlackList", "addr", addr, "in", ok)
	}
	return ok
}
