package pool

import (
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/consensus"

	"github.com/vitelabs/go-vite/pool/lock"
	"github.com/vitelabs/go-vite/vite/net"

	"github.com/vitelabs/go-vite/pool/batch"

	"github.com/vitelabs/go-vite/pool/tree"

	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
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
	lock.ChainInsert
	AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error
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
		accountV verifier.Verifier,
		cs consensus.Consensus)
}

type commonBlock interface {
	Height() uint64
	Hash() types.Hash
	PrevHash() types.Hash
	checkForkVersion() bool
	resetForkVersion()
	forkVersion() uint64
	Source() types.BlockSource
	Latency() time.Duration
	ShouldFetch() bool
	ReferHashes() ([]types.Hash, []types.Hash, *types.Hash)
}

func newForkBlock(v *common.Version, source types.BlockSource) *forkBlock {
	return &forkBlock{firstV: v.Val(), v: v, source: source, nTime: time.Now()}
}

type forkBlock struct {
	firstV uint64
	v      *common.Version
	source types.BlockSource
	nTime  time.Time
}

func (self *forkBlock) forkVersion() uint64 {
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

func (self *forkBlock) ShouldFetch() bool {
	if self.Source() != types.RemoteBroadcast {
		return true
	}
	if self.Latency() > time.Millisecond*200 {
		return true
	}
	return false
}

func (self *forkBlock) Source() types.BlockSource {
	return self.source
}

type pool struct {
	lock.EasyImpl
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

	version *common.Version

	closed chan struct{}
	wg     sync.WaitGroup

	log log15.Logger

	stat *recoverStat

	addrCache     *lru.Cache
	hashBlacklist Blacklist
	cs            consensus.Consensus
}

func (pl *pool) Snapshot() map[string]interface{} {
	return pl.pendingSc.info()
}
func (pl *pool) SnapshotPendingNum() uint64 {
	return pl.pendingSc.CurrentChain().Size()
}

func (pl *pool) AccountPendingNum() *big.Int {
	result := big.NewInt(0)
	pl.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		size := p.CurrentChain().Size()
		if size > 0 {
			result.Add(result, big.NewInt(0).SetUint64(size))
		}
		return true
	})
	return result
}

func (pl *pool) Account(addr types.Address) map[string]interface{} {
	return pl.selfPendingAc(addr).info()
}

func (pl *pool) SnapshotChainDetail(chainId string) map[string]interface{} {
	return pl.pendingSc.detailChain(chainId)
}

func (pl *pool) AccountChainDetail(addr types.Address, chainId string) map[string]interface{} {
	return pl.selfPendingAc(addr).detailChain(chainId)
}

func NewPool(bc chainDb) (*pool, error) {
	self := &pool{bc: bc, version: &common.Version{}}
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

func (pl *pool) Init(s syncer,
	wt *wallet.Manager,
	snapshotV *verifier.SnapshotVerifier,
	accountV verifier.Verifier,
	cs consensus.Consensus) {
	pl.sync = s
	pl.wt = wt
	rw := &snapshotCh{version: pl.version, bc: pl.bc, log: pl.log}
	fe := &snapshotSyncer{fetcher: s, log: pl.log.New("t", "snapshot")}
	v := &snapshotVerifier{v: snapshotV}
	pl.accountVerifier = accountV
	snapshotPool := newSnapshotPool("snapshotPool", pl.version, v, fe, rw, pl.hashBlacklist, pl.newSnapshotBlockCond, pl.log)
	snapshotPool.init(
		newTools(fe, rw),
		pl)

	pl.cs = cs
	pl.bc.SetConsensus(cs)
	pl.pendingSc = snapshotPool
	pl.stat = (&recoverStat{}).init(10, time.Second*10)
	pl.worker.init()

}
func (pl *pool) Info(addr *types.Address) string {
	if addr == nil {
		bp := pl.pendingSc.blockpool
		cp := pl.pendingSc.chainpool

		freeSize := len(bp.freeBlocks)
		compoundSize := common.SyncMapLen(&bp.compoundBlocks)
		snippetSize := len(cp.snippetChains)
		currentLen := cp.tree.Main().Size()
		treeSize := cp.tree.Size()
		chainSize := len(cp.tree.Branches())
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, treeSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, treeSize, currentLen, chainSize)
	} else {
		ac := pl.selfPendingAc(*addr)
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
		chainSize := len(cp.tree.Branches())
		return fmt.Sprintf("freeSize:%d, compoundSize:%d, snippetSize:%d, treeSize:%d, currentLen:%d, chainSize:%d",
			freeSize, compoundSize, snippetSize, treeSize, currentLen, chainSize)
	}
}
func (pl *pool) AccountBlockInfo(addr types.Address, hash types.Hash) interface{} {
	b, s := pl.selfPendingAc(addr).blockpool.sprint(hash)
	if b != nil {
		sb := b.(*accountPoolBlock)
		return sb.block
	}
	if s != nil {
		return *s
	}
	return nil
}

