package onroad

import (
	"container/heap"
	"fmt"
	"github.com/pkg/errors"
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

	testDefaultQuota = uint64(20)

	testlog = log15.New("test", "onroad")
)

func TestSelectivePendingCache(t *testing.T) {
	chain := NewTestChainDb()
	worker := newTestContractWoker(chain)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	defer wg.Done()

	common.Go(worker.Work)

	chain.Start()

	time.AfterFunc(1*time.Minute, func() {
		chain.Stop()
		//worker.stop()
		var remain = 0
		for _, v := range chain.db {
			remain += len(v)
		}
		fmt.Printf("hhaghagahgaaghghghghghghgh %v", remain)
	})

	wg.Wait()
}

type testChainDb struct {
	db map[types.Address][]*ledger.AccountBlock
	//dbMutex sync.RWMutex

	quotaMap   map[types.Address]uint64
	quotaMutex sync.RWMutex

	blockFn func(address *types.Address)

	breaker chan struct{}
}

func NewTestChainDb() *testChainDb {
	chain := &testChainDb{
		db:       make(map[types.Address][]*ledger.AccountBlock),
		quotaMap: make(map[types.Address]uint64, len(ContractList)),
	}
	for _, v := range ContractList {
		chain.quotaMap[*v] = testDefaultQuota
	}
	return chain
}

func (c *testChainDb) Start() {
	testlog.Info("writeOnroad start")

	c.writeOnroad(false)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.writeOnroad(true)
		case <-c.breaker:
			testlog.Info("chain work stop")
			break
		}
	}
	testlog.Info("writeOnroad end")
}

func (c *testChainDb) Stop() {
	c.breaker <- struct{}{}
}

func (c *testChainDb) SetNewSignal(f func(address *types.Address)) {
	c.blockFn = f
}

func (c *testChainDb) InsertIntoChain(cAddr *types.Address, sHash *types.Hash) error {
	return c.deleteOnroad(cAddr, sHash)
}

func (c *testChainDb) CheckQuota(addr *types.Address) bool {
	if c.quotaMap[*addr] <= 5 {
		return false
	}
	return true
}

func (w *testChainDb) GetOnRoadBlockByAddr(addr *types.Address, pageNum, pageCount uint8) ([]*ledger.AccountBlock, error) {
	blocks := make([]*ledger.AccountBlock, 0)
	var index uint8 = 0
	for _, l := range w.db {
		for _, v := range l {
			if index >= pageCount*pageNum {
				if index >= pageCount*(pageNum+1) {
					return blocks, nil
				}
				blocks = append(blocks, v)
			}
			index++
		}
	}
	return blocks, nil
}

func (c *testChainDb) writeOnroad(quotaRandom bool) {
	rand.Seed(time.Now().UnixNano())
	blocks := make([]*ledger.AccountBlock, 0)
	for i := 0; i < int(DefaultPullCount); i++ {
		u8height := rand.Intn(math.MaxUint8 + 1)
		b := &ledger.AccountBlock{
			BlockType:      ledger.BlockTypeSendCall,
			Height:         uint64(u8height),
			Hash:           types.DataHash([]byte{uint8(u8height)}),
			AccountAddress: *GeneralList[rand.Intn(len(GeneralList))],
			ToAddress:      *ContractList[rand.Intn(len(ContractList))],
		}
		fmt.Printf("height:%v,hash:%v,caller:%v, contract:%v\n", b.Height, b.Hash, b.AccountAddress, b.ToAddress)
		blocks = append(blocks, b)
	}
	fmt.Printf("data to prepared, start Work, len %v\n\n", len(blocks))

	var vCount = 0
	for _, v := range blocks {
		vCount++
		if l, ok := c.db[v.ToAddress]; ok && l != nil {
			l = append(l, v)
			c.db[v.ToAddress] = l
		} else {
			nl := make([]*ledger.AccountBlock, 0)
			nl = append(nl, v)
			c.db[v.ToAddress] = nl
		}
		if c.blockFn != nil {
			fmt.Printf("Signal to %v hash %v\n", v.ToAddress, v.Hash)
			c.blockFn(&v.ToAddress)
		}
	}

	for k, _ := range c.db {
		if !quotaRandom {
			c.quotaMap[k] = testDefaultQuota
		} else {
			//c.modifyQuotaMap_Add(&k)
		}
	}

}

