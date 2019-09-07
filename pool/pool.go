package pool

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/pool/batch"
	"github.com/vitelabs/go-vite/pool/lock"
	"github.com/vitelabs/go-vite/pool/tree"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_db"
	"github.com/vitelabs/go-vite/wallet"
)

// Writer is a writer of BlockPool
type Writer interface {
	// for normal account
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_db.VmAccountBlock) error

	// for contract account
	//AddDirectAccountBlocks(address types.Address, received *vm_db.VmAccountBlock, sendBlocks []*vm_db.VmAccountBlock) error
}

// SnapshotProducerWriter is a writer for snapshot producer
type SnapshotProducerWriter interface {
	lock.ChainInsert
	lock.ChainRollback
	AddDirectSnapshotBlock(block *ledger.SnapshotBlock) error
}

// Reader is a reader of BlockPool
type Reader interface {
	GetIrreversibleBlock() *ledger.SnapshotBlock
}

// Debug provide more detail info for BlockPool
type Debug interface {
	Info() map[string]interface{}
	AccountBlockInfo(addr types.Address, hash types.Hash) interface{}
	SnapshotBlockInfo(hash types.Hash) interface{}
	Snapshot() map[string]interface{}
	SnapshotPendingNum() uint64
	AccountPendingNum() *big.Int
	Account(addr types.Address) map[string]interface{}
	SnapshotChainDetail(chainID string, height uint64) map[string]interface{}
	AccountChainDetail(addr types.Address, chainID string, height uint64) map[string]interface{}
}

// BlockPool is responsible for organizing blocks and inserting it into the chain
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

func (fb *forkBlock) forkVersion() uint64 {
	return fb.v.Val()
}
func (fb *forkBlock) checkForkVersion() bool {
	return fb.firstV == fb.v.Val()
}
func (fb *forkBlock) resetForkVersion() {
	val := fb.v.Val()
	fb.firstV = val
}
func (fb *forkBlock) Latency() time.Duration {
	if fb.Source() == types.RemoteBroadcast || fb.Source() == types.RemoteFetch {
		return time.Now().Sub(fb.nTime)
	}
	return time.Duration(0)
}

func (fb *forkBlock) ShouldFetch() bool {
	if fb.Source() != types.RemoteBroadcast {
		return true
	}
	if fb.Latency() > time.Millisecond*200 {
		return true
	}
	return false
}