func (pl *pool) SnapshotBlockInfo(hash types.Hash) interface{} {
	b, s := pl.pendingSc.blockpool.sprint(hash)
	if b != nil {
		sb := b.(*snapshotPoolBlock)
		return sb.block
	}
	if s != nil {
		return *s
	}
	return nil
}

func (pl *pool) Start() {
	pl.log.Info("pool start.")
	defer pl.log.Info("pool started.")
	pl.closed = make(chan struct{})

	pl.accountSubId = pl.sync.SubscribeAccountBlock(pl.AddAccountBlock)
	pl.snapshotSubId = pl.sync.SubscribeSnapshotBlock(pl.AddSnapshotBlock)

	pl.pendingSc.Start()

	pl.newSnapshotBlockCond.Start(time.Millisecond * 30)
	pl.newAccBlockCond.Start(time.Millisecond * 40)
	pl.worker.closed = pl.closed
	common.Go(func() {
		pl.wg.Add(1)
		defer pl.wg.Done()
		pl.worker.work()
	})
}
func (pl *pool) Stop() {
	pl.log.Info("pool stop.")
	defer pl.log.Info("pool stopped.")
	pl.sync.UnsubscribeAccountBlock(pl.accountSubId)
	pl.accountSubId = 0
	pl.sync.UnsubscribeSnapshotBlock(pl.snapshotSubId)
	pl.snapshotSubId = 0

	pl.pendingSc.Stop()
	close(pl.closed)
	pl.newAccBlockCond.Stop()
	pl.newSnapshotBlockCond.Stop()
	pl.wg.Wait()
}

func (pl *pool) AddSnapshotBlock(block *ledger.SnapshotBlock, source types.BlockSource) {

	pl.log.Info("receive snapshot block from network. height:" + strconv.FormatUint(block.Height, 10) + ", hash:" + block.Hash.String() + ".")
	if pl.bc.IsGenesisSnapshotBlock(block.Hash) {
		return
	}

	err := pl.pendingSc.v.verifySnapshotData(block)
	if err != nil {
		pl.log.Error("snapshot error", "err", err, "height", block.Height, "hash", block.Hash)
		return
	}
	pl.pendingSc.addBlock(newSnapshotPoolBlock(block, pl.version, source))

	pl.newSnapshotBlockCond.Broadcast()
	pl.worker.bus.newSBlockEvent()
}

func (pl *pool) AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error {
	defer pl.version.Inc()
	err := pl.pendingSc.v.verifySnapshotData(block)
	if err != nil {
		return err
	}
	cBlock := newSnapshotPoolBlock(block, pl.version, types.Local)
	abs, err := pl.pendingSc.AddDirectBlock(cBlock)
	if err != nil {
		return err
	}
	pl.pendingSc.checkCurrent()
	pl.pendingSc.f.broadcastBlock(block)
	if abs == nil || len(abs) == 0 {
		return nil
	}

	for k, v := range abs {
		err := pl.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
		pl.selfPendingAc(k).checkCurrent()
	}
	return nil
}

