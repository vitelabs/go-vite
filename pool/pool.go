package pool

import (
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/pool/batch"

	"github.com/vitelabs/go-vite/pool/tree"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_db"
	"github.com/vitelabs/go-vite/wallet"
)

type Writer interface {
	// for normal account
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_db.VmAccountBlock) error

	// for contract account
	//AddDirectAccountBlocks(address types.Address, received *vm_db.VmAccountBlock, sendBlocks []*vm_db.VmAccountBlock) error
}

type SnapshotProducerWriter interface {
	Lock()

	UnLock()

	AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error

	RollbackAccountTo(addr types.Address, hash types.Hash, height uint64) error
}

type Reader interface {
}
type Debug interface {
	Info(addr *types.Address) string
	AccountBlockInfo(addr types.Address, hash types.Hash) interface{}
	SnapshotBlockInfo(hash types.Hash) interface{}
	Snapshot() map[string]interface{}
	SnapshotPendingNum() uint64
	AccountPendingNum() *big.Int
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
		accountV verifier.Verifier)
}

type commonBlock interface {
	Height() uint64
	Hash() types.Hash
	PrevHash() types.Hash
	checkForkVersion() bool
	resetForkVersion()
	forkVersion() int
	Source() types.BlockSource
	Latency() time.Duration
	ReferHashes() ([]types.Hash, []types.Hash, *types.Hash)
}

func newForkBlock(v *ForkVersion, source types.BlockSource) *forkBlock {
	return &forkBlock{firstV: v.Val(), v: v, source: source, nTime: time.Now()}
}

type forkBlock struct {
	firstV int
	v      *ForkVersion
	source types.BlockSource
	nTime  time.Time
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
func (self *forkBlock) Latency() time.Duration {
	if self.Source() == types.RemoteBroadcast || self.Source() == types.RemoteFetch {
		return time.Now().Sub(self.nTime)
	}
	return time.Duration(0)
}

func (self *forkBlock) Source() types.BlockSource {
	return self.source
}

type pool struct {
	pendingSc *snapshotPool
	pendingAc sync.Map // key:address v:*accountPool

	sync syncer
	bc   chainDb
	wt   *wallet.Manager

	snapshotVerifier *verifier.SnapshotVerifier
	accountVerifier  verifier.Verifier

	accountSubId  int
	snapshotSubId int

	newAccBlockCond      *common.CondTimer
	newSnapshotBlockCond *common.CondTimer
	worker               *worker

	rwMutex sync.RWMutex
	version *ForkVersion

	closed chan struct{}
	wg     sync.WaitGroup

	log log15.Logger

	stat *recoverStat

	addrCache     *lru.Cache
	hashBlacklist Blacklist
}

func (self *pool) Snapshot() map[string]interface{} {
	return self.pendingSc.info()
}
func (self *pool) SnapshotPendingNum() uint64 {
	return self.pendingSc.CurrentChain().Size()
}

func (self *pool) AccountPendingNum() *big.Int {
	result := big.NewInt(0)
	self.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		size := p.CurrentChain().Size()
		if size > 0 {
			result.Add(result, big.NewInt(0).SetUint64(size))
		}
		return true
	})
	return result
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

func NewPool(bc chainDb) (*pool, error) {
	self := &pool{bc: bc, rwMutex: sync.RWMutex{}, version: &ForkVersion{}}
	self.log = log15.New("module", "pool")
	cache, err := lru.New(1024)
	if err != nil {
		panic(err)
	}
	self.addrCache = cache

	self.hashBlacklist, err = NewBlacklist()
	self.newAccBlockCond = common.NewCondTimer()
	self.newSnapshotBlockCond = common.NewCondTimer()
	if err != nil {
		return nil, err
	}
	self.worker = &worker{p: self}
	return self, nil
}

