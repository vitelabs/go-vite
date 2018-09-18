package pool

import (
	"fmt"
	"sync"
	"time"

	ch "github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/viteshan/naive-vite/common"
	"github.com/viteshan/naive-vite/common/face"
	"github.com/viteshan/naive-vite/common/log"
	"github.com/viteshan/naive-vite/syncer"
)

type PoolWriter interface {
	AddSnapshotBlock(block *ledger.SnapshotBlock) error
	AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error

	AddAccountBlock(address types.Address, block *ledger.AccountBlock) error
	// for normal account
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error

	AddAccountBlocks(address types.Address, blocks []*ledger.AccountBlock) error
	// for contract account
	AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error
}

type PoolReader interface {
	ExistInPool(address types.Address, requestHash types.Hash) bool // request对应的response是否在current链上
}

type BlockPool interface {
	PoolWriter
	PoolReader
	Start()
	Stop()
	Init(syncer.Fetcher)
	Info(addr *types.Address) string
}

type commonBlock interface {
	Height() uint64
	Hash() types.Hash
	PreHash() types.Hash
	checkForkVersion() bool
	resetForkVersion()
	forkVersion() int
}
type commonHashHeight struct {
	Hash   types.Hash
	Height uint64
}

func newForkBlock(v *ForkVersion) *forkBlock {
	return &forkBlock{firstV: v.Val(), v: v}
}

type forkBlock struct {
	firstV int
	v      *ForkVersion
}

func (self *forkBlock) forkVersion() int {
	return self.v.Val()
}
func (self *forkBlock) checkForkVersion() bool {
	return self.firstV == self.v.Val()
}
func (self *forkBlock) resetForkVersion() {
	val := self.v.Val()
	self.firstV = val
}

type pool struct {
	pendingSc *snapshotPool
	pendingAc sync.Map
	fetcher   syncer.Fetcher
	bc        ch.Chain

	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  *verifier.AccountVerifier

	rwMutex *sync.RWMutex
	acMu    sync.Mutex
	version *ForkVersion

	closed chan struct{}
	wg     sync.WaitGroup
}

func NewPool(bc ch.Chain) BlockPool {
	self := &pool{bc: bc, rwMutex: &sync.RWMutex{}, version: &ForkVersion{}, closed: make(chan struct{})}
	return self
}

