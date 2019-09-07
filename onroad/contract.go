package onroad

import (
	"container/heap"
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"
	"strconv"
	"sync"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"go.uber.org/atomic"
)

var signalLog = slog.New("signal", "contract")

// ContractWorker managers the task processor, it also maintains the blacklist and queues with priority for callers.
type ContractWorker struct {
	address types.Address

	manager *Manager

	gid                 types.Gid
	contractAddressList []types.Address

	status      int
	statusMutex sync.Mutex

	isCancel *atomic.Bool

	newBlockCond *common.TimeoutCond
	wg           sync.WaitGroup

	contractTaskProcessors []*ContractTaskProcessor

	blackList      map[types.Address]bool
	blackListMutex sync.RWMutex

	workingAddrList      map[types.Address]bool
	workingAddrListMutex sync.RWMutex

	contractTaskPQueue contractTaskPQueue
	ctpMutex           sync.RWMutex

	selectivePendingCache *sync.Map //map[types.Address]*callerPendingMap

	log log15.Logger
}

// NewContractWorker creates a ContractWorker.
func NewContractWorker(manager *Manager) *ContractWorker {
	worker := &ContractWorker{
		manager: manager,

		status:       create,
		isCancel:     atomic.NewBool(false),
		newBlockCond: common.NewTimeoutCond(),

		blackList:             make(map[types.Address]bool),
		workingAddrList:       make(map[types.Address]bool),
		selectivePendingCache: &sync.Map{},

		log: slog.New("worker", "contract"),
	}
	processors := make([]*ContractTaskProcessor, ContractTaskProcessorSize)
	for i, _ := range processors {
		processors[i] = NewContractTaskProcessor(worker, i)
	}
	worker.contractTaskProcessors = processors

	return worker
}

// Start is to start the ContractWorker's work, it listens to the event triggered by other module.
func (w *ContractWorker) Start(accEvent producerevent.AccountStartEvent) {
	w.gid = accEvent.Gid
	w.address = accEvent.Address

	w.log = slog.New("worker", "contract", "gid", accEvent.Gid)

	log := w.log.New("method", "start")
	log.Info("Start() status=" + strconv.Itoa(w.status))
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != start {
		w.isCancel.Store(false)

		// 1. get gid`s all contract address if error happened return immediately
		addressList, err := w.manager.Chain().GetContractList(w.gid)
		if err != nil {
			w.log.Error("GetAddrListByGid ", "err", err)
			return
		}
		if len(addressList) == 0 {
			w.log.Info("newContractWorker addressList nil")
			return
		}
		w.contractAddressList = addressList
		log.Info(fmt.Sprintf("get addresslist len %v", len(addressList)))

		// 2. get getAndSortAllAddrQuota it is a heavy operation so we call it only once in start
		w.getAndSortAllAddrQuota()
		log.Info(fmt.Sprintf("getAndSortAllAddrQuota len %v", len(w.contractTaskPQueue)))

		// 3. register listening events, including addContractLis and addSnapshotEventLis
		w.manager.addContractLis(w.gid, func(address types.Address) {
			if w.isContractInBlackList(address) {
				return
			}
			q := w.GetPledgeQuota(address)
			c := &contractTask{
				Addr:  address,
				Quota: q,
			}

			if !w.isCancel.Load() {
				w.pushContractTask(c)
				signalLog.Info(fmt.Sprintf("signal to %v and wake it up", address))
				w.wakeupOneTp()
			}
		})

		log.Info("addSnapshotEventLis", "gid", w.gid, "event", "snapshotEvent")
		w.manager.addSnapshotEventLis(w.gid, func(latestHeight uint64) {
			for _, addr := range w.contractAddressList {
				if w.isContractInBlackList(addr) {
					continue
				}
				count := w.releaseContractCallers(addr, RETRY)
				if count <= 0 {
					continue
				}

				q := w.GetPledgeQuota(addr)
				c := &contractTask{
					Addr:  addr,
					Quota: q,
				}
				if !w.isCancel.Load() {
					w.pushContractTask(c)
					signalLog.Info(fmt.Sprintf("snapshot line changed, signal to %v to release RETRY callers, len %v", addr, count), "snapshot", latestHeight, "event", "snapshotEvent")
					w.wakeupOneTp()
				}
			}
		})

		log.Info("start all tp")
		for _, v := range w.contractTaskProcessors {
			common.Go(v.work)
		}
		log.Info("end start all tp")

		w.status = start
	} else {
		// awake it in order to run at least once
		w.wakeupAllTps()
	}
	w.log.Info("end start")
}