func (self *pool) Init(s syncer,
	wt *wallet.Manager,
	snapshotV *verifier.SnapshotVerifier,
	accountV verifier.Verifier) {
	self.sync = s
	self.wt = wt
	rw := &snapshotCh{version: self.version, bc: self.bc, log: self.log}
	fe := &snapshotSyncer{fetcher: s, log: self.log.New("t", "snapshot")}
	v := &snapshotVerifier{v: snapshotV}
	self.accountVerifier = accountV
	snapshotPool := newSnapshotPool("snapshotPool", self.version, v, fe, rw, self.hashBlacklist, self.newSnapshotBlockCond, self.log)
	snapshotPool.init(
		newTools(fe, rw),
		self)

	self.pendingSc = snapshotPool
	self.stat = (&recoverStat{}).init(10, time.Second*10)
	self.worker.init()

}
func (self *pool) Info(addr *types.Address) string {
	if addr == nil {
		bp := self.pendingSc.blockpool
		cp := self.pendingSc.chainpool

		freeSize := len(bp.freeBlocks)
		compoundSize := common.SyncMapLen(&bp.compoundBlocks)
		snippetSize := len(cp.snippetChains)
		currentLen := cp.tree.Main().Size()
		treeSize := cp.tree.Size()
		chainSize := cp.size()
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, treeSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, treeSize, currentLen, chainSize)
	} else {
		ac := self.selfPendingAc(*addr)
		if ac == nil {
			return "pool not exist."
		}
		bp := ac.blockpool
		cp := ac.chainpool

		freeSize := len(bp.freeBlocks)
		compoundSize := common.SyncMapLen(&bp.compoundBlocks)
		snippetSize := len(cp.snippetChains)
		treeSize := cp.tree.Size()
		currentLen := cp.tree.Main().Size()
		chainSize := cp.size()
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, treeSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, treeSize, currentLen, chainSize)
	}
}
func (self *pool) AccountBlockInfo(addr types.Address, hash types.Hash) interface{} {
	b, s := self.selfPendingAc(addr).blockpool.sprint(hash)
	if b != nil {
		sb := b.(*accountPoolBlock)
		return sb.block
	}
	if s != nil {
		return *s
	}
	return nil
}

func (self *pool) SnapshotBlockInfo(hash types.Hash) interface{} {
	b, s := self.pendingSc.blockpool.sprint(hash)
	if b != nil {
		sb := b.(*snapshotPoolBlock)
		return sb.block
	}
	if s != nil {
		return *s
	}
	return nil
}

func (self *pool) Start() {
	self.log.Info("pool start.")
	defer self.log.Info("pool started.")
	self.closed = make(chan struct{})

	self.accountSubId = self.sync.SubscribeAccountBlock(self.AddAccountBlock)
	self.snapshotSubId = self.sync.SubscribeSnapshotBlock(self.AddSnapshotBlock)

	self.pendingSc.Start()

	self.newSnapshotBlockCond.Start(time.Millisecond * 30)
	self.newAccBlockCond.Start(time.Millisecond * 40)
	self.worker.closed = self.closed
	common.Go(func() {
		self.wg.Add(1)
		defer self.wg.Done()
		self.worker.work()
	})
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
	self.newAccBlockCond.Stop()
	self.newSnapshotBlockCond.Stop()
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

func (self *pool) AddSnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) {

	self.log.Info("receive snapshot block from network. height:" + strconv.FormatUint(block.Height, 10) + ", hash:" + block.Hash.String() + ".")
	if self.bc.IsGenesisSnapshotBlock(block.Hash) {
		return
	}

	err := self.pendingSc.v.verifySnapshotData(block)
	if err != nil {
		self.log.Error("snapshot error", "err", err, "height", block.Height, "hash", block.Hash)
		return
	}
	self.pendingSc.AddBlock(newSnapshotPoolBlock(block, self.version, source))

	self.newSnapshotBlockCond.Broadcast()
	self.worker.bus.newSBlockEvent()
}

func (self *pool) AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error {
	defer self.version.Inc()
	err := self.pendingSc.v.verifySnapshotData(block)
	if err != nil {
		return err
	}
	cBlock := newSnapshotPoolBlock(block, self.version, types.Local)
	abs, err := self.pendingSc.AddDirectBlock(cBlock)
	if err != nil {
		return err
	}
	self.pendingSc.checkCurrent()
	self.pendingSc.f.broadcastBlock(block)
	if abs == nil || len(abs) == 0 {
		return nil
	}

	for k, v := range abs {
		err := self.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
		self.selfPendingAc(k).checkCurrent()
	}
	return nil
}