func (self *pool) Init(f syncer.Fetcher) {
	self.fetcher = f
	rw := &snapshotCh{version: self.version}
	fe := &snapshotFetcher{fetcher: f}
	v := &snapshotVerifier{}
	snapshotPool := newSnapshotPool("snapshotPool", self.version, v, fe, rw)
	snapshotPool.init(
		newTools(fe, v, rw),
		self.rwMutex,
		self)

	self.pendingSc = snapshotPool
}
func (self *pool) Info(addr *types.Address) string {
	if addr == nil {
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
		ac := self.selfPendingAc(*addr)
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

func (self *pool) AddSnapshotBlock(block *ledger.SnapshotBlock) error {

	log.Info("receive snapshot block from network. height:%d, hash:%s.", block.Height, block.Hash)

	self.pendingSc.AddBlock(newSnapshotPoolBlock(block, self.version))
	return nil
}

func (self *pool) AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error {
	self.rwMutex.RLock()
	defer self.rwMutex.RUnlock()
	return self.pendingSc.AddDirectBlock(newSnapshotPoolBlock(block, self.version))
}

func (self *pool) AddAccountBlock(address types.Address, block *ledger.AccountBlock) error {
	log.Info("receive account block from network. addr:%s, height:%d, hash:%s.", address, block.Height, block.Hash)
	self.selfPendingAc(address).AddBlock(newAccountPoolBlock(block, nil, self.version))
	return nil
}

func (self *pool) AddDirectAccountBlock(address types.Address, block *vm_context.VmAccountBlock) error {
	defer monitor.LogTime("pool", "addDirectAccount", time.Now())
	self.rwMutex.RLock()
	defer self.rwMutex.RUnlock()
	ac := self.selfPendingAc(address)
	return ac.AddDirectBlock(newAccountPoolBlock(block.AccountBlock, block.VmContext, self.version))

}
func (self *pool) AddAccountBlocks(address types.Address, blocks []*ledger.AccountBlock) error {
	defer monitor.LogTime("pool", "addAccountArr", time.Now())
	for _, b := range blocks {
		self.AddAccountBlock(address, b)
	}
	return nil
}

func (self *pool) AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error {
	defer monitor.LogTime("pool", "addDirectAccountArr", time.Now())
	self.rwMutex.RLock()
	defer self.rwMutex.RUnlock()
	ac := self.selfPendingAc(address)
	// todo
	var accountPoolBlocks []*accountPoolBlock
	for _, v := range sendBlocks {
		accountPoolBlocks = append(accountPoolBlocks, newAccountPoolBlock(v.AccountBlock, v.VmContext, self.version))
	}
	return ac.AddDirectBlocks(newAccountPoolBlock(received.AccountBlock, received.VmContext, self.version), accountPoolBlocks)
}

func (self *pool) ExistInPool(address types.Address, requestHash types.Hash) bool {
	panic("implement me")
}

//func (self *pool) ForkAccounts(keyPoint *ledger.SnapshotBlock, forkPoint *ledger.SnapshotBlock) error {
//	tasks := make(map[string]*common.AccountHashH)
//	self.pendingAc.Range(func(k, v interface{}) bool {
//		a := v.(*accountPool)
//		ok, block, err := a.FindRollbackPointByReferSnapshot(forkPoint.Height(), forkPoint.Hash())
//		if err != nil {
//			log.Error("%v", err)
//			return true
//		}
//		if !ok {
//			return true
//		} else {
//			h := common.NewAccountHashH(k.(string), block.Hash(), block.Height())
//			tasks[h.Addr] = h
//		}
//		return true
//		//}
//	})
//	waitRollbackAccounts := self.getWaitRollbackAccounts(tasks)
//
//	for _, v := range waitRollbackAccounts {
//		err := self.selfPendingAc(v.Addr).Rollback(v.Height, v.Hash)
//		if err != nil {
//			return err
//		}
//	}
//	for _, v := range keyPoint.Accounts {
//		self.ForkAccountTo(v)
//	}
//	return nil
//}

func (self *pool) ForkAccounts(accounts map[types.Address][]commonBlock) error {

	for k, v := range accounts {
		self.selfPendingAc(k).rollbackCurrent(v)
	}
	return nil
}

//func (self *pool) getWaitRollbackAccounts(tasks map[string]*common.AccountHashH) map[string]*common.AccountHashH {
//	waitRollback := make(map[string]*common.AccountHashH)
//	for {
//		var sendBlocks []*ledger.AccountBlock
//		for k, v := range tasks {
//			delete(tasks, k)
//			if canAdd(waitRollback, v) {
//				waitRollback[v.Addr] = v
//			}
//			addWaitRollback(waitRollback, v)
//			tmpBlocks, err := self.selfPendingAc(v.Addr).TryRollback(v.Height, v.Hash)
//			if err == nil {
//				for _, v := range tmpBlocks {
//					sendBlocks = append(sendBlocks, v)
//				}
//			} else {
//				log.Error("%v", err)
//			}
//		}
//		for _, v := range sendBlocks {
//			sourceHash := v.Hash()
//			req := self.bc.GetAccountBySourceHash(v.To, sourceHash)
//			h := &common.AccountHashH{Addr: req.Signer(), HashHeight: common.HashHeight{Hash: req.Hash(), Height: req.Height()}}
//			if req != nil {
//				if canAdd(tasks, h) {
//					tasks[h.Addr] = h
//				}
//			}
//		}
//		if len(tasks) == 0 {
//			break
//		}
//	}
//
//	return waitRollback
//}

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
func (self *pool) PendingAccountTo(addr types.Address, h *ledger.SnapshotContentItem) error {
	this := self.selfPendingAc(addr)

	targetChain := this.findInTree(h.AccountBlockHash, h.AccountBlockHeight)
	if targetChain != nil {
		this.CurrentModifyToChain(targetChain)
		return nil
	}
	inPool := this.findInPool(h.AccountBlockHash, h.AccountBlockHeight)
	if !inPool {
		this.f.fetch(commonHashHeight{Hash: h.AccountBlockHash, Height: h.AccountBlockHeight}, 5)
	}
	return nil
}

func (self *pool) ForkAccountTo(addr types.Address, h *ledger.SnapshotContentItem) error {
	this := self.selfPendingAc(addr)

	// find in disk
	chainHash := this.rw.getHashByHeight(h.AccountBlockHeight)
	if chainHash != nil && *chainHash == h.AccountBlockHash {
		// disk block is ok
		return nil
	}
	// todo  del some blcoks
	snapshots, accounts, e := this.rw.delToHeight(h.AccountBlockHeight)
	if e != nil {
		return e
	}

	// todo rollback snapshot chain
	err := self.pendingSc.rollbackCurrent(snapshots)
	if err != nil {
		return err
	}
	// todo rollback accounts chain
	for k, v := range accounts {
		err = self.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
	}

	// find in tree
	targetChain := this.findInTree(h.AccountBlockHash, h.AccountBlockHeight)

	if targetChain == nil {
		this.f.fetch(commonHashHeight{Height: h.AccountBlockHeight, Hash: h.AccountBlockHash}, 5)
		return nil
	}

	// todo modify pool current
	if targetChain != nil {
		err = this.CurrentModifyToChain(targetChain)
	} else {
		err = this.CurrentModifyToEmpty()
	}
	return err
}

func (self *pool) selfPendingAc(addr types.Address) *accountPool {
	chain, ok := self.pendingAc.Load(addr)

	if ok {
		return chain.(*accountPool)
	}

	rw := &accountCh{address: addr, rw: self.bc, version: self.version}
	f := &accountFetcher{address: addr, fetcher: self.fetcher}
	v := &accountVerifier{}
	p := newAccountPool("accountChainPool-"+addr.Hex(), rw, self.version)

	p.Init(newTools(f, v, rw), self.rwMutex.RLocker())

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
	var tasks []verifyTask
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
func (self *pool) fetchForTask(task verifyTask) []*face.FetchRequest {
	reqs := task.requests()
	if len(reqs) <= 0 {
		return nil
	}
	// if something in pool, deal with it.
	var existReqs []*face.FetchRequest
	for _, r := range reqs {
		exist := false
		//if r.Chain == "" {
		//	exist = self.pendingSc.ExistInCurrent(r)
		//} else {
		//	exist = self.selfPendingAc(r.Chain).ExistInCurrent(r)
		//}

		if !exist {
			self.fetcher.Fetch(r)
		} else {
			log.Info("block[%s] exist, should not fetch.", r.String())
			existReqs = append(existReqs, &r)
		}
	}
	return existReqs
}
