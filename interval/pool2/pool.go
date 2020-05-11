package pool

import (
	"encoding/json"

	"github.com/vitelabs/go-vite/interval/utils"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/syncer"
	"github.com/vitelabs/go-vite/interval/verifier"

	"sync"

	"time"

	"fmt"

	ch "github.com/vitelabs/go-vite/interval/chain"
	"github.com/vitelabs/go-vite/interval/monitor"
	"github.com/vitelabs/go-vite/interval/version"
)

type BlockPool interface {
	face.PoolWriter
	face.PoolReader
	Start()
	Stop()
	Init(syncer.Fetcher)
	Info(string) string
}

type pool struct {
	pendingSc *snapshotPool
	pendingAc sync.Map
	fetcher   syncer.Fetcher
	bc        ch.BlockChain

	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  *verifier.AccountVerifier

	rwMutex *sync.RWMutex
	acMu    sync.Mutex
	version *version.Version

	closed chan struct{}
	wg     sync.WaitGroup
}

func NewPool(bc ch.BlockChain, rwMutex *sync.RWMutex) BlockPool {
	self := &pool{bc: bc, rwMutex: rwMutex, version: &version.Version{}, closed: make(chan struct{})}
	return self
}

func (pl *pool) Init(f syncer.Fetcher) {
	pl.snapshotVerifier = verifier.NewSnapshotVerifier(pl.bc, pl.version)
	pl.accountVerifier = verifier.NewAccountVerifier(pl.bc, pl.version)
	pl.fetcher = f
	snapshotPool := newSnapshotPool("snapshotPool", pl.version)
	snapshotPool.init(&snapshotCh{pl.bc, pl.version},
		pl.snapshotVerifier,
		NewFetcher("", pl.fetcher),
		pl.rwMutex,
		pl)

	pl.pendingSc = snapshotPool
}
func (pl *pool) Info(id string) string {
	if id == "" {
		bp := pl.pendingSc.blockpool
		cp := pl.pendingSc.chainpool

		freeSize := len(bp.freeBlocks)
		compoundSize := len(bp.compoundBlocks)
		snippetSize := len(cp.snippetChains)
		currentLen := cp.current.size()
		chainSize := len(cp.chains)
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, currentLen, chainSize)
	} else {
		ac := pl.selfPendingAc(id)
		if ac == nil {
			return "pool not exist."
		}
		bp := ac.blockpool
		cp := ac.chainpool

		freeSize := len(bp.freeBlocks)
		compoundSize := len(bp.compoundBlocks)
		snippetSize := len(cp.snippetChains)
		currentLen := cp.current.size()
		chainSize := len(cp.chains)
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, currentLen, chainSize)
	}
}
func (pl *pool) Start() {
	pl.pendingSc.Start()
	go pl.loopTryInsert()
	go pl.loopCompact()
}
func (pl *pool) Stop() {
	pl.pendingSc.Stop()
	close(pl.closed)
	pl.wg.Wait()
}

func (pl *pool) AddSnapshotBlock(block *common.SnapshotBlock) error {
	log.Info("receive snapshot block from network. height:%d, hash:%s.", block.Height(), block.Hash())
	pl.pendingSc.AddBlock(block)
	return nil
}

func (pl *pool) AddDirectSnapshotBlock(block *common.SnapshotBlock) error {
	pl.rwMutex.RLock()
	defer pl.rwMutex.RUnlock()
	return pl.pendingSc.AddDirectBlock(block)
}

func (pl *pool) AddAccountBlock(address string, block *common.AccountStateBlock) error {
	log.Info("receive account block from network. addr:%s, height:%d, hash:%s.", address, block.Height(), block.Hash())
	pl.selfPendingAc(address).AddBlock(block)
	return nil
}

func (pl *pool) AddDirectAccountBlock(address string, block *common.AccountStateBlock) error {
	defer monitor.LogTime("pool", "addDirectAccount", time.Now())
	pl.rwMutex.RLock()
	defer pl.rwMutex.RUnlock()
	ac := pl.selfPendingAc(address)
	return ac.AddDirectBlock(block)

}