func (c *testChainDb) deleteOnroad(cAddr *types.Address, sHash *types.Hash) error {
	if l, ok := c.db[*cAddr]; ok && l != nil {
		for k, v := range l {
			if v.Hash == *sHash {
				if !c.modifyQuotaMap_Sub(cAddr) {
					return errors.New("cause modifyQuotaMap_Sub failed")
				}
				if k >= len(l)-1 {
					l = l[0:k]
				} else {
					l = append(l[0:k], l[k+1:]...)
				}
				break
			}
		}
	}
	return nil
}

func (c *testChainDb) getQuotaMap(addr *types.Address) uint64 {
	c.quotaMutex.Lock()
	defer c.quotaMutex.Unlock()
	quotaCurrent := c.quotaMap[*addr]
	return quotaCurrent
}

func (c *testChainDb) modifyQuotaMap_Add(addr *types.Address) {
	c.quotaMutex.RLock()
	defer c.quotaMutex.RUnlock()
	rand.Seed(time.Now().UnixNano())
	quotaCurrent := c.quotaMap[*addr] + uint64(rand.Intn(10)+1)
	c.quotaMap[*addr] = quotaCurrent
}

func (c *testChainDb) modifyQuotaMap_Sub(addr *types.Address) bool {
	c.quotaMutex.RLock()
	defer c.quotaMutex.RUnlock()
	rand.Seed(time.Now().UnixNano())
	qUsed := uint64(rand.Intn(10) + 1)
	if c.quotaMap[*addr] < qUsed {
		return false
	} else {
		c.quotaMap[*addr] = c.quotaMap[*addr] - qUsed
		return true
	}
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

func (w *testContractWoker) Work() {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {
		w.isCancel.Store(false)

		w.initContractTask()

		w.chain.SetNewSignal(func(addr *types.Address) {
			if w.isContractInBlackList(addr) {
				return
			}
			rand.Seed(time.Now().UnixNano())
			c := &contractTask{
				Addr:  *ContractList[rand.Intn(100)%len(ContractList)],
				Quota: uint64(rand.Intn(10)),
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
	w.contractTaskPQueue = make([]*contractTask, len(ContractList))
	for k, v := range ContractList {
		w.contractTaskPQueue[k] = &contractTask{
			Addr:  *v,
			Index: k,
			Quota: w.chain.getQuotaMap(v),
		}
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
	if pendingMap, ok := w.testPendingCache[*contractAddr]; ok && pendingMap != nil {
		success := pendingMap.deletePendingMap(sendBlock.AccountAddress, &sendBlock.Hash)
		if success {
			if pendingMap.isInferiorStateRetry(sendBlock.AccountAddress) {
				fmt.Printf("delete before len %v", len(pendingMap.InferiorList))
				pendingMap.removeFromInferiorList(sendBlock.AccountAddress)
				fmt.Printf("delete after len %v", len(pendingMap.InferiorList))
			}
		}
	}
}

func (w *testContractWoker) acquireNewOnroadBlocks(contractAddr *types.Address) {
	if pendingMap, ok := w.testPendingCache[*contractAddr]; ok && pendingMap != nil {
		var pageNum uint8 = 0
		for pendingMap.isPendingMapNotSufficient() {
			pageNum++
			blocks, _ := w.chain.GetOnRoadBlockByAddr(contractAddr, pageNum-1, DefaultPullCount)
			if len(blocks) <= 0 {
				return
			}
			testlog.Info("GetOnRoadBlockByAddr blocks", "len", len(blocks))
			for _, v := range blocks {
				if !pendingMap.existInInferiorList(v.AccountAddress) {
					pendingMap.addPendingMap(v)
				}
			}
		}
	} else {
		callerMap := newCallerPendingMap()
		blocks, _ := w.chain.GetOnRoadBlockByAddr(contractAddr, 0, DefaultPullCount)
		if len(blocks) <= 0 {
			return
		}
		testlog.Info("GetOnRoadBlockByAddr blocks", "len", len(blocks))
		for _, v := range blocks {
			callerMap.addPendingMap(v)
		}
		w.testPendingCache[*contractAddr] = callerMap
	}
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
		testlog.Info("isContractInBlackList", "addr", addr, "in", ok)

	}
	return ok
}
