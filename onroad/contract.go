package onroad

import (
	"container/heap"
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"go.uber.org/atomic"
	"strconv"
	"sync"
)

type ContractWorker struct {
	manager *Manager

	gid                 types.Gid
	address             types.Address
	accEvent            producerevent.AccountStartEvent
	currentSnapshotHash types.Hash

	status      int
	statusMutex sync.Mutex

	isCancel *atomic.Bool

	newBlockCond *common.TimeoutCond
	wg           sync.WaitGroup

	contractTaskProcessors []*ContractTaskProcessor
	contractAddressList    []types.Address

	contractTaskPQueue contractTaskPQueue
	ctpMutex           sync.RWMutex

	blackList      map[types.Address]bool
	blackListMutex sync.RWMutex

	log log15.Logger
}

func NewContractWorker(manager *Manager) *ContractWorker {
	worker := &ContractWorker{
		manager: manager,

		status:       Create,
		isCancel:     atomic.NewBool(false),
		newBlockCond: common.NewTimeoutCond(),

		blackList: make(map[types.Address]bool),
		log:       slog.New("worker", "c"),
	}
	processors := make([]*ContractTaskProcessor, ContractTaskProcessorSize)
	for i, _ := range processors {
		processors[i] = NewContractTaskProcessor(worker, i)
	}
	worker.contractTaskProcessors = processors

	return worker
}

func (w ContractWorker) getAccEvent() *producerevent.AccountStartEvent {
	return &w.accEvent
}

func (w *ContractWorker) Start(accEvent producerevent.AccountStartEvent) {
	w.gid = accEvent.Gid
	w.address = accEvent.Address
	w.accEvent = accEvent
	if sb := w.manager.chain.GetLatestSnapshotBlock(); sb != nil {
		w.currentSnapshotHash = sb.Hash
	} else {
		w.currentSnapshotHash = w.accEvent.SnapshotHash
	}

	w.log = slog.New("worker", "c", "addr", accEvent.Address, "gid", accEvent.Gid)

	log := w.log.New("method", "start")
	log.Info("Start() current status" + strconv.Itoa(w.status))
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {
		w.isCancel.Store(false)

		// 1. get gid`s all contract address if error happened return immediately
		addressList, err := w.manager.chain.GetContractList(w.gid)
		if err != nil {
			w.log.Error("GetAddrListByGid ", "err", err)
			return
		}
		if len(addressList) == 0 {
			w.log.Info("newContractWorker addressList nil")
			return
		}
		w.contractAddressList = addressList
		log.Info("get addresslist", "len", len(addressList))

		// 2. get getAndSortAllAddrQuota it is a heavy operation so we call it only once in Start
		w.getAndSortAllAddrQuota()
		log.Info("getAndSortAllAddrQuota", "len", len(w.contractTaskPQueue))

		w.manager.addContractLis(w.gid, func(address types.Address) {
			if w.isContractInBlackList(address) {
				return
			}

			q := w.GetPledgeQuota(address)
			c := &contractTask{
				Addr:  address,
				Quota: q,
			}

			w.ctpMutex.Lock()
			heap.Push(&w.contractTaskPQueue, c)
			w.ctpMutex.Unlock()

			w.WakeupOneTp()
		})

		log.Info("start all tp")
		for _, v := range w.contractTaskProcessors {
			common.Go(v.work)
		}
		log.Info("end start all tp")

		w.status = Start
	} else {
		// awake it in order to run at least once
		w.WakeupAllTps()
	}
	w.log.Info("end start")
}

func (w *ContractWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status == Start {
		w.manager.removeContractLis(w.gid)

		w.isCancel.Store(true)
		w.newBlockCond.Broadcast()

		//w.uBlocksPool.DeleteContractCache(w.gid)
		w.clearContractBlackList()

		w.log.Info("stop all task")
		w.wg.Wait()
		w.log.Info("end stop all task")
		w.status = Stop
	}
	w.log.Info("stopped")
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

func (w *ContractWorker) WakeupOneTp() {
	w.log.Info("WakeupOneTp")
	w.newBlockCond.Signal()
}

func (w *ContractWorker) WakeupAllTps() {
	w.log.Info("WakeupAllTPs")
	w.newBlockCond.Broadcast()
}

func (w *ContractWorker) pushContractTask(t *contractTask) {
	w.ctpMutex.Lock()
	defer w.ctpMutex.Unlock()
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

func (w *ContractWorker) clearContractBlackList() {
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList = make(map[types.Address]bool)
}

// Don't deal with it for this around of blocks-generating period
func (w *ContractWorker) addContractIntoBlackList(addr types.Address) {
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList[addr] = true
	//w.uBlocksPool.ReleaseContractCache(addr)
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

func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}

func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *ContractWorker) GetPledgeQuota(addr types.Address) uint64 {
	if types.IsBuiltinContractAddrInUseWithoutQuota(addr) {
		return math.MaxUint64
	}
	quota, err := w.manager.Chain().GetPledgeQuota(addr)
	if err != nil {
		w.log.Error("GetPledgeQuotas err", "error", err)
	}
	return quota.Current()
}

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
				quotas[k] = v.Current()
			}
		}
	} else {
		var qRrr error
		// todo
		_, qRrr = w.manager.Chain().GetPledgeQuotas(beneficialList)
		if qRrr != nil {
			w.log.Error("GetPledgeQuotas err", "error", qRrr)
		}
	}
	return quotas
}

func (w *ContractWorker) VerifierConfirmedTimes(contractAddr *types.Address, fromHash *types.Hash) error {
	meta, err := w.manager.chain.GetContractMeta(*contractAddr)
	if err != nil {
		return err
	}
	if meta == nil {
		return errors.New("contract meta is nil")
	}
	if meta.SendConfirmedTimes == 0 {
		return nil
	}
	sendConfirmedTimes, err := w.manager.chain.GetConfirmedTimes(*fromHash)
	if err != nil {
		return err
	}
	if sendConfirmedTimes < uint64(meta.SendConfirmedTimes) {
		w.log.Error(fmt.Sprintf("contract(addr:%v,ct:%v), from(hash:%v,ct:%v),", contractAddr, meta.SendConfirmedTimes, fromHash, sendConfirmedTimes))
		return errors.New("sendBlock is not ready")
	}
	return nil
}