func (pl *pool) ExistInPool(address string, requestHash string) bool {
	panic("implement me")
}

func (pl *pool) ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error {
	tasks := make(map[string]*common.AccountHashH)
	pl.pendingAc.Range(func(k, v interface{}) bool {
		a := v.(*accountPool)
		ok, block, err := a.FindRollbackPointByReferSnapshot(forkPoint.Height(), forkPoint.Hash())
		if err != nil {
			log.Error("%v", err)
			return true
		}
		if !ok {
			return true
		} else {
			h := common.NewAccountHashH(k.(string), block.Hash(), block.Height())
			tasks[h.Addr] = h
		}
		return true
		//}
	})
	waitRollbackAccounts := pl.getWaitRollbackAccounts(tasks)

	for _, v := range waitRollbackAccounts {
		err := pl.selfPendingAc(v.Addr).Rollback(v.Height, v.Hash)
		if err != nil {
			return err
		}
	}
	for _, v := range keyPoint.Accounts {
		pl.ForkAccountTo(v)
	}
	return nil
}
func (pl *pool) getWaitRollbackAccounts(tasks map[string]*common.AccountHashH) map[string]*common.AccountHashH {
	waitRollback := make(map[string]*common.AccountHashH)
	for {
		var sendBlocks []*common.AccountStateBlock
		for k, v := range tasks {
			delete(tasks, k)
			if canAdd(waitRollback, v) {
				waitRollback[v.Addr] = v
			}
			addWaitRollback(waitRollback, v)
			tmpBlocks, err := pl.selfPendingAc(v.Addr).TryRollback(v.Height, v.Hash)
			if err == nil {
				for _, v := range tmpBlocks {
					sendBlocks = append(sendBlocks, v)
				}
			} else {
				log.Error("%v", err)
			}
		}
		for _, v := range sendBlocks {
			sourceHash := v.Hash()
			req := pl.bc.GetAccountByFromHash(v.To, sourceHash)
			h := &common.AccountHashH{Addr: req.Signer(), HashHeight: common.HashHeight{Hash: req.Hash(), Height: req.Height()}}
			if req != nil {
				if canAdd(tasks, h) {
					tasks[h.Addr] = h
				}
			}
		}
		if len(tasks) == 0 {
			break
		}
	}

	return waitRollback
}

// h is closer to genesis
func canAdd(hs map[string]*common.AccountHashH, h *common.AccountHashH) bool {
	hashH := hs[h.Addr]
	if hashH == nil {
		return true
	}

	if h.Height < hashH.Height {
		return true
	}
	return false
}
func addWaitRollback(hs map[string]*common.AccountHashH, h *common.AccountHashH) {
	hashH := hs[h.Addr]
	if hashH == nil {
		hs[h.Addr] = h
		return
	}

	if hashH.Height < h.Height {
		hs[h.Addr] = h
		return
	}
}
func (pl *pool) PendingAccountTo(h *common.AccountHashH) error {
	this := pl.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		pl.fetcher.Fetch(face.FetchRequest{Chain: h.Addr, Height: h.Height, Hash: h.Hash, PrevCnt: 5})
		return nil
	}
	return nil
}

func (pl *pool) ForkAccountTo(h *common.AccountHashH) error {
	this := pl.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		pl.fetcher.Fetch(face.FetchRequest{Chain: h.Addr, Height: h.Height, Hash: h.Hash, PrevCnt: 5})
		return nil
	}
	ok, block, chain, err := this.FindRollbackPointForAccountHashH(h.Height, h.Hash)
	if err != nil {
		log.Error("%v", err)
	}
	if !ok {
		return nil
	}

	tasks := make(map[string]*common.AccountHashH)
	tasks[h.Addr] = common.NewAccountHashH(h.Addr, block.Hash(), block.Height())
	waitRollback := pl.getWaitRollbackAccounts(tasks)
	for _, v := range waitRollback {
		pl.selfPendingAc(v.Addr).Rollback(v.Height, v.Hash)
	}
	err = this.CurrentModifyToChain(chain)
	if err != nil {
		log.Error("%v", err)
	}
	return err
}

