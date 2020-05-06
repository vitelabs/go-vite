package pool

import (
	"encoding/json"

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

func (self *pool) Init(f syncer.Fetcher) {
	self.snapshotVerifier = verifier.NewSnapshotVerifier(self.bc, self.version)
	self.accountVerifier = verifier.NewAccountVerifier(self.bc, self.version)
	self.fetcher = f
	snapshotPool := newSnapshotPool("snapshotPool", self.version)
	snapshotPool.init(&snapshotCh{self.bc, self.version},
		self.snapshotVerifier,
		NewFetcher("", self.fetcher),
		self.rwMutex,
		self)

	self.pendingSc = snapshotPool
}
func (self *pool) Info(id string) string {
	if id == "" {
		bp := self.pendingSc.blockpool
		cp := self.pendingSc.chainpool

		freeSize := len(bp.freeBlocks)
		compoundSize := len(bp.compoundBlocks)
		snippetSize := len(cp.snippetChains)
		currentLen := cp.current.size()
		chainSize := len(cp.chains)
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, currentLen, chainSize)
	} else {
		ac := self.selfPendingAc(id)
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
func (self *pool) Start() {
	self.pendingSc.Start()
	go self.loopTryInsert()
	go self.loopCompact()
}
func (self *pool) Stop() {
	self.pendingSc.Stop()
	close(self.closed)
	self.wg.Wait()
}

func (self *pool) AddSnapshotBlock(block *common.SnapshotBlock) error {
	log.Info("receive snapshot block from network. height:%d, hash:%s.", block.Height(), block.Hash())
	self.pendingSc.AddBlock(block)
	return nil
}

func (self *pool) AddDirectSnapshotBlock(block *common.SnapshotBlock) error {
	self.rwMutex.RLock()
	defer self.rwMutex.RUnlock()
	return self.pendingSc.AddDirectBlock(block)
}

func (self *pool) AddAccountBlock(address string, block *common.AccountStateBlock) error {
	log.Info("receive account block from network. addr:%s, height:%d, hash:%s.", address, block.Height(), block.Hash())
	self.selfPendingAc(address).AddBlock(block)
	return nil
}

func (self *pool) AddDirectAccountBlock(address string, block *common.AccountStateBlock) error {
	defer monitor.LogTime("pool", "addDirectAccount", time.Now())
	self.rwMutex.RLock()
	defer self.rwMutex.RUnlock()
	ac := self.selfPendingAc(address)
	return ac.AddDirectBlock(block)

}

func (self *pool) ExistInPool(address string, requestHash string) bool {
	panic("implement me")
}

func (self *pool) ForkAccounts(keyPoint *common.SnapshotBlock, forkPoint *common.SnapshotBlock) error {
	tasks := make(map[string]*common.AccountHashH)
	self.pendingAc.Range(func(k, v interface{}) bool {
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
	waitRollbackAccounts := self.getWaitRollbackAccounts(tasks)

	for _, v := range waitRollbackAccounts {
		err := self.selfPendingAc(v.Addr).Rollback(v.Height, v.Hash)
		if err != nil {
			return err
		}
	}
	for _, v := range keyPoint.Accounts {
		self.ForkAccountTo(v)
	}
	return nil
}
func (self *pool) getWaitRollbackAccounts(tasks map[string]*common.AccountHashH) map[string]*common.AccountHashH {
	waitRollback := make(map[string]*common.AccountHashH)
	for {
		var sendBlocks []*common.AccountStateBlock
		for k, v := range tasks {
			delete(tasks, k)
			if canAdd(waitRollback, v) {
				waitRollback[v.Addr] = v
			}
			addWaitRollback(waitRollback, v)
			tmpBlocks, err := self.selfPendingAc(v.Addr).TryRollback(v.Height, v.Hash)
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
			req := self.bc.GetAccountBySourceHash(v.To, sourceHash)
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
func (self *pool) PendingAccountTo(h *common.AccountHashH) error {
	this := self.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		self.fetcher.Fetch(face.FetchRequest{Chain: h.Addr, Height: h.Height, Hash: h.Hash, PrevCnt: 5})
		return nil
	}
	return nil
}

func (self *pool) ForkAccountTo(h *common.AccountHashH) error {
	this := self.selfPendingAc(h.Addr)

	inChain := this.FindInChain(h.Hash, h.Height)
	bytes, _ := json.Marshal(h)
	log.Info("inChain:%v, accounts:%s", inChain, string(bytes))
	if !inChain {
		self.fetcher.Fetch(face.FetchRequest{Chain: h.Addr, Height: h.Height, Hash: h.Hash, PrevCnt: 5})
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
	waitRollback := self.getWaitRollbackAccounts(tasks)
	for _, v := range waitRollback {
		self.selfPendingAc(v.Addr).Rollback(v.Height, v.Hash)
	}
	err = this.CurrentModifyToChain(chain)
	if err != nil {
		log.Error("%v", err)
	}
	return err
}

func (self *pool) UnLockAccounts(startAcs map[string]*common.SnapshotPoint, endAcs map[string]*common.SnapshotPoint) error {
	for k, v := range startAcs {
		err := self.bc.RollbackSnapshotPoint(k, v, endAcs[k])
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *pool) selfPendingAc(addr string) *accountPool {
	chain, ok := self.pendingAc.Load(addr)

	if ok {
		return chain.(*accountPool)
	}

	p := newAccountPool("accountChainPool-"+addr, &accountCh{addr, self.bc, self.version}, self.version)
	p.Init(self.accountVerifier, NewFetcher(addr, self.fetcher), self.rwMutex.RLocker())

	self.acMu.Lock()
	defer self.acMu.Unlock()
	chain, ok = self.pendingAc.Load(addr)
	if ok {
		return chain.(*accountPool)
	}
	self.pendingAc.Store(addr, p)
	return p

}
func (self *pool) loopTryInsert() {
	self.wg.Add(1)
	defer self.wg.Done()

	t := time.NewTicker(time.Millisecond * 20)
	sum := 0
	for {
		select {
		case <-self.closed:
			return
		case <-t.C:
			if sum == 0 {
				time.Sleep(time.Millisecond * 10)
				monitor.LogEvent("pool", "tryInsertSleep")
			}
			sum = 0
			sum += self.accountsTryInsert()
		default:
			sum += self.accountsTryInsert()
		}
	}
}

func (self *pool) accountsTryInsert() int {
	monitor.LogEvent("pool", "tryInsert")
	sum := 0
	var pending []*accountPool
	self.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		pending = append(pending, p)
		return true
	})
	var tasks []verifier.Task
	for _, p := range pending {
		task := p.TryInsert()
		if task != nil {
			self.fetchForTask(task)
			tasks = append(tasks, task)
			sum = sum + 1
		}
	}
	return sum
}

func (self *pool) loopCompact() {
	self.wg.Add(1)
	defer self.wg.Done()

	t := time.NewTicker(time.Millisecond * 40)
	sum := 0
	for {
		select {
		case <-self.closed:
			return
		case <-t.C:
			if sum == 0 {
				time.Sleep(time.Millisecond * 20)
			}
			sum = 0

			sum += self.accountsCompact()
		default:
			sum += self.accountsCompact()
		}
	}
}

func (self *pool) accountsCompact() int {
	sum := 0
	var pendings []*accountPool
	self.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		pendings = append(pendings, p)
		return true
	})
	for _, p := range pendings {
		sum = sum + p.Compact()
	}
	return sum
}
func (self *pool) fetchForTask(task verifier.Task) []*face.FetchRequest {
	reqs := task.Requests()
	if len(reqs) <= 0 {
		return nil
	}
	// if something in pool, deal with it.
	var existReqs []*face.FetchRequest
	for _, r := range reqs {
		exist := false
		if r.Chain == "" {
			exist = self.pendingSc.ExistInCurrent(r)
		} else {
			exist = self.selfPendingAc(r.Chain).ExistInCurrent(r)
		}

		if !exist {
			self.fetcher.Fetch(r)
		} else {
			log.Info("block[%s] exist, should not fetch.", r.String())
			existReqs = append(existReqs, &r)
		}
	}
	return existReqs
}