// Stop is to stop the ContractWorker and free up memory.
func (w *ContractWorker) Stop() {
	log := w.log.New("method", "stop")
	log.Info("Stop() status=" + strconv.Itoa(w.status))
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status == start {
		w.manager.removeContractLis(w.gid)

		log.Info("removeSnapshotEventLis", "gid", w.gid, "event", "snapshotEvent")
		w.manager.removeSnapshotEventLis(w.gid)

		w.isCancel.Store(true)
		w.newBlockCond.Broadcast()

		log.Info("stop all task")
		w.wg.Wait()
		log.Info("end stop all task")

		w.clearContractBlackList()
		w.clearWorkingAddrList()
		w.clearSelectiveBlocksCache()

		w.status = stop
	}
	w.log.Info("stopped")
}

// Close is to stop the ContractWorker.
func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}

// Status returns the status of a ContractWorker.
func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *ContractWorker) getAndSortAllAddrQuota() {
	quotas := w.GetPledgeQuotas(w.contractAddressList)

	w.contractTaskPQueue = make([]*contractTask, len(quotas))
	i := 0
	for addr, quota := range quotas {
		task := &contractTask{
			Addr:  addr,
			Index: i,
			Quota: quota,
		}
		w.contractTaskPQueue[i] = task
		i++
	}

	heap.Init(&w.contractTaskPQueue)
}

func (w *ContractWorker) wakeupOneTp() {
	w.newBlockCond.Signal()
}

func (w *ContractWorker) wakeupAllTps() {
	w.log.Info("wakeupAllTps")
	w.newBlockCond.Broadcast()
}

func (w *ContractWorker) pushContractTask(t *contractTask) {
	w.ctpMutex.Lock()
	defer w.ctpMutex.Unlock()
	for _, v := range w.contractTaskPQueue {
		if v.Addr == t.Addr {
			v.Quota = t.Quota
			heap.Fix(&w.contractTaskPQueue, v.Index)
			return
		}
	}
	heap.Push(&w.contractTaskPQueue, t)
}

func (w *ContractWorker) popContractTask() *contractTask {
	w.ctpMutex.Lock()
	defer w.ctpMutex.Unlock()
	if w.contractTaskPQueue.Len() > 0 {
		return heap.Pop(&w.contractTaskPQueue).(*contractTask)
	}
	return nil
}

func (w *ContractWorker) clearWorkingAddrList() {
	w.workingAddrListMutex.Lock()
	defer w.workingAddrListMutex.Unlock()
	w.workingAddrList = make(map[types.Address]bool)
}

func (w *ContractWorker) clearSelectiveBlocksCache() {
	w.selectivePendingCache = &sync.Map{}
}

// Don't deal with it for this around of blocks-generating period
func (w *ContractWorker) addContractIntoWorkingList(addr types.Address) bool {
	w.workingAddrListMutex.Lock()
	defer w.workingAddrListMutex.Unlock()
	result, ok := w.workingAddrList[addr]
	if result && ok {
		return false
	}
	w.workingAddrList[addr] = true
	return true
}

func (w *ContractWorker) removeContractFromWorkingList(addr types.Address) {
	w.workingAddrListMutex.Lock()
	defer w.workingAddrListMutex.Unlock()
	w.workingAddrList[addr] = false
}

func (w *ContractWorker) clearContractBlackList() {
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList = make(map[types.Address]bool)
}

// Don't deal with it for this around of blocks-generating period
func (w *ContractWorker) addContractIntoBlackList(addr types.Address) {
	w.blackListMutex.Lock()
	w.blackList[addr] = true
	w.blackListMutex.Unlock()

	w.selectivePendingCache.Delete(addr)
}

func (w *ContractWorker) isContractInBlackList(addr types.Address) bool {
	w.blackListMutex.RLock()
	defer w.blackListMutex.RUnlock()
	_, ok := w.blackList[addr]
	if ok {
		w.log.Info("isContractInBlackList", "addr", addr, "in", ok)
	}
	return ok
}

