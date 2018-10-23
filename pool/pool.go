package pool

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/wallet/seedstore"
)

type Writer interface {
	// for normal account
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error

	// for contract account
	AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error
}

type SnapshotProducerWriter interface {
	Lock()

	UnLock()

	AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error

	RollbackAccountTo(addr types.Address, hash types.Hash, height uint64) error
}

type Reader interface {
	// received block in current? (key is requestHash)
	ExistInPool(address types.Address, requestHash types.Hash) bool
}
type Debug interface {
	Info(addr *types.Address) string
	Snapshot() map[string]interface{}
	Account(addr types.Address) map[string]interface{}
	SnapshotChainDetail(chainId string) map[string]interface{}
	AccountChainDetail(addr types.Address, chainId string) map[string]interface{}
}

type BlockPool interface {
	Writer
	Reader
	SnapshotProducerWriter
	Debug

	Start()
	Stop()
	Init(s syncer,
		wt *wallet.Manager,
		snapshotV *verifier.SnapshotVerifier,
		accountV *verifier.AccountVerifier)
	Details(addr *types.Address, hash types.Hash) string
}

type commonBlock interface {
	Height() uint64
	Hash() types.Hash
	PrevHash() types.Hash
	checkForkVersion() bool
	resetForkVersion()
	forkVersion() int
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
	pendingAc sync.Map // key:address v:*accountPool

	sync syncer
	bc   chainDb
	wt   *wallet.Manager

	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  *verifier.AccountVerifier

	accountSubId  int
	snapshotSubId int

	rwMutex sync.RWMutex
	version *ForkVersion

	closed      chan struct{}
	wg          sync.WaitGroup
	accountCond *sync.Cond // if new block add, notify

	log log15.Logger

	stat *recoverStat
}

func (self *pool) Snapshot() map[string]interface{} {
	return self.pendingSc.info()
}

func (self *pool) Account(addr types.Address) map[string]interface{} {
	return self.selfPendingAc(addr).info()
}

func (self *pool) SnapshotChainDetail(chainId string) map[string]interface{} {
	return self.pendingSc.detailChain(chainId)
}

func (self *pool) AccountChainDetail(addr types.Address, chainId string) map[string]interface{} {
	return self.selfPendingAc(addr).detailChain(chainId)
}

func (self *pool) Lock() {
	self.rwMutex.Lock()
}

func (self *pool) UnLock() {
	self.rwMutex.Unlock()
}

func (self *pool) RLock() {
	self.rwMutex.RLock()
}

func (self *pool) RUnLock() {
	self.rwMutex.RUnlock()
}

func NewPool(bc chainDb) *pool {
	self := &pool{bc: bc, rwMutex: sync.RWMutex{}, version: &ForkVersion{}, accountCond: sync.NewCond(&sync.Mutex{})}
	self.log = log15.New("module", "pool")
	return self
}

func (self *pool) Init(s syncer,
	wt *wallet.Manager,
	snapshotV *verifier.SnapshotVerifier,
	accountV *verifier.AccountVerifier) {
	self.sync = s
	self.wt = wt
	rw := &snapshotCh{version: self.version, bc: self.bc}
	fe := &snapshotSyncer{fetcher: s, log: self.log.New("t", "snapshot")}
	v := &snapshotVerifier{v: snapshotV}
	self.accountVerifier = accountV
	snapshotPool := newSnapshotPool("snapshotPool", self.version, v, fe, rw, self.log)
	snapshotPool.init(
		newTools(fe, rw),
		self)

	self.pendingSc = snapshotPool
	self.stat = (&recoverStat{}).reset(10)
}
func (self *pool) Info(addr *types.Address) string {
	if addr == nil {
		bp := self.pendingSc.blockpool
		cp := self.pendingSc.chainpool

		freeSize := len(bp.freeBlocks)
		compoundSize := len(bp.compoundBlocks)
		snippetSize := len(cp.snippetChains)
		currentLen := cp.current.size()
		chainSize := cp.size()
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
		chainSize := cp.size()
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, currentLen, chainSize)
	}
}
func (self *pool) Details(addr *types.Address, hash types.Hash) string {
	if addr == nil {
		bp := self.pendingSc.blockpool

		b := bp.get(hash)
		if b == nil {
			return "not exist"
		}
		bytes, _ := json.Marshal(b.(*snapshotPoolBlock).block)
		return string(bytes)
	} else {
		ac := self.selfPendingAc(*addr)
		if ac == nil {
			return "pool not exist."
		}
		bp := ac.blockpool
		b := bp.get(hash)
		if b == nil {
			return "not exist"
		}
		bytes, _ := json.Marshal(b.(*snapshotPoolBlock).block)
		return string(bytes)
	}
}
func (self *pool) Start() {
	self.log.Info("pool start.")
	defer self.log.Info("pool started.")
	self.closed = make(chan struct{})

	self.accountSubId = self.sync.SubscribeAccountBlock(self.AddAccountBlock)
	self.snapshotSubId = self.sync.SubscribeSnapshotBlock(self.AddSnapshotBlock)

	self.pendingSc.Start()
	common.Go(self.loopTryInsert)
	common.Go(self.loopCompact)
	common.Go(self.loopBroadcastAndDel)
}
func (self *pool) Stop() {
	self.log.Info("pool stop.")
	defer self.log.Info("pool stopped.")
	self.sync.UnsubscribeAccountBlock(self.accountSubId)
	self.accountSubId = 0
	self.sync.UnsubscribeSnapshotBlock(self.snapshotSubId)
	self.snapshotSubId = 0

	self.pendingSc.Stop()
	close(self.closed)
	self.wg.Wait()
}
func (self *pool) Restart() {
	self.Lock()
	defer self.UnLock()
	self.log.Info("pool restart.")
	defer self.log.Info("pool restarted.")
	self.Stop()
	self.Start()
}