func (fb *forkBlock) Source() types.BlockSource {
	return fb.source
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

	accountSubID  int
	snapshotSubID int

	newAccBlockCond      *common.CondTimer
	newSnapshotBlockCond *common.CondTimer
	worker               *worker

	version *common.Version

	rollbackVersion *common.Version

	closed chan struct{}
	wg     sync.WaitGroup

	log log15.Logger

	stat *recoverStat

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

func (pl *pool) SnapshotChainDetail(chainID string, height uint64) map[string]interface{} {
	return pl.pendingSc.detailChain(chainID, height)
}

func (pl *pool) AccountChainDetail(addr types.Address, chainID string, height uint64) map[string]interface{} {
	return pl.selfPendingAc(addr).detailChain(chainID, height)
}

// NewPool create a new BlockPool
func NewPool(bc chainDb) (BlockPool, error) {
	self := &pool{bc: bc, version: &common.Version{}, rollbackVersion: &common.Version{}}
	self.log = log15.New("module", "pool")
	var err error
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

func (pl pool) Info() map[string]interface{} {
	result := make(map[string]interface{})
	result["snapshot"] = pl.pendingSc.info()
	accResult := make(map[types.Address]interface{})
	accSize := 0
	pl.pendingAc.Range(func(key, value interface{}) bool {
		k := key.(types.Address)
		cp := value.(*accountPool)
		accResult[k] = cp.info()
		accSize += 1
		return true
	})

	result["accounts"] = accResult
	result["accLen"] = accSize
	return result
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

	pl.accountSubID = pl.sync.SubscribeAccountBlock(pl.AddAccountBlock)
	pl.snapshotSubID = pl.sync.SubscribeSnapshotBlock(pl.AddSnapshotBlock)

	pl.pendingSc.Start()

	pl.newSnapshotBlockCond.Start(time.Millisecond * 30)
	pl.newAccBlockCond.Start(time.Millisecond * 40)
	pl.worker.closed = pl.closed
	pl.bc.Register(pl)
	common.Go(func() {
		pl.wg.Add(1)
		defer pl.wg.Done()
		pl.worker.work()
	})
}
func (pl *pool) Stop() {
	pl.log.Info("pool stop.")
	defer pl.log.Info("pool stopped.")
	pl.bc.UnRegister(pl)
	pl.sync.UnsubscribeAccountBlock(pl.accountSubID)
	pl.accountSubID = 0
	pl.sync.UnsubscribeSnapshotBlock(pl.snapshotSubID)
	pl.snapshotSubID = 0

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

	ac.setCompactDirty(true)
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
	return nil

}
func (pl *pool) AddAccountBlocks(address types.Address, blocks []*ledger.AccountBlock, source types.BlockSource) error {
	defer monitor.LogTime("pool", "addAccountArr", time.Now())

	for _, b := range blocks {
		pl.AddAccountBlock(address, b, source)
	}

	return nil
}

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
		return errors.Errorf("forkAccountTo key point is nil, target:%s, current:%s, targetTail:%s, targetHead:%s, currentTail:%s, currentHead:%s",
			targetChain.ID(), cu.ID(), targetChain.SprintTail(), targetChain.SprintHead(), cu.SprintTail(), cu.SprintHead())
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

func (pl *pool) destroyPendingAc(addr types.Address) {
	pl.pendingAc.Delete(addr)
}

func (pl *pool) ReadDownloadedChunks() *net.Chunk {
	chunk := pl.sync.Peek()
	return chunk
}

func (pl *pool) PopDownloadedChunks(hashH ledger.HashHeight) {
	pl.log.Info(fmt.Sprintf("pop chunks[%d-%s]", hashH.Height, hashH.Hash))
	pl.sync.Pop(hashH.Hash)
}

func (pl *pool) broadcastUnConfirmedBlocks() {
	blocks := pl.bc.GetAllUnconfirmedBlocks()
	for _, v := range blocks {
		pl.log.Info("broadcast unconfirmed blocks", "address", v.AccountAddress, "Height", v.Height, "Hash", v.Hash)
	}
	pl.sync.BroadcastAccountBlocks(blocks)
}

func (pl *pool) delUseLessChains() {
	if pl.sync.SyncState() != net.Syncing {
		pl.RLockInsert()
		defer pl.RUnLockInsert()
		info := pl.pendingSc.irreversible
		pl.delChainsForIrreversible(info)
		pl.pendingSc.checkPool()
		pl.pendingSc.loopDelUselessChain()
		var pendings []*accountPool
		pl.pendingAc.Range(func(_, v interface{}) bool {
			p := v.(*accountPool)
			pendings = append(pendings, p)
			return true
		})
		for _, v := range pendings {
			v.loopDelUselessChain()
			v.checkPool()
		}
	}
}

func (pl *pool) destroyAccounts() {
	var destroyList []types.Address
	pl.pendingAc.Range(func(key, value interface{}) bool {
		addr := key.(types.Address)

		accP := value.(*accountPool)
		if accP.shouldDestroy() {
			destroyList = append(destroyList, addr)
		}
		return true
	})
	for _, v := range destroyList {
		accP := pl.selfPendingAc(v)

		byt, _ := json.Marshal(accP.info())
		pl.log.Warn("destroy account pool", "addr", v, "Id", string(byt))
		pl.destroyPendingAc(v)
	}
}

func (pl *pool) delChainsForIrreversible(info *irreversibleInfo) {
	rollbackV := pl.rollbackVersion.Val()
	if info == nil || info.point == nil || info.rollbackV != rollbackV {
		return
	}
	// todo
}

func (pl *pool) compact() int {
	sum := 0
	sum += pl.accountsCompact(true)
	sum += pl.pendingSc.loopCompactSnapshot()
	return sum
}
func (pl *pool) snapshotCompact() int {
	return pl.pendingSc.loopCompactSnapshot()
}

func (pl *pool) accountsCompact(filterDirty bool) int {
	sum := 0
	var pendings []*accountPool
	pl.pendingAc.Range(func(_, v interface{}) bool {
		p := v.(*accountPool)
		if filterDirty && p.compactDirty {
			pendings = append(pendings, p)
			p.setCompactDirty(false)
		} else if !filterDirty {
			pendings = append(pendings, p)
		}
		return true
	})
	if len(pendings) > 0 {
		monitor.LogEventNum("pool", "AccountsCompact", len(pendings))
		for _, p := range pendings {
			pl.log.Debug("account compact", "addr", p.address, "filterDirty", filterDirty)
			sum = sum + p.Compact()
		}
	}
	return sum
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

		fc := ac.findInTreeDisk(v.Hash, v.Height, true)
		if fc == nil {
			result = false
			if ac.findInPool(v.Hash, v.Height) {
				continue
			}
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

func (fstat *failStat) init(d time.Duration) *failStat {
	fstat.timeThreshold = d
	return fstat
}
func (fstat *failStat) inc() bool {
	update := fstat.update
	if update != nil {
		if time.Now().Sub(*update) > fstat.timeThreshold {
			fstat.clear()
			return false
		}
	}
	if fstat.first == nil {
		now := time.Now()
		fstat.first = &now
	}
	now := time.Now()
	fstat.update = &now

	if fstat.update.Sub(*fstat.first) > fstat.timeThreshold {
		return false
	}
	return true
}

func (fstat *failStat) isFail() bool {
	first := fstat.first
	if first == nil {
		return false
	}
	update := fstat.update
	if update == nil {
		return false
	}

	if time.Now().Sub(*update) > 10*fstat.timeThreshold {
		fstat.clear()
		return false
	}

	if update.Sub(*first) > fstat.timeThreshold {
		return true
	}
	return false
}

func (fstat *failStat) clear() {
	fstat.first = nil
	fstat.update = nil
}

func (rstat *recoverStat) init(t int32, d time.Duration) *recoverStat {
	rstat.num = 0
	rstat.updateTime = time.Now()
	rstat.threshold = t
	rstat.timeThreshold = d
	return rstat
}

func (rstat *recoverStat) reset() *recoverStat {
	rstat.num = 0
	rstat.updateTime = time.Now()
	return rstat
}

func (rstat *recoverStat) inc() bool {
	atomic.AddInt32(&rstat.num, 1)
	now := time.Now()
	if now.Sub(rstat.updateTime) > rstat.timeThreshold {
		rstat.updateTime = now
		atomic.StoreInt32(&rstat.num, 0)
	} else {
		if rstat.num > rstat.threshold {
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