func (w *ContractWorker) acquireOnRoadBlocks(contractAddr types.Address) *ledger.AccountBlock {
	addNewCount := 0
	revertHappened := false

	value, ok := w.selectivePendingCache.LoadOrStore(contractAddr, newCallerPendingMap())
	p := value.(*callerPendingMap)
	if !ok {
		blocks, _ := w.manager.GetAllCallersFrontOnRoad(w.gid, contractAddr)
		if len(blocks) <= 0 {
			return nil
		}
		for _, v := range blocks {
			if isExist := p.addPendingMap(v); !isExist {
				addNewCount++
			}
		}
	} else {
		sendBlock := p.getOnePending()
		if sendBlock != nil {
			isFront, err := w.manager.IsFrontOnRoadOfCaller(w.gid, contractAddr, sendBlock.AccountAddress, sendBlock.Hash)
			if isFront && err == nil {
				return sendBlock
			}
			if err != nil {
				w.log.Error("IsFrontOnRoadOfCaller fail, cause err is "+err.Error(), "caller", sendBlock.AccountAddress, "contract", contractAddr)
			}
			revertHappened = true
			p.clearPendingMap()
		}

		blocks, _ := w.manager.GetAllCallersFrontOnRoad(w.gid, contractAddr)
		for _, v := range blocks {
			if p.existInInferiorList(v.AccountAddress) {
				continue
			}
			if isExist := p.addPendingMap(v); !isExist {
				addNewCount++
			}
		}
	}

	w.log.Info(fmt.Sprintf("acquire new %v, current %v revert %v", addNewCount, p.Len(), revertHappened), "contract", contractAddr, "waitSBCallerLen", p.lenOfCallersByState(RETRY))
	return p.getOnePending()
}

func (w *ContractWorker) addContractCallerToInferiorList(contract, caller types.Address, state inferiorState) {
	value, ok := w.selectivePendingCache.Load(contract)
	if ok && value != nil {
		value.(*callerPendingMap).addIntoInferiorList(caller, state)
	}
}

func (w *ContractWorker) releaseContractCallers(contract types.Address, state inferiorState) int {
	value, ok := w.selectivePendingCache.Load(contract)
	var count int
	if ok && value != nil {
		count = value.(*callerPendingMap).releaseCallerByState(state)
	}
	return count
}

// GetPledgeQuota returns the available quota the contract can use at current.
func (w *ContractWorker) GetPledgeQuota(addr types.Address) uint64 {
	if types.IsBuiltinContractAddrInUseWithoutQuota(addr) {
		return math.MaxUint64
	}
	_, quota, err := w.manager.Chain().GetPledgeQuota(addr)
	if err != nil {
		w.log.Error("GetPledgeQuota err", "error", err)
	}
	if quota == nil {
		return 0
	}
	return quota.Current()
}

// GetPledgeQuotas returns the available quota the contract can use at current in batch.
func (w *ContractWorker) GetPledgeQuotas(beneficialList []types.Address) map[types.Address]uint64 {
	quotas := make(map[types.Address]uint64)
	if w.gid == types.DELEGATE_GID {
		commonContractAddressList := make([]types.Address, 0, len(beneficialList))
		for _, addr := range beneficialList {
			if types.IsBuiltinContractAddrInUseWithoutQuota(addr) {
				quotas[addr] = math.MaxUint64
			} else {
				commonContractAddressList = append(commonContractAddressList, addr)
			}
		}
		commonQuotas, err := w.manager.Chain().GetPledgeQuotas(commonContractAddressList)
		if err != nil {
			w.log.Error("GetPledgeQuotas err", "error", err)
		} else {
			for k, v := range commonQuotas {
				if v == nil {
					continue
				}
				quotas[k] = v.Current()
			}
		}
	} else {
		quotasMap, qRrr := w.manager.Chain().GetPledgeQuotas(beneficialList)
		if qRrr != nil {
			w.log.Error("GetPledgeQuotas err", "error", qRrr)
		} else {
			for k, v := range quotasMap {
				if v == nil {
					continue
				}
				quotas[k] = v.Current()
			}
		}
	}
	return quotas
}

func (w *ContractWorker) verifyConfirmedTimes(contractAddr *types.Address, fromHash *types.Hash, sbHeight uint64) error {
	meta, err := w.manager.Chain().GetContractMeta(*contractAddr)
	if err != nil {
		return err
	}
	if meta == nil {
		return errors.New("contract meta is nil")
	}
	if meta.SendConfirmedTimes == 0 {
		return nil
	}
	sendConfirmedTimes, err := w.manager.Chain().GetConfirmedTimes(*fromHash)
	if err != nil {
		return err
	}
	if sendConfirmedTimes < uint64(meta.SendConfirmedTimes) {
		return errors.New("sendBlock confirmedTimes is not ready")
	}

	if fork.IsSeedFork(sbHeight) && meta.SeedConfirmedTimes > 0 {
		isSeedCountOk, err := w.manager.Chain().IsSeedConfirmedNTimes(*fromHash, uint64(meta.SeedConfirmedTimes))
		if err != nil {
			return err
		}
		if !isSeedCountOk {
			return errors.New("sendBlock seed confirmedTimes is not ready")
		}
	}
	return nil
}