func (self *pool) AddSnapshotBlock(block *ledger.SnapshotBlock) {

	self.log.Info("receive snapshot block from network. height:" + strconv.FormatUint(block.Height, 10) + ", hash:" + block.Hash.String() + ".")

	err := self.pendingSc.v.verifySnapshotData(block)
	if err != nil {
		self.log.Error("snapshot error", "err", err, "height", block.Height, "hash", block.Hash)
		return
	}
	self.pendingSc.AddBlock(newSnapshotPoolBlock(block, self.version))
}

func (self *pool) AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error {
	err := self.pendingSc.v.verifySnapshotData(block)
	if err != nil {
		return err
	}
	cBlock := newSnapshotPoolBlock(block, self.version)
	err = self.pendingSc.AddDirectBlock(cBlock)
	if err != nil {
		return err
	}
	self.pendingSc.f.broadcastBlock(block)
	return nil
}

func (self *pool) AddAccountBlock(address types.Address, block *ledger.AccountBlock) {
	self.log.Info(fmt.Sprintf("receive account block from network. addr:%s, height:%d, hash:%s.", address, block.Height, block.Hash))

	ac := self.selfPendingAc(address)
	err := ac.v.verifyAccountData(block)
	if err != nil {
		self.log.Error("account err", "err", err, "height", block.Height, "hash", block.Hash, "addr", address)
		return
	}
	ac.AddBlock(newAccountPoolBlock(block, nil, self.version))
	ac.AddReceivedBlock(block)

	self.accountCond.L.Lock()
	defer self.accountCond.L.Unlock()
	self.accountCond.Broadcast()
}

func (self *pool) AddDirectAccountBlock(address types.Address, block *vm_context.VmAccountBlock) error {
	self.log.Info(fmt.Sprintf("receive account block from direct. addr:%s, height:%d, hash:%s.", address, block.AccountBlock.Height, block.AccountBlock.Hash))
	defer monitor.LogTime("pool", "addDirectAccount", time.Now())
	self.RLock()
	defer self.RUnLock()

	ac := self.selfPendingAc(address)

	err := ac.v.verifyAccountData(block.AccountBlock)
	if err != nil {
		self.log.Error("account err", "err", err, "height", block.AccountBlock.Height, "hash", block.AccountBlock.Hash, "addr", address)
		return err
	}

	cBlock := newAccountPoolBlock(block.AccountBlock, block.VmContext, self.version)
	err = ac.AddDirectBlocks(cBlock, nil)
	if err != nil {
		return err
	}
	ac.f.broadcastBlock(block.AccountBlock)
	self.accountCond.L.Lock()
	defer self.accountCond.L.Unlock()
	self.accountCond.Broadcast()
	return nil

}
func (self *pool) AddAccountBlocks(address types.Address, blocks []*ledger.AccountBlock) error {
	defer monitor.LogTime("pool", "addAccountArr", time.Now())

	for _, b := range blocks {
		self.AddAccountBlock(address, b)
	}

	self.accountCond.L.Lock()
	defer self.accountCond.L.Unlock()
	self.accountCond.Broadcast()
	return nil
}

func (self *pool) AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error {
	defer monitor.LogTime("pool", "addDirectAccountArr", time.Now())
	self.RLock()
	defer self.RUnLock()
	ac := self.selfPendingAc(address)
	// todo
	var accountPoolBlocks []*accountPoolBlock
	for _, v := range sendBlocks {
		accountPoolBlocks = append(accountPoolBlocks, newAccountPoolBlock(v.AccountBlock, v.VmContext, self.version))
	}
	err := ac.AddDirectBlocks(newAccountPoolBlock(received.AccountBlock, received.VmContext, self.version), accountPoolBlocks)
	if err != nil {
		return err
	}
	ac.f.broadcastReceivedBlocks(received, sendBlocks)

	self.accountCond.L.Lock()
	defer self.accountCond.L.Unlock()
	self.accountCond.Broadcast()
	return nil
}