func (pl *pool) UnLockAccounts(startAcs map[string]*common.SnapshotPoint, endAcs map[string]*common.SnapshotPoint) error {
	for k, v := range startAcs {
		err := pl.bc.RollbackSnapshotPoint(k, v, endAcs[k])
		if err != nil {
			return err
		}
	}
	return nil
}

func (pl *pool) selfPendingAc(addr string) *accountPool {
	chain, ok := pl.pendingAc.Load(addr)

	if ok {
		return chain.(*accountPool)
	}

	p := newAccountPool("accountChainPool-"+addr, &accountCh{addr, pl.bc, pl.version}, pl.version)
	p.Init(pl.accountVerifier, NewFetcher(addr, pl.fetcher), pl.rwMutex.RLocker())

	pl.acMu.Lock()
	defer pl.acMu.Unlock()
	chain, ok = pl.pendingAc.Load(addr)
	if ok {
		return chain.(*accountPool)
	}
	pl.pendingAc.Store(addr, p)
	return p

}
func (pl *pool) loopTryInsert() {
	pl.wg.Add(1)
	defer pl.wg.Done()

	t := time.NewTicker(time.Millisecond * 20)
	sum := 0
	for {
		select {
		case <-pl.closed:
			return
		case <-t.C:
			if sum == 0 {
				time.Sleep(time.Millisecond * 10)
				monitor.LogEvent("pool", "tryInsertSleep")
			}
			sum = 0
			sum += pl.accountsTryInsert()
		default:
			sum += pl.accountsTryInsert()
		}
	}
}

func (pl *pool) accountsTryInsert() int {
	monitor.LogEvent("pool", "tryInsert")
	sum := 0
	var pending []*accountPool
	pl.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		pending = append(pending, p)
		return true
	})
	var tasks []verifier.Task
	for _, p := range pending {
		task := p.TryInsert()
		if task != nil {
			pl.fetchForTask(task)
			tasks = append(tasks, task)
			sum = sum + 1
		}
	}
	return sum
}

func (pl *pool) loopCompact() {
	pl.wg.Add(1)
	defer pl.wg.Done()

	t := time.NewTicker(time.Millisecond * 40)
	sum := 0
	for {
		select {
		case <-pl.closed:
			return
		case <-t.C:
			if sum == 0 {
				time.Sleep(time.Millisecond * 20)
			}
			sum = 0

			sum += pl.accountsCompact()
		default:
			sum += pl.accountsCompact()
		}
	}
}

func (pl *pool) accountsCompact() int {
	sum := 0
	var pendings []*accountPool
	pl.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		pendings = append(pendings, p)
		return true
	})
	for _, p := range pendings {
		sum = sum + p.Compact()
	}
	return sum
}
func (pl *pool) fetchForTask(task verifier.Task) []*face.FetchRequest {
	reqs := task.Requests()
	if len(reqs) <= 0 {
		return nil
	}
	// if something in pool, deal with it.
	var existReqs []*face.FetchRequest
	for _, r := range reqs {
		exist := false
		if r.Chain == "" {
			exist = pl.pendingSc.ExistInCurrent(r)
		} else {
			exist = pl.selfPendingAc(r.Chain).ExistInCurrent(r)
		}

		if !exist {
			pl.fetcher.Fetch(r)
		} else {
			log.Info("block[%s] exist, should not fetch.", r.String())
			existReqs = append(existReqs, &r)
		}
	}
	return existReqs
}

func (pl *pool) Rollback(height uint64) error {
	snapshotBlocks, acctBlocksMap, err := pl.bc.RollbackSnapshotBlockTo(height)
	if err != nil {
		return err
	}
	for _, block := range utils.ReverseSnapshotBlocks(snapshotBlocks) {
		err := pl.pendingSc.AddTail(block)
		if err != nil {
			return err
		}
	}

	for addr, blocks := range acctBlocksMap {
		for _, block := range blocks {
			err := pl.selfPendingAc(addr).AddTail(block)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