func (self *pool) AddAccountBlock(address types.Address, block *ledger.AccountBlock, source types.BlockSource) {
	self.log.Info(fmt.Sprintf("receive account block from network. addr:%s, height:%d, hash:%s.", address, block.Height, block.Hash))
	if self.bc.IsGenesisAccountBlock(block.Hash) {
		return
	}
	ac := self.selfPendingAc(address)
	err := ac.v.verifyAccountData(block)
	if err != nil {
		self.log.Error("account err", "err", err, "height", block.Height, "hash", block.Hash, "addr", address)
		return
	}
	ac.AddBlock(newAccountPoolBlock(block, nil, self.version, source))

	self.newAccBlockCond.Broadcast()
	self.worker.bus.newABlockEvent()
}

func (self *pool) AddDirectAccountBlock(address types.Address, block *vm_db.VmAccountBlock) error {
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

	cBlock := newAccountPoolBlock(block.AccountBlock, block.VmDb, self.version, types.Local)
	err = ac.AddDirectBlocks(cBlock)
	if err != nil {
		return err
	}
	ac.f.broadcastBlock(block.AccountBlock)
	self.addrCache.Add(address, time.Now().Add(time.Hour*24))
	return nil

}
func (self *pool) AddAccountBlocks(address types.Address, blocks []*ledger.AccountBlock, source types.BlockSource) error {
	defer monitor.LogTime("pool", "addAccountArr", time.Now())

	for _, b := range blocks {
		self.AddAccountBlock(address, b, source)
	}

	self.newAccBlockCond.Broadcast()
	self.worker.bus.newABlockEvent()
	return nil
}

//func (self *pool) AddDirectAccountBlocks(address types.Address, received *vm_db.VmAccountBlock, sendBlocks []*vm_db.VmAccountBlock) error {
//	self.log.Info(fmt.Sprintf("receive account blocks from direct. addr:%s, height:%d, hash:%s.", address, received.AccountBlock.Height, received.AccountBlock.Hash))
//	defer monitor.LogTime("pool", "addDirectAccountArr", time.Now())
//	self.RLock()
//	defer self.RUnLock()
//	ac := self.selfPendingAc(address)
//	// todo
//	var accountPoolBlocks []*accountPoolBlock
//	for _, v := range sendBlocks {
//		accountPoolBlocks = append(accountPoolBlocks, newAccountPoolBlock(v.AccountBlock, v.VmDb, self.version, types.Local))
//	}
//	err := ac.AddDirectBlocks(newAccountPoolBlock(received.AccountBlock, received.VmDb, self.version, types.Local), accountPoolBlocks)
//	if err != nil {
//		return err
//	}
//	ac.f.broadcastReceivedBlocks(received, sendBlocks)
//
//	self.addrCache.Add(address, time.Now().Add(time.Hour*24))
//	return nil
//}

func (self *pool) ForkAccounts(accounts map[types.Address][]commonBlock) error {

	for k, v := range accounts {
		err := self.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
		self.selfPendingAc(k).checkCurrent()
	}
	return nil
}

func (self *pool) ForkAccountTo(addr types.Address, h *ledger.HashHeight) error {
	this := self.selfPendingAc(addr)
	this.chainHeadMu.Lock()
	defer this.chainHeadMu.Unlock()
	this.chainTailMu.Lock()
	defer this.chainTailMu.Unlock()

	// find in tree
	targetChain := this.findInTree(h.Hash, h.Height)

	if targetChain == nil {
		self.log.Info("CurrentModifyToEmpty", "addr", addr, "hash", h.Hash, "height", h.Height,
			"currentId", this.CurrentChain().Id(), "Tail", this.CurrentChain().SprintTail(), "Head", this.CurrentChain().SprintHead())
		err := this.CurrentModifyToEmpty()
		return err
	}
	if targetChain.Id() == this.CurrentChain().Id() {
		return nil
	}
	cu := this.CurrentChain()
	curTailHeight, _ := cu.TailHH()
	keyPoint, forkPoint, err := this.chainpool.tree.FindForkPointFromMain(targetChain)
	if err != nil {
		return err
	}
	if keyPoint == nil {
		return errors.Errorf("forkAccountTo key point is nil, target:%s, current:%s, targetTail:%s, currentTail:%s",
			targetChain.Id(), cu.Id(), targetChain.SprintTail(), cu.SprintTail())
	}
	// fork point in disk chain
	if forkPoint.Height() <= curTailHeight {
		self.log.Info("RollbackAccountTo[2]", "addr", addr, "hash", h.Hash, "height", h.Height, "targetChain", targetChain.Id(),
			"targetChainTail", targetChain.SprintTail(),
			"targetChainHead", targetChain.SprintHead(),
			"keyPoint", keyPoint.Height(),
			"currentId", cu.Id(), "Tail", cu.SprintTail(), "Head", cu.SprintTail())
		err := self.RollbackAccountTo(addr, keyPoint.Hash(), keyPoint.Height())
		if err != nil {
			return err
		}
	}

	self.log.Info("ForkAccountTo", "addr", addr, "hash", h.Hash, "height", h.Height, "targetChain", targetChain.Id(),
		"targetChainTail", targetChain.SprintTail(), "targetChainHead", targetChain.SprintHead(),
		"currentId", cu.Id(), "Tail", cu.SprintTail(), "Head", cu.SprintHead())
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

	self.pendingSc.checkCurrent()
	// rollback accounts chain in pool
	for k, v := range accounts {
		err = self.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
		self.selfPendingAc(k).checkCurrent()
	}
	return err
}