func (self *pool) ExistInPool(address types.Address, requestHash types.Hash) bool {
	return self.selfPendingAc(address).ExistInCurrent(requestHash)
}

func (self *pool) ForkAccounts(accounts map[types.Address][]commonBlock) error {

	for k, v := range accounts {
		self.selfPendingAc(k).rollbackCurrent(v)
	}
	return nil
}

func (self *pool) PendingAccountTo(addr types.Address, h *ledger.HashHeight) (*ledger.HashHeight, error) {
	this := self.selfPendingAc(addr)

	targetChain := this.findInTree(h.Hash, h.Height)
	if targetChain != nil {
		if targetChain.ChainId() == this.chainpool.current.ChainId() {
			return nil, nil
		}

		_, forkPoint, err := this.getForkPointByChains(targetChain, this.CurrentChain())
		if err != nil {
			return nil, err
		}
		// fork point in disk chain
		if forkPoint.Height() <= this.CurrentChain().tailHeight {
			return h, nil
		}

		this.LockForInsert()
		defer this.UnLockForInsert()
		this.CurrentModifyToChain(targetChain, h)
		return nil, nil
	}
	inPool := this.findInPool(h.Hash, h.Height)
	if !inPool {
		this.f.fetch(ledger.HashHeight{Hash: h.Hash, Height: h.Height}, 5)
	}
	return nil, nil
}

func (self *pool) ForkAccountTo(addr types.Address, h *ledger.HashHeight) error {
	this := self.selfPendingAc(addr)
	err := self.RollbackAccountTo(addr, h.Hash, h.Height)

	if err != nil {
		return err
	}
	// find in tree
	targetChain := this.findInTree(h.Hash, h.Height)

	if targetChain == nil {
		cnt := h.Height - this.chainpool.diskChain.Head().Height()
		this.f.fetch(ledger.HashHeight{Height: h.Height, Hash: h.Hash}, cnt)
		err = this.CurrentModifyToEmpty()
		return err
	}

	keyPoint, forkPoint, err := this.getForkPointByChains(targetChain, this.CurrentChain())
	if err != nil {
		return err
	}
	if keyPoint == nil {
		return errors.New("forkAccountTo key point is nil.")
	}
	// fork point in disk chain
	if forkPoint.Height() <= this.CurrentChain().tailHeight {
		err := self.RollbackAccountTo(addr, keyPoint.Hash(), keyPoint.Height())
		if err != nil {
			return err
		}
	}

	err = this.CurrentModifyToChain(targetChain, h)
	if err != nil {
		return err
	}
	return nil
}

func (self *pool) RollbackAccountTo(addr types.Address, hash types.Hash, height uint64) error {
	p := self.selfPendingAc(addr)

	// del some blcoks
	snapshots, accounts, e := p.rw.delToHeight(height)
	if e != nil {
		return e
	}

	// rollback snapshot chain in pool
	err := self.pendingSc.rollbackCurrent(snapshots)
	if err != nil {
		return err
	}
	// rollback accounts chain in pool
	for k, v := range accounts {
		err = self.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
	}
	return err
}