func (pl *pool) AddAccountBlock(address types.Address, block *ledger.AccountBlock, source types.BlockSource) {
	pl.log.Info(fmt.Sprintf("receive account block from network. addr:%s, height:%d, hash:%s.", address, block.Height, block.Hash))
	if pl.bc.IsGenesisAccountBlock(block.Hash) {
		return
	}
	ac := pl.selfPendingAc(address)
	ac.addBlock(newAccountPoolBlock(block, nil, pl.version, source))

	pl.newAccBlockCond.Broadcast()
	pl.worker.bus.newABlockEvent()
}

func (pl *pool) AddDirectAccountBlock(address types.Address, block *vm_db.VmAccountBlock) error {
	pl.log.Info(fmt.Sprintf("receive account block from direct. addr:%s, height:%d, hash:%s.", address, block.AccountBlock.Height, block.AccountBlock.Hash))
	defer monitor.LogTime("pool", "addDirectAccount", time.Now())
	pl.RLockInsert()
	defer pl.RUnLockInsert()

	ac := pl.selfPendingAc(address)

	err := ac.v.verifyAccountData(block.AccountBlock)
	if err != nil {
		pl.log.Error("account err", "err", err, "height", block.AccountBlock.Height, "hash", block.AccountBlock.Hash, "addr", address)
		return err
	}

	cBlock := newAccountPoolBlock(block.AccountBlock, block.VmDb, pl.version, types.Local)
	err = ac.AddDirectBlocks(cBlock)
	if err != nil {
		return err
	}
	ac.f.broadcastBlock(block.AccountBlock)
	pl.addrCache.Add(address, time.Now().Add(time.Hour*24))
	return nil

}
func (pl *pool) AddAccountBlocks(address types.Address, blocks []*ledger.AccountBlock, source types.BlockSource) error {
	defer monitor.LogTime("pool", "addAccountArr", time.Now())

	for _, b := range blocks {
		pl.AddAccountBlock(address, b, source)
	}

	pl.newAccBlockCond.Broadcast()
	pl.worker.bus.newABlockEvent()
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

func (pl *pool) ForkAccounts(accounts map[types.Address][]commonBlock) error {

	for k, v := range accounts {
		err := pl.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
		pl.selfPendingAc(k).checkCurrent()
	}
	return nil
}

func (pl *pool) ForkAccountTo(addr types.Address, h *ledger.HashHeight) error {
	this := pl.selfPendingAc(addr)
	this.chainHeadMu.Lock()
	defer this.chainHeadMu.Unlock()
	this.chainTailMu.Lock()
	defer this.chainTailMu.Unlock()

	// find in tree
	targetChain := this.findInTree(h.Hash, h.Height)

	if targetChain == nil {
		pl.log.Info("CurrentModifyToEmpty", "addr", addr, "hash", h.Hash, "height", h.Height,
			"currentId", this.CurrentChain().ID(), "Tail", this.CurrentChain().SprintTail(), "Head", this.CurrentChain().SprintHead())
		err := this.CurrentModifyToEmpty()
		return err
	}
	if targetChain.ID() == this.CurrentChain().ID() {
		return nil
	}
	cu := this.CurrentChain()
	curTailHeight, _ := cu.TailHH()
	keyPoint, _, err := this.chainpool.tree.FindForkPointFromMain(targetChain)
	if err != nil {
		return err
	}
	if keyPoint == nil {
		return errors.Errorf("forkAccountTo key point is nil, target:%s, current:%s, targetTail:%s, currentTail:%s",
			targetChain.ID(), cu.ID(), targetChain.SprintTail(), cu.SprintTail())
	}
	// fork point in disk chain
	if keyPoint.Height() <= curTailHeight {
		pl.log.Info("RollbackAccountTo[2]", "addr", addr, "hash", h.Hash, "height", h.Height, "targetChain", targetChain.ID(),
			"targetChainTail", targetChain.SprintTail(),
			"targetChainHead", targetChain.SprintHead(),
			"keyPoint", keyPoint.Height(),
			"currentId", cu.ID(), "Tail", cu.SprintTail(), "Head", cu.SprintTail())
		err := pl.RollbackAccountTo(addr, keyPoint.Hash(), keyPoint.Height())
		if err != nil {
			return err
		}
	}

	pl.log.Info("ForkAccountTo", "addr", addr, "hash", h.Hash, "height", h.Height, "targetChain", targetChain.ID(),
		"targetChainTail", targetChain.SprintTail(), "targetChainHead", targetChain.SprintHead(),
		"currentId", cu.ID(), "Tail", cu.SprintTail(), "Head", cu.SprintHead())
	err = this.CurrentModifyToChain(targetChain)
	if err != nil {
		return err
	}
	return nil
}

func (pl *pool) RollbackAccountTo(addr types.Address, hash types.Hash, height uint64) error {
	p := pl.selfPendingAc(addr)

	// del some blcoks
	snapshots, accounts, e := p.rw.delToHeight(height)
	if e != nil {
		return e
	}

	// rollback snapshot chain in pool
	err := pl.pendingSc.rollbackCurrent(snapshots)
	if err != nil {
		return err
	}

	pl.pendingSc.checkCurrent()
	// rollback accounts chain in pool
	for k, v := range accounts {
		err = pl.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
		pl.selfPendingAc(k).checkCurrent()
	}
	return err
}

func (pl *pool) selfPendingAc(addr types.Address) *accountPool {
	chain, ok := pl.pendingAc.Load(addr)

	if ok {
		return chain.(*accountPool)
	}

	// lazy load
	rw := &accountCh{address: addr, rw: pl.bc, version: pl.version, log: pl.log.New("account", addr)}
	f := &accountSyncer{address: addr, fetcher: pl.sync, log: pl.log.New()}
	v := &accountVerifier{v: pl.accountVerifier, log: pl.log.New()}
	p := newAccountPool("accountChainPool-"+addr.Hex(), rw, pl.version, pl.hashBlacklist, pl.log)
	p.address = addr
	p.Init(newTools(f, rw), pl, v, f)

	chain, _ = pl.pendingAc.LoadOrStore(addr, p)
	return chain.(*accountPool)
}

func (pl *pool) ReadDownloadedChunks() *net.Chunk {
	chunk := pl.sync.Peek()
	return chunk
}

func (pl *pool) PopDownloadedChunks(hashH ledger.HashHeight) {
	pl.log.Info(fmt.Sprintf("pop chunks[%d-%s]", hashH.Height, hashH.Hash))
	pl.sync.Pop(hashH.Hash)
}

func (pl *pool) loopCompact() {
	pl.wg.Add(1)
	defer pl.wg.Done()

	sum := 0
	for {
		select {
		case <-pl.closed:
			return
		default:
			if sum == 0 {
				pl.newAccBlockCond.Wait()
			}
			sum = 0
			sum += pl.accountsCompact()
		}
	}
}

func (pl *pool) broadcastUnConfirmedBlocks() {
	addrList := pl.listPoolRelAddr()
	// todo all unconfirmed
	for _, addr := range addrList {
		pl.selfPendingAc(addr).broadcastUnConfirmedBlocks()
	}
}

func (pl *pool) delUseLessChains() {
	if pl.sync.SyncState() != net.Syncing {
		pl.pendingSc.loopDelUselessChain()
		var pendings []*accountPool
		pl.pendingAc.Range(func(_, v interface{}) bool {
			p := v.(*accountPool)
			pendings = append(pendings, p)
			return true
		})
		for _, v := range pendings {
			v.loopDelUselessChain()
		}
	}
}

func (pl *pool) listPoolRelAddr() []types.Address {
	var todoAddress []types.Address
	keys := pl.addrCache.Keys()
	now := time.Now()
	for _, k := range keys {
		value, ok := pl.addrCache.Get(k)
		if ok {
			t := value.(time.Time)
			if t.Before(now) {
				pl.addrCache.Remove(k)
			} else {
				todoAddress = append(todoAddress, k.(types.Address))
			}
		}
	}
	return todoAddress
}
func (pl *pool) compact() int {
	sum := 0
	sum += pl.accountsCompact()
	sum += pl.pendingSc.loopCompactSnapshot()
	return sum
}
func (pl *pool) snapshotCompact() int {
	return pl.pendingSc.loopCompactSnapshot()
}

func (pl *pool) accountsCompact() int {
	sum := 0
	var pendings []*accountPool
	pl.pendingAc.Range(func(_, v interface{}) bool {
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
func (pl *pool) fetchForTask(task verifyTask) {
	reqs := task.requests()
	if len(reqs) <= 0 {
		return
	}
	// if something in pool, deal with it.
	for _, r := range reqs {
		exist := false
		if r.snapshot {
			exist = pl.pendingSc.existInPool(r.hash)
		} else {
			if r.chain != nil {
				exist = pl.selfPendingAc(*r.chain).existInPool(r.hash)
			}
		}
		if exist {
			pl.log.Info(fmt.Sprintf("block[%s] exist, should not fetch.", r.String()))
			continue
		}

		if r.snapshot {
			pl.pendingSc.f.fetchByHash(r.hash, 5)
		} else {
			// todo
			pl.sync.FetchAccountBlocks(r.hash, 5, r.chain)
		}
	}
	return
}

func (pl *pool) checkBlock(block *snapshotPoolBlock) bool {
	fail := block.failStat.isFail()
	if fail {
		return false
	}
	if pl.hashBlacklist.Exists(block.Hash()) {
		return false
	}
	var result = true
	for k, v := range block.block.SnapshotContent {
		ac := pl.selfPendingAc(k)
		if ac.findInPool(v.Hash, v.Height) {
			continue
		}
		fc := ac.findInTreeDisk(v.Hash, v.Height, true)
		if fc == nil {
			result = false
			if block.ShouldFetch() {
				ac.f.fetchBySnapshot(ledger.HashHeight{Hash: v.Hash, Height: v.Height}, k, 1, block.Height(), block.Hash())
			}
		}
	}
	return result
}

func (pl *pool) realSnapshotHeight(fc tree.Branch) uint64 {
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
			block.checkResult = pl.checkBlock(block)
		}

		if !block.checkResult {
			return h
		}
		h = h + 1
	}
}

func (pl *pool) fetchForSnapshot(fc tree.Branch) error {
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

		if !sb.ShouldFetch() {
			continue
		}
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
		ac := pl.selfPendingAc(*v.chain)
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
func (pl *pool) snapshotPendingFix(p batch.Batch, snapshot *ledger.HashHeight, pending *snapshotPending) {
	if pending.snapshot != nil && pending.snapshot.ShouldFetch() {
		pl.fetchAccounts(pending.addrM, snapshot.Height, snapshot.Hash)
	}
	pl.LockInsert()
	defer pl.UnLockInsert()
	if p.Version() != pl.version.Val() {
		pl.log.Warn("new version happened.")
		return
	}

	accounts := make(map[types.Address]*ledger.HashHeight)
	for k, account := range pending.addrM {
		pl.log.Debug("db for account.", "addr", k.String(), "height", account.Height, "hash", account.Hash, "sbHash", snapshot.Hash, "sbHeight", snapshot.Height)
		this := pl.selfPendingAc(k)
		hashH, e := this.pendingAccountTo(account, account.Height)
		if e != nil {
			pl.log.Error("db for account fail.", "err", e, "address", k, "hashH", account)
		}
		if hashH != nil {
			accounts[k] = account
		}
	}

	if len(accounts) > 0 {
		monitor.LogEventNum("pool", "snapshotPendingFork", len(accounts))
		pl.forkAccountsFor(accounts, snapshot)
	}
}

func (pl *pool) fetchAccounts(accounts map[types.Address]*ledger.HashHeight, sHeight uint64, sHash types.Hash) {
	for addr, hashH := range accounts {
		ac := pl.selfPendingAc(addr)
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

func (pl *pool) forkAccountsFor(accounts map[types.Address]*ledger.HashHeight, snapshot *ledger.HashHeight) {
	for k, v := range accounts {
		pl.log.Debug("forkAccounts", "Addr", k.String(), "Height", v.Height, "Hash", v.Hash)
		err := pl.ForkAccountTo(k, v)
		if err != nil {
			pl.log.Error("forkaccountTo err", "err", err)
			time.Sleep(time.Second)
			// todo
			panic(errors.Errorf("snapshot:%s-%d", snapshot.Hash, snapshot.Height))
		}
	}

	pl.version.Inc()
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