func (self *pool) selfPendingAc(addr types.Address) *accountPool {
	chain, ok := self.pendingAc.Load(addr)

	if ok {
		return chain.(*accountPool)
	}

	// lazy load
	rw := &accountCh{address: addr, rw: self.bc, version: self.version, log: self.log.New("account", addr)}
	f := &accountSyncer{address: addr, fetcher: self.sync, log: self.log.New()}
	v := &accountVerifier{v: self.accountVerifier, log: self.log.New()}
	p := newAccountPool("accountChainPool-"+addr.Hex(), rw, self.version, self.hashBlacklist, self.log)
	p.address = addr
	p.Init(newTools(f, rw), self, v, f)

	chain, _ = self.pendingAc.LoadOrStore(addr, p)
	return chain.(*accountPool)
}

//func (self *pool) loopTryInsert() {
//	defer self.poolRecover()
//	self.wg.Add(1)
//	defer self.wg.Done()
//
//	t := time.NewTicker(time.Millisecond * 100)
//	t2 := time.NewTicker(time.Millisecond * 40)
//	defer t.Stop()
//	sum := 0
//	for {
//		select {
//		case <-self.closed:
//			return
//		case <-t.C:
//			if sum == 0 {
//				time.Sleep(100 * time.Millisecond)
//				monitor.LogEvent("pool", "tryInsertSleep100")
//			}
//			sum = 0
//			sum += self.accountsTryInsert()
//		case <-t2.C:
//			if sum == 0 {
//				time.Sleep(20 * time.Millisecond)
//				monitor.LogEvent("pool", "tryInsertSleep20")
//			}
//			sum = 0
//			sum += self.accountsTryInsert()
//		default:
//			sum += self.accountsTryInsert()
//		}
//	}
//}

//func (self *pool) accountsTryInsert() int {
//	monitor.LogEvent("pool", "tryInsert")
//	sum := 0
//	var db []*accountPool
//	self.pendingAc.Range(func(_, v interface{}) bool {
//		p := v.(*accountPool)
//		db = append(db, p)
//		return true
//	})
//	var tasks []verifyTask
//	for _, p := range db {
//		task := p.TryInsert()
//		if task != nil {
//			self.fetchForTask(task)
//			tasks = append(tasks, task)
//			sum = sum + 1
//		}
//	}
//	return sum
//}

func (self *pool) loopCompact() {
	defer self.poolRecover()
	self.wg.Add(1)
	defer self.wg.Done()

	sum := 0
	for {
		select {
		case <-self.closed:
			return
		default:
			if sum == 0 {
				self.newAccBlockCond.Wait()
			}
			sum = 0
			sum += self.accountsCompact()
		}
	}
}
func (self *pool) poolRecover() {
	//if err := recover(); err != nil {
	//	var e error
	//	switch t := err.(type) {
	//	case error:
	//		e = errors.WithStack(t)
	//	case string:
	//		e = errors.New(t)
	//	default:
	//		e = errors.Errorf("unknown type, %+v", err)
	//	}
	//
	//	self.log.Error("panic", "err", err, "withstack", fmt.Sprintf("%+v", e))
	//	fmt.Printf("%+v", e)
	//	if self.stat.inc() {
	//		common.Go(self.Restart)
	//	} else {
	//		panic(e)
	//	}
	//}
}