func (self *pool) selfPendingAc(addr types.Address) *accountPool {
	chain, ok := self.pendingAc.Load(addr)

	if ok {
		return chain.(*accountPool)
	}

	// lazy load
	rw := &accountCh{address: addr, rw: self.bc, version: self.version}
	f := &accountSyncer{address: addr, fetcher: self.sync, log: self.log.New()}
	v := &accountVerifier{v: self.accountVerifier, log: self.log.New()}
	p := newAccountPool("accountChainPool-"+addr.Hex(), rw, self.version, self.log)

	p.Init(newTools(f, rw), self, v, f)

	chain, _ = self.pendingAc.LoadOrStore(addr, p)
	return chain.(*accountPool)
}
func (self *pool) loopTryInsert() {
	defer self.poolRecover()
	self.wg.Add(1)
	defer self.wg.Done()

	t := time.NewTicker(time.Millisecond * 20)
	defer t.Stop()
	sum := 0
	for {
		select {
		case <-self.closed:
			return
		case <-t.C:
			if sum == 0 {
				//self.accountCond.L.Lock()
				//self.accountCond.Wait()
				//self.accountCond.L.Unlock()
				time.Sleep(200 * time.Millisecond)
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
	defer self.poolRecover()
	self.wg.Add(1)
	defer self.wg.Done()

	t := time.NewTicker(time.Millisecond * 40)
	defer t.Stop()
	sum := 0
	for {
		select {
		case <-self.closed:
			return
		case <-t.C:
			if sum == 0 {
				//self.accountCond.L.Lock()
				//self.accountCond.Wait()
				//self.accountCond.L.Unlock()
				time.Sleep(200 * time.Millisecond)
			}
			sum = 0

			sum += self.accountsCompact()
		default:
			sum += self.accountsCompact()
		}
	}
}
func (self *pool) poolRecover() {
	if err := recover(); err != nil {
		var e error
		switch t := err.(type) {
		case error:
			e = errors.WithStack(t)
		case string:
			e = errors.New(t)
		default:
			e = errors.Errorf("unknown type", err)
		}

		self.log.Error("panic", "err", err, "withstack", fmt.Sprintf("%+v", e))
		fmt.Printf("%+v", e)
		if self.stat.inc() {
			common.Go(self.Restart)
		} else {
			panic(e)
		}
	}
}
func (self *pool) loopBroadcastAndDel() {
	defer self.poolRecover()
	self.wg.Add(1)
	defer self.wg.Done()

	broadcastT := time.NewTicker(time.Second * 30)
	delT := time.NewTicker(time.Second * 40)
	delUselessChainT := time.NewTicker(time.Minute)

	defer broadcastT.Stop()
	defer delT.Stop()
	for {
		select {
		case <-self.closed:
			return
		case <-broadcastT.C:
			addrList := self.listUnlockedAddr()
			for _, addr := range addrList {
				self.selfPendingAc(addr).broadcastUnConfirmedBlocks()
			}
		case <-delT.C:
			addrList := self.listUnlockedAddr()
			for _, addr := range addrList {
				self.delTimeoutUnConfirmedBlocks(addr)
			}
		case <-delUselessChainT.C:
			// del some useless chain in pool
			self.delUseLessChains()
		}
	}
}

func (self *pool) delUseLessChains() {
	self.pendingSc.loopDelUselessChain()
	var pendings []*accountPool
	self.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		pendings = append(pendings, p)
		return true
	})
	for _, v := range pendings {
		v.loopDelUselessChain()
	}
}

func (self *pool) listUnlockedAddr() []types.Address {
	var todoAddress []types.Address
	status, e := self.wt.SeedStoreManagers.Status()
	if e != nil {
		return todoAddress
	}
	for k, v := range status {
		if v == seedstore.Locked {
			todoAddress = append(todoAddress, k)
		}
	}
	return todoAddress
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
func (self *pool) fetchForTask(task verifyTask) {
	reqs := task.requests()
	if len(reqs) <= 0 {
		return
	}
	// if something in pool, deal with it.
	for _, r := range reqs {
		exist := false
		if r.snapshot {
			exist = self.pendingSc.existInPool(r.hash)
		} else {
			if r.chain != nil {
				exist = self.selfPendingAc(*r.chain).existInPool(r.hash)
			}
		}
		if exist {
			self.log.Info(fmt.Sprintf("block[%s] exist, should not fetch.", r.String()))
			continue
		}

		if r.snapshot {
			self.pendingSc.f.fetchByHash(r.hash, 5)
		} else {
			// todo
			self.sync.FetchAccountBlocks(r.hash, 5, r.chain)
			//self.selfPendingAc(*r.chain).f.fetchByHash(r.hash, 5)
		}
	}
	return
}
func (self *pool) delTimeoutUnConfirmedBlocks(addr types.Address) {
	headSnapshot := self.pendingSc.rw.headSnapshot()
	ac := self.selfPendingAc(addr)
	firstUnconfirmedBlock := ac.rw.getFirstUnconfirmedBlock(headSnapshot)
	if firstUnconfirmedBlock == nil {
		return
	}
	referSnapshot := self.pendingSc.rw.getSnapshotBlockByHash(firstUnconfirmedBlock.SnapshotHash)

	// verify account timeout
	if !self.pendingSc.v.verifyAccountTimeout(headSnapshot, referSnapshot) {
		self.Lock()
		defer self.UnLock()
		self.RollbackAccountTo(addr, firstUnconfirmedBlock.Hash, firstUnconfirmedBlock.Height)
	}
}

type recoverStat struct {
	num        int32
	updateTime time.Time
	threshold  int32
}

func (self *recoverStat) reset(t int32) *recoverStat {
	self.num = 0
	self.updateTime = time.Now()
	self.threshold = t
	return self
}

func (self *recoverStat) inc() bool {
	atomic.AddInt32(&self.num, 1)
	now := time.Now()
	if now.Sub(self.updateTime) > time.Second*10 {
		self.updateTime = now
		atomic.StoreInt32(&self.num, 0)
	} else {
		if self.num > self.threshold {
			return false
		}
	}
	return true
}