//func (self *pool) loopBroadcastAndDel() {
//	defer self.poolRecover()
//	self.wg.Add(1)
//	defer self.wg.Done()
//
//	broadcastT := time.NewTicker(time.Second * 30)
//	delUselessChainT := time.NewTicker(time.Minute)
//
//	defer broadcastT.Stop()
//	for {
//		select {
//		case <-self.closed:
//			return
//		case <-broadcastT.C:
//			addrList := self.listPoolRelAddr()
//			// todo all unconfirmed
//			for _, addr := range addrList {
//				self.selfPendingAc(addr).broadcastUnConfirmedBlocks()
//			}
//		case <-delUselessChainT.C:
//			// del some useless chain in pool
//			self.delUseLessChains()
//		}
//	}
//}

func (self *pool) broadcastUnConfirmedBlocks() {
	addrList := self.listPoolRelAddr()
	// todo all unconfirmed
	for _, addr := range addrList {
		self.selfPendingAc(addr).broadcastUnConfirmedBlocks()
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

func (self *pool) listPoolRelAddr() []types.Address {
	var todoAddress []types.Address
	keys := self.addrCache.Keys()
	now := time.Now()
	for _, k := range keys {
		value, ok := self.addrCache.Get(k)
		if ok {
			t := value.(time.Time)
			if t.Before(now) {
				self.addrCache.Remove(k)
			} else {
				todoAddress = append(todoAddress, k.(types.Address))
			}
		}
	}
	return todoAddress
}
func (self *pool) compact() int {
	sum := 0
	sum += self.accountsCompact()
	sum += self.pendingSc.loopCompactSnapshot()
	return sum
}
func (self *pool) snapshotCompact() int {
	return self.pendingSc.loopCompactSnapshot()
}

func (self *pool) accountsCompact() int {
	sum := 0
	var pendings []*accountPool
	self.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		pendings = append(pendings, p)
		return true
	})
	if len(pendings) > 0 {
		monitor.LogEventNum("pool", "AccountsCompact", len(pendings))
		for _, p := range pendings {
			sum = sum + p.Compact()
		}
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
	//self.log.Debug("try to delete timeout unconfirmed blocks.", "addr", addr)
	//headSnapshot := self.pendingSc.rw.headSnapshot()
	//ac := self.selfPendingAc(addr)
	//firstUnconfirmedBlock := ac.rw.getFirstUnconfirmedBlock(headSnapshot)
	//if firstUnconfirmedBlock == nil {
	//	return
	//}
	//self.log.Debug("account block unconfirmed.", "acc", addr, "hash", firstUnconfirmedBlock.Hash, "height", firstUnconfirmedBlock.Height)
	//referSnapshot := self.pendingSc.rw.getSnapshotBlockByHash(firstUnconfirmedBlock.SnapshotHash)
	//
	//// verify account timeout
	//if !self.pendingSc.v.verifyAccountTimeout(headSnapshot, referSnapshot) {
	//	self.log.Info("account block timeout, rollback", "hash", firstUnconfirmedBlock.Hash, "height", firstUnconfirmedBlock.Height)
	//	self.Lock()
	//	defer self.Unlock()
	//	err := self.RollbackAccountTo(addr, firstUnconfirmedBlock.Hash, firstUnconfirmedBlock.Height)
	//	if err != nil {
	//		self.log.Error("rollback account fail.", "err", err)
	//	} else {
	//		self.selfPendingAc(addr).CurrentModifyToEmpty()
	//	}
	//}
}

func (self *pool) checkBlock(block *snapshotPoolBlock) bool {
	fail := block.failStat.isFail()
	if fail {
		return false
	}
	var result = true
	for k, v := range block.block.SnapshotContent {
		ac := self.selfPendingAc(k)
		if ac.findInPool(v.Hash, v.Height) {
			continue
		}
		fc := ac.findInTreeDisk(v.Hash, v.Height, true)
		if fc == nil {
			ac.f.fetchBySnapshot(ledger.HashHeight{Hash: v.Hash, Height: v.Height}, k, 1, block.Height(), block.Hash())
			result = false
		}
	}
	return result
}

func (self *pool) realSnapshotHeight(fc tree.Branch) uint64 {
	h, _ := fc.TailHH()
	for {
		b := fc.GetKnot(h+1, false)
		if b == nil {
			return h
		}
		block := b.(*snapshotPoolBlock)
		now := time.Now()
		if now.After(block.lastCheckTime.Add(time.Second * 5)) {
			block.lastCheckTime = now
			block.checkResult = self.checkBlock(block)
		}

		if !block.checkResult {
			return h
		}
		h = h + 1
	}
}

func (self *pool) fetchForSnapshot(fc tree.Branch) error {
	var reqs []*fetchRequest
	j := 0
	tailHeight, _ := fc.TailHH()
	headHeight, headHash := fc.HeadHH()
	addrM := make(map[types.Address]*ledger.HashHeight)
	for i := tailHeight + 1; i <= headHeight; i++ {
		j++
		b := fc.GetKnot(i, false)
		if b == nil {
			continue
		}

		sb := b.(*snapshotPoolBlock)

		for k, v := range sb.block.SnapshotContent {

			hh, ok := addrM[k]
			if ok {
				if hh.Height < v.Height {
					hh.Hash = v.Hash
					hh.Height = v.Height
				}
			} else {
				vv := *v
				addrM[k] = &vv
			}

		}
	}

	for k, v := range addrM {
		addr := k
		reqs = append(reqs, &fetchRequest{
			snapshot:       false,
			chain:          &addr,
			hash:           v.Hash,
			accHeight:      v.Height,
			prevCnt:        1,
			snapshotHash:   &headHash,
			snapshotHeight: headHeight,
		})
	}

	for _, v := range reqs {
		if v.chain == nil {
			continue
		}
		ac := self.selfPendingAc(*v.chain)
		if ac.findInPool(v.hash, v.accHeight) {
			continue
		}
		fc := ac.findInTreeDisk(v.hash, v.accHeight, true)
		if fc == nil {
			ac.f.fetchBySnapshot(ledger.HashHeight{Hash: v.hash, Height: v.accHeight}, *v.chain, 1, v.snapshotHeight, *v.snapshotHash)
		}
	}
	return nil
}
func (self *pool) insertLevel(p batch.Batch, l batch.Level, version int) error {
	if l.Snapshot() {
		return self.insertSnapshotLevel(p, l, version)
	} else {
		return self.insertAccountLevel(p, l, version)
	}
}
func (self *pool) insertSnapshotLevel(p batch.Batch, l batch.Level, version int) error {
	t1 := time.Now()
	num := 0
	defer func() {
		sub := time.Now().Sub(t1)
		levelInfo := fmt.Sprintf("\tlevel[%d][%d][%s][%d]->%dS", l.Index(), (int64(num)*time.Second.Nanoseconds())/sub.Nanoseconds(), sub, num, num)
		fmt.Println(levelInfo)
	}()
	for _, b := range l.Buckets() {
		num = num + len(b.Items())
		return self.insertSnapshotBucket(p, b, version)
	}
	return nil
}

var MAX_PARALLEL = 5

func (self *pool) insertAccountLevel(p batch.Batch, l batch.Level, version int) error {
	bs := l.Buckets()
	lenBs := len(bs)
	if lenBs == 0 {
		return nil
	}

	N := helper.MinInt(lenBs, MAX_PARALLEL)
	bucketCh := make(chan batch.Bucket, lenBs)

	var wg sync.WaitGroup
	wg.Add(N)

	var num int32
	t1 := time.Now()
	var globalErr error
	for i := 0; i < N; i++ {
		common.Go(func() {
			defer wg.Done()
			for b := range bucketCh {
				if globalErr != nil {
					return
				}
				err := self.insertAccountBucket(p, b, version)
				atomic.AddInt32(&num, int32(len(b.Items())))
				if err != nil {
					globalErr = err
					fmt.Printf("error[%s] for insert account block.\n", err)
					return
				}
			}
		})
	}
	levelInfo := ""
	for _, bucket := range bs {
		levelInfo += "|" + strconv.Itoa(len(bucket.Items()))
		if bucket.Owner() == nil {
			levelInfo += "S"
		}

		bucketCh <- bucket
	}
	close(bucketCh)
	wg.Wait()
	sub := time.Now().Sub(t1)
	levelInfo = fmt.Sprintf("\tlevel[%d][%d][%s][%d]->%s, %s", l.Index(), (int64(num)*time.Second.Nanoseconds())/sub.Nanoseconds(), sub, num, levelInfo, globalErr)
	fmt.Println(levelInfo)

	if globalErr != nil {
		return globalErr
	}
	return nil
}
func (self *pool) snapshotPendingFix(p batch.Batch, snapshot *ledger.HashHeight, pending *snapshotPending) {
	self.fetchAccounts(pending.addrM, snapshot.Height, snapshot.Hash)

	self.Lock()
	defer self.UnLock()
	if p.Version() != self.version.Val() {
		self.log.Warn("new version happened.")
		return
	}

	accounts := make(map[types.Address]*ledger.HashHeight)
	for k, account := range pending.addrM {
		self.log.Debug("db for account.", "addr", k.String(), "height", account.Height, "hash", account.Hash, "sbHash", snapshot.Hash, "sbHeight", snapshot.Height)
		this := self.selfPendingAc(k)
		hashH, e := this.pendingAccountTo(account, account.Height)
		if e != nil {
			self.log.Error("db for account fail.", "err", e, "address", k, "hashH", account)
		}
		if hashH != nil {
			accounts[k] = account
		}
	}

	if len(accounts) > 0 {
		monitor.LogEventNum("pool", "snapshotPendingFork", len(accounts))
		self.forkAccountsFor(accounts, snapshot)
	}
}

func (self *pool) fetchAccounts(accounts map[types.Address]*ledger.HashHeight, sHeight uint64, sHash types.Hash) {
	for addr, hashH := range accounts {
		ac := self.selfPendingAc(addr)
		if !ac.existInPool(hashH.Hash) {
			head, _ := ac.chainpool.diskChain.HeadHH()
			u := uint64(10)
			if hashH.Height > head {
				u = hashH.Height - head
			}
			ac.f.fetchBySnapshot(*hashH, addr, u, sHeight, sHash)
		}
	}

}

func (self *pool) forkAccountsFor(accounts map[types.Address]*ledger.HashHeight, snapshot *ledger.HashHeight) {
	for k, v := range accounts {
		self.log.Debug("forkAccounts", "Addr", k.String(), "Height", v.Height, "Hash", v.Hash)
		err := self.ForkAccountTo(k, v)
		if err != nil {
			self.log.Error("forkaccountTo err", "err", err)
			time.Sleep(time.Second)
			// todo
			panic(errors.Errorf("snapshot:%s-%d", snapshot.Hash, snapshot.Height))
		}
	}

	self.version.Inc()
}

type recoverStat struct {
	num           int32
	updateTime    time.Time
	threshold     int32
	timeThreshold time.Duration
}
type failStat struct {
	first         *time.Time
	update        *time.Time
	timeThreshold time.Duration
}

func (self *failStat) init(d time.Duration) *failStat {
	self.timeThreshold = d
	return self
}
func (self *failStat) inc() bool {
	update := self.update
	if update != nil {
		if time.Now().Sub(*update) > self.timeThreshold {
			self.clear()
			return false
		}
	}
	if self.first == nil {
		now := time.Now()
		self.first = &now
	}
	now := time.Now()
	self.update = &now

	if self.update.Sub(*self.first) > self.timeThreshold {
		return false
	}
	return true
}

func (self *failStat) isFail() bool {
	first := self.first
	if first == nil {
		return false
	}
	update := self.update
	if update == nil {
		return false
	}

	if time.Now().Sub(*update) > 10*self.timeThreshold {
		self.clear()
		return false
	}

	if update.Sub(*first) > self.timeThreshold {
		return true
	}
	return false
}

func (self *failStat) clear() {
	self.first = nil
	self.update = nil
}

func (self *recoverStat) init(t int32, d time.Duration) *recoverStat {
	self.num = 0
	self.updateTime = time.Now()
	self.threshold = t
	self.timeThreshold = d
	return self
}

func (self *recoverStat) reset() *recoverStat {
	self.num = 0
	self.updateTime = time.Now()
	return self
}

func (self *recoverStat) inc() bool {
	atomic.AddInt32(&self.num, 1)
	now := time.Now()
	if now.Sub(self.updateTime) > self.timeThreshold {
		self.updateTime = now
		atomic.StoreInt32(&self.num, 0)
	} else {
		if self.num > self.threshold {
			return false
		}
	}
	return true
}

/**

about the lock:

# lock in pool

this is a rw lock.

Write Lock: pool.Lock() <-> pool.UnLock()
Read Lock: pool.RLock() <-> pool.RUnLock()

the scope of the lock is the ledger in pool.

the account's tailHash modification and the snapshot's tailHash modification,

these two things will not happen at the same time.

*/
