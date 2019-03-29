package pool

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
)

type snapshotPool struct {
	BCPool
	//rwMu *sync.RWMutex
	//consensus consensus.AccountsConsensus
	closed          chan struct{}
	wg              sync.WaitGroup
	pool            *pool
	rw              *snapshotCh
	v               *snapshotVerifier
	f               *snapshotSyncer
	nextFetchTime   time.Time
	nextInsertTime  time.Time
	nextCompactTime time.Time
	hashBlacklist   Blacklist
}

func newSnapshotPoolBlock(block *ledger.SnapshotBlock, version *ForkVersion, source types.BlockSource) *snapshotPoolBlock {
	return &snapshotPoolBlock{block: block, forkBlock: *newForkBlock(version, source), failStat: (&failStat{}).init(time.Second * 20)}
}

type snapshotPoolBlock struct {
	forkBlock
	block *ledger.SnapshotBlock

	// last check data time
	lastCheckTime time.Time
	checkResult   bool
	failStat      *failStat
}

func (self *snapshotPoolBlock) ReferHashes() (keys []types.Hash, accounts []types.Hash, snapshot *types.Hash) {
	for _, v := range self.block.SnapshotContent {
		accounts = append(accounts, v.Hash)
	}
	if self.Height() > types.GenesisHeight {
		prev := self.PrevHash()
		snapshot = &prev
	}
	keys = append(keys, self.Hash())
	return
}

func (self *snapshotPoolBlock) Height() uint64 {
	return self.block.Height
}

func (self *snapshotPoolBlock) Hash() types.Hash {
	return self.block.Hash
}

func (self *snapshotPoolBlock) PrevHash() types.Hash {
	return self.block.PrevHash
}

func (self *snapshotPoolBlock) Source() types.BlockSource {
	return self.source
}

func newSnapshotPool(
	name string,
	version *ForkVersion,
	v *snapshotVerifier,
	f *snapshotSyncer,
	rw *snapshotCh,
	hashBlacklist Blacklist,
	log log15.Logger,
) *snapshotPool {
	pool := &snapshotPool{}
	pool.Id = name
	pool.version = version
	pool.rw = rw
	pool.v = v
	pool.f = f
	pool.log = log.New("snapshotPool", name)
	now := time.Now()
	pool.nextFetchTime = now
	pool.nextInsertTime = now
	pool.nextCompactTime = now
	pool.hashBlacklist = hashBlacklist
	return pool
}

func (self *snapshotPool) init(
	tools *tools,
	pool *pool) {
	//self.consensus = accountsConsensus
	self.pool = pool
	self.BCPool.init(tools)
}

func (self *snapshotPool) loopCheckFork() {
	// recover logic
	//defer func() {
	//	if err := recover(); err != nil {
	//		var e error
	//		switch t := err.(type) {
	//		case error:
	//			e = errors.WithStack(t)
	//		case string:
	//			e = errors.New(t)
	//		default:
	//			e = errors.Errorf("unknown type, %+v", err)
	//		}
	//
	//		self.log.Error("loopCheckFork start recover", "err", err, "withstack", fmt.Sprintf("%+v", e))
	//		fmt.Printf("%+v", e)
	//		defer self.log.Warn("loopCheckFork end recover.")
	//		self.pool.Lock()
	//		defer self.pool.Unlock()
	//		self.initPool()
	//		if self.rstat.inc() {
	//			common.Go(self.loopCheckFork)
	//		} else {
	//			panic(e)
	//		}
	//
	//		self.pool.version.Inc()
	//	}
	//}()
	self.wg.Add(1)
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			self.checkFork()
			// check fork every 2 sec.
			time.Sleep(2 * time.Second)
		}
	}
}

func (self *snapshotPool) checkFork() {
	current := self.CurrentChain()
	minHeight := self.pool.realSnapshotHeight(current)

	self.log.Debug("current chain.", "id", current.id(), "realH", minHeight, "headH", current.headHeight)

	longers := self.LongerChain(minHeight)

	var longest *forkedChain
	longestH := minHeight
	for _, l := range longers {
		if l.headHeight < longestH {
			continue
		}
		lH := self.pool.realSnapshotHeight(l)
		self.log.Debug("find chain.", "id", l.id(), "realH", lH, "headH", l.headHeight)
		if lH > longestH {
			longestH = lH
			longest = l
			self.log.Info("find more longer chain.", "id", l.id(), "realH", lH, "headH", l.headHeight, "tailH", l.tailHeight)
		}
	}

	if longest == nil {
		return
	}

	if longest.ChainId() == current.ChainId() {
		return
	}
	if longestH-self.LIMIT_LONGEST_NUM < current.headHeight {
		return
	}
	self.log.Info("current chain.", "id", current.id(), "realH", minHeight, "headH", current.headHeight, "tailH", current.tailHeight)

	monitor.LogEvent("pool", "snapshotFork")
	err := self.snapshotFork(longest, current)
	if err != nil {
		self.log.Error("checkFork", "err", err)
	}
}

func (self *snapshotPool) snapshotFork(longest *forkedChain, current *forkedChain) error {
	defer monitor.LogTime("pool", "snapshotFork", time.Now())
	self.log.Warn("[try]snapshot chain start fork.", "longest", longest.ChainId(), "current", current.ChainId(),
		"longestTailHeight", longest.tailHeight, "longestHeadHeight", longest.headHeight, "currentTailHeight", current.tailHeight, "currentHeadHeight", current.headHeight)
	self.pool.Lock()
	defer self.pool.UnLock()
	self.log.Warn("[lock]snapshot chain start fork.", "longest", longest.ChainId(), "current", current.ChainId())

	k, forked, err := self.getForkPointByChains(longest, current)
	if err != nil {
		self.log.Error("get snapshot forkPoint err.", "err", err)
		return err
	}
	if k == nil {
		self.log.Error("keypoint is empty.", "forked", forked.Height())
		return errors.New("key point is nil.")
	}
	keyPoint := k.(*snapshotPoolBlock)
	self.log.Info("fork point", "height", keyPoint.Height(), "hash", keyPoint.Hash())

	snapshots, accounts, e := self.rw.delToHeight(keyPoint.block.Height)
	if e != nil {
		return e
	}

	if len(snapshots) > 0 {
		err = self.rollbackCurrent(snapshots)
		if err != nil {
			return err
		}
	}

	if len(accounts) > 0 {
		err = self.pool.ForkAccounts(accounts)
		if err != nil {
			return err
		}
	}

	self.log.Debug("snapshotFork longest modify", "id", longest.id(), "TailHeight", longest.tailHeight, "HeadHeight", longest.headHeight)
	err = self.CurrentModifyToChain(longest, &ledger.HashHeight{Hash: forked.Hash(), Height: forked.Height()})
	if err != nil {
		return err
	}
	self.version.Inc()
	return nil
}

func (self *snapshotPool) loop() {
	//// recover logic
	//defer func() {
	//	if err := recover(); err != nil {
	//		var e error
	//		switch t := err.(type) {
	//		case error:
	//			e = errors.WithStack(t)
	//		case string:
	//			e = errors.New(t)
	//		default:
	//			e = errors.Errorf("unknown type, %+v", err)
	//		}
	//
	//		self.log.Error("snapshot loop start recover", "err", err, "withstack", fmt.Sprintf("%+v", e))
	//		fmt.Printf("%+v", e)
	//		defer self.log.Warn("snapshot loop end recover.")
	//		self.pool.Lock()
	//		defer self.pool.Unlock()
	//		self.initPool()
	//		if self.rstat.inc() {
	//			common.Go(self.loop)
	//		} else {
	//			panic(e)
	//		}
	//		self.pool.version.Inc()
	//	}
	//}()

	self.wg.Add(1)
	defer self.wg.Done()
	last := time.Now()
	for {
		select {
		case <-self.closed:
			return
		default:
			monitor.LogTime("pool", "snapshot_selectTime", last)
			now := time.Now()
			if now.After(self.nextCompactTime) {
				self.nextCompactTime = now.Add(50 * time.Millisecond)
				self.loopCompactSnapshot()
			}

			//if now.After(self.nextInsertTime) {
			//	size := self.CurrentChain().size()
			//	sleep := 200 * time.Millisecond
			//	if size > 10000 {
			//		sleep = 2 * time.Millisecond
			//		monitor.LogEvent("pool", "trySnapshotInsertSleep2")
			//	} else if size > 1000 {
			//		sleep = 20 * time.Millisecond
			//		monitor.LogEvent("pool", "trySnapshotInsertSleep20")
			//	} else if size > 100 {
			//		sleep = 50 * time.Millisecond
			//		monitor.LogEvent("pool", "trySnapshotInsertSleep50")
			//	} else {
			//		sleep = 200 * time.Millisecond
			//		monitor.LogEvent("pool", "trySnapshotInsertSleep200")
			//	}
			//
			//	self.nextInsertTime = now.Add(sleep)
			//	self.loopCheckCurrentInsert()
			//}
			n2 := time.Now()
			s1 := self.nextCompactTime.Sub(n2)
			s2 := self.nextInsertTime.Sub(n2)
			if s1 > s2 {
				time.Sleep(s2)
			} else {
				time.Sleep(s1)
			}
			monitor.LogTime("pool", "snapshotRealSleep", n2)
			last = time.Now()
		}
	}
}

func (self *snapshotPool) loopCompactSnapshot() {
	//defer monitor.LogTime("pool", "loopCompactSnapshotRLock", time.Now())
	//self.pool.RLock()
	//defer self.pool.RUnLock()
	//defer monitor.LogTime("pool", "loopCompactSnapshotMuLock", time.Now())
	//self.rMu.Lock()
	//defer self.rMu.Unlock()
	defer monitor.LogTime("pool", "snapshot_loopGenSnippetChains", time.Now())
	self.loopGenSnippetChains()
	defer monitor.LogTime("pool", "snapshot_loopAppendChains", time.Now())
	self.loopAppendChains()
	now := time.Now()
	if now.After(self.nextFetchTime) {
		self.nextFetchTime = now.Add(time.Millisecond * 200)
		defer monitor.LogTime("pool", "snapshot_loopFetchForSnippets", time.Now())
		self.loopFetchForSnippets()
		defer monitor.LogTime("pool", "snapshot_loopFetchForSnapshot", time.Now())
		self.loopFetchForSnapshot()
	}
}

func (self *snapshotPool) loopCheckCurrentInsert() {
	if self.chainpool.current.size() == 0 {
		return
	}
	defer monitor.LogTime("pool", "loopCheckCurrentInsert", time.Now())
	stat, block := self.snapshotTryInsert()

	if stat != nil {
		if stat.verifyResult() == verifier.FAIL {
			self.insertVerifyFail(block.(*snapshotPoolBlock), stat)
		} else if stat.verifyResult() == verifier.PENDING {
			self.insertVerifyPending(block.(*snapshotPoolBlock), stat)
		}
	}
}

func (self *snapshotPool) snapshotTryInsert() (*poolSnapshotVerifyStat, commonBlock) {
	defer monitor.LogTime("pool", "snapshotTryInsert", time.Now())
	self.pool.RLock()
	defer self.pool.RUnLock()
	defer monitor.LogTime("pool", "snapshotTryInsertRMu", time.Now())
	self.rMu.Lock()
	defer self.rMu.Unlock()

	pool := self.chainpool
	current := pool.current
	minH := current.tailHeight + 1
	headH := current.headHeight
L:
	for i := minH; i <= headH; i++ {
		block := current.getBlock(i, false)

		if !block.checkForkVersion() {
			block.resetForkVersion()
		}
		stat := self.v.verifySnapshot(block.(*snapshotPoolBlock))
		if !block.checkForkVersion() {
			block.resetForkVersion()
			continue
		}
		result := stat.verifyResult()
		switch result {
		case verifier.PENDING:
			return stat, block
		case verifier.FAIL:
			self.log.Error("snapshot verify fail."+stat.errMsg(),
				"hash", block.Hash(), "height", block.Height())
			return stat, block
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := pool.writeToChain(current, block)
				if err != nil {
					self.log.Error("insert snapshot chain fail.",
						"hash", block.Hash(), "height", block.Height(), "err", err)
					break L
				} else {
					self.blockpool.afterInsert(block)
				}
			} else {
				break L
			}
		default:
			self.log.Crit("Unexpected things happened. ",
				"result", result, "hash", block.Hash(), "height", block.Height())
			break L
		}
	}
	return nil, nil
}
func (self *snapshotPool) snapshotInsertItems(items []*Item) (map[types.Address][]commonBlock, *Item, error) {
	// lock current chain tail
	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()

	pool := self.chainpool
	current := pool.current

	for _, item := range items {
		block := item.commonBlock

		if block.Height() == current.tailHeight+1 &&
			block.PrevHash() == current.tailHash {
			block.resetForkVersion()
			stat := self.v.verifySnapshot(block.(*snapshotPoolBlock))
			if !block.checkForkVersion() {
				block.resetForkVersion()
				return nil, item, errors.New("new fork version")
			}
			switch stat.verifyResult() {
			case verifier.FAIL:
				self.log.Warn("add snapshot block to blacklist.", "hash", block.Hash(), "height", block.Height())
				self.hashBlacklist.AddAddTimeout(block.Hash(), time.Second*10)
				return nil, item, errors.New("fail verifier")
			case verifier.PENDING:
				self.log.Error("snapshot db.", "hash", block.Hash(), "height", block.Height())
				return nil, item, errors.New("fail verifier db.")
			}
			accBlocks, err := self.snapshotWriteToChain(current, block.(*snapshotPoolBlock))
			if err != nil {
				return nil, item, err
			}
			self.blockpool.afterInsert(block)
			if len(accBlocks) > 0 {
				return accBlocks, item, err
			}
		} else {
			return nil, item, errors.New("tail not match")
		}
	}
	return nil, nil, nil
}

func (self *snapshotPool) snapshotTryInsertItems(items []*Item) error {
	defer monitor.LogTime("pool", "snapshotTryInsertItems", time.Now())
	self.pool.RLock()
	defer self.pool.RUnLock()
	defer monitor.LogTime("pool", "snapshotTryInsertItemsRMu", time.Now())
	self.rMu.Lock()
	defer self.rMu.Unlock()

	pool := self.chainpool
	current := pool.current

	for _, item := range items {
		block := item.commonBlock

		if block.Height() == current.tailHeight+1 &&
			block.PrevHash() == current.tailHash {
			block.resetForkVersion()
			stat := self.v.verifySnapshot(block.(*snapshotPoolBlock))
			if !block.checkForkVersion() {
				block.resetForkVersion()
				return errors.New("new fork version")
			}
			switch stat.verifyResult() {
			case verifier.FAIL:
				self.log.Warn("add snapshot block to blacklist.", "hash", block.Hash(), "height", block.Height())
				self.hashBlacklist.AddAddTimeout(block.Hash(), time.Second*10)
				return errors.New("fail verifier")
			case verifier.PENDING:
				self.log.Error("snapshot db.", "hash", block.Hash(), "height", block.Height())
				return errors.New("fail verifier db.")
			}
			err := pool.writeToChain(current, block)
			if err != nil {
				return err
			}
			self.blockpool.afterInsert(block)
		} else {
			return errors.New("tail not match")
		}
	}
	return nil
}

func (self *snapshotPool) snapshotWriteToChain(current *forkedChain, block *snapshotPoolBlock) (map[types.Address][]commonBlock, error) {
	height := block.Height()
	hash := block.Hash()
	delAbs, err := self.rw.insertSnapshotBlock(block)
	if err == nil {
		current.removeTail(block)
		//self.fixReferInsert(chain, self.diskChain, height)
		return delAbs, nil
	} else {
		self.log.Error(fmt.Sprintf("waiting pool insert forkedChain fail. height:[%d], hash:[%s]", height, hash))
		return nil, err
	}
}

func (self *snapshotPool) Start() {
	self.closed = make(chan struct{})
	common.Go(self.loop)
	common.Go(self.loopCheckFork)
	self.log.Info("snapshot_pool started.")
}
func (self *snapshotPool) Stop() {
	close(self.closed)
	self.wg.Wait()
	self.log.Info("snapshot_pool stopped.")
}

func (self *snapshotPool) insertVerifyFail(b *snapshotPoolBlock, stat *poolSnapshotVerifyStat) {
	defer monitor.LogTime("pool", "insertVerifyFail", time.Now())
	block := b.block
	b.failStat.inc()
	results := stat.results

	accounts := make(map[types.Address]*ledger.HashHeight)

	for k, account := range block.SnapshotContent {
		result := results[k]
		if result == verifier.FAIL {
			accounts[k] = account
		}
	}

	if len(accounts) > 0 {
		self.log.Debug("insertVerifyFail", "accountsLen", len(accounts))
		monitor.LogEventNum("pool", "snapshotFailFork", len(accounts))
		self.forkAccounts(accounts)
		self.fetchAccounts(accounts, b.Height())
	}
}

func (self *snapshotPool) forkAccounts(accounts map[types.Address]*ledger.HashHeight) {
	self.pool.Lock()
	defer self.pool.UnLock()

	for k, v := range accounts {
		self.log.Debug("forkAccounts", "Addr", k.String(), "Height", v.Height, "Hash", v.Hash)
		err := self.pool.ForkAccountTo(k, v)
		if err != nil {
			self.log.Error("forkaccountTo err", "err", err)
		}
	}

	self.version.Inc()
}

func (self *snapshotPool) insertVerifyPending(b *snapshotPoolBlock, stat *poolSnapshotVerifyStat) {
	defer monitor.LogTime("pool", "insertVerifyPending", time.Now())
	block := b.block

	results := stat.results

	accounts := make(map[types.Address]*ledger.HashHeight)

	for k, account := range block.SnapshotContent {
		result := results[k]
		if result == verifier.PENDING {
			monitor.LogEvent("pool", "snapshotPending")
			self.log.Debug("db for account.", "addr", k.String(), "height", account.Height, "hash", account.Hash)
			hashH, e := self.pool.PendingAccountTo(k, account, b.Height())
			self.fetchAccounts(accounts, b.Height())
			if e != nil {
				self.log.Error("db for account fail.", "err", e, "address", k, "hashH", account)
			}
			if hashH != nil {
				accounts[k] = account
			}
		}
	}
	if len(accounts) > 0 {
		monitor.LogEventNum("pool", "snapshotPendingFork", len(accounts))
		self.forkAccounts(accounts)
	}
}

func (self *snapshotPool) AddDirectBlock(block *snapshotPoolBlock) error {
	self.rMu.Lock()
	defer self.rMu.Unlock()

	stat := self.v.verifySnapshot(block)
	result := stat.verifyResult()
	switch result {
	case verifier.PENDING:
		return errors.New("db for something")
	case verifier.FAIL:
		return errors.New(stat.errMsg())
	case verifier.SUCCESS:
		err := self.chainpool.diskChain.rw.insertBlock(block)
		if err != nil {
			return err
		}
		head := self.chainpool.diskChain.Head()
		self.chainpool.insertNotify(head)
		return nil
	default:
		self.log.Crit("verify unexpected.")
		return errors.New("verify unexpected")
	}
}
func (self *snapshotPool) loopFetchForSnapshot() {
	defer monitor.LogTime("pool", "loopFetchForSnapshot", time.Now())
	curHeight := self.pool.realSnapshotHeight(self.CurrentChain())
	longers := self.LongerChain(curHeight)

	self.pool.fetchForSnapshot(self.CurrentChain())
	for _, v := range longers {
		self.pool.fetchForSnapshot(v)
	}
	return
}
func (self *snapshotPool) makeQueue(q Package, info *offsetInfo) (uint64, error) {
	self.pool.RLock()
	defer self.pool.RUnLock()
	self.rMu.Lock()
	defer self.rMu.Unlock()

	cp := self.chainpool
	current := cp.current

	if info.offset == nil {
		info.offset = &ledger.HashHeight{Hash: current.tailHash, Height: current.tailHeight}
	} else {
		block := current.getBlock(info.offset.Height+1, false)
		if block == nil || block.PrevHash() != info.offset.Hash {
			return uint64(0), errors.New("current chain modify.")
		}
	}

	minH := info.offset.Height + 1
	headH := current.headHeight
	for i := minH; i <= headH; i++ {
		block := self.getCurrentBlock(i)
		if block == nil {
			return uint64(i - minH), errors.New("current chain modify")
		}

		if self.hashBlacklist.Exists(block.Hash()) {
			return uint64(i - minH), errors.New("block in blacklist")
		}

		item := NewItem(block, nil)

		err := q.AddItem(item)
		if err != nil {
			return uint64(i - minH), err
		}
		info.offset.Hash = item.Hash()
		info.offset.Height = item.Height()
	}

	return uint64(headH - minH), errors.New("all in")

}
func (self *snapshotPool) getCurrentBlock(i uint64) *snapshotPoolBlock {
	b := self.chainpool.current.getBlock(i, false)
	if b != nil {
		return b.(*snapshotPoolBlock)
	} else {
		return nil
	}
}
func (self *snapshotPool) getPendingForCurrent() ([]commonBlock, error) {
	begin := self.chainpool.current.tailHeight + 1
	blocks := self.chainpool.getCurrentBlocks(begin, begin+10)
	err := self.checkChain(blocks)
	if err != nil {
		return nil, err
	}

	return blocks, nil
}
func (self *snapshotPool) fetchAccounts(accounts map[types.Address]*ledger.HashHeight, sHeight uint64) {
	for addr, hashH := range accounts {
		ac := self.pool.selfPendingAc(addr)
		if !ac.existInPool(hashH.Hash) {
			head := ac.chainpool.diskChain.Head()
			u := uint64(10)
			if hashH.Height > head.Height() {
				u = hashH.Height - head.Height()
			}
			ac.f.fetchBySnapshot(*hashH, u, sHeight)
		}
	}

}

func (self *snapshotPool) makePackage(snapshotF SnapshotExistsFunc, accountF AccountExistsFunc, info *offsetInfo) (*snapshotPackage, error) {
	self.pool.RLock()
	defer self.pool.RUnLock()
	self.rMu.Lock()
	defer self.rMu.Unlock()

	cp := self.chainpool
	current := cp.current

	if info.offset == nil {
		info.offset = &ledger.HashHeight{Hash: current.tailHash, Height: current.tailHeight}
	}

	if current.size() == 0 {
		return NewSnapshotPackage2(snapshotF, accountF, 50, nil), nil
	}
	block := current.getBlock(info.offset.Height+1, false)
	if block == nil || block.PrevHash() != info.offset.Hash {
		return nil, errors.New("current chain modify.")
	}

	c := block.(*snapshotPoolBlock)
	return NewSnapshotPackage2(snapshotF, accountF, 50, c.block), nil

	//minH := info.offset.Height + 1
	//headH := current.headHeight
	//for i := minH; i <= headH; i++ {
	//	block := self.getCurrentBlock(i)
	//	if block == nil {
	//		return uint64(i - minH), errors.New("current chain modify")
	//	}
	//
	//	if self.hashBlacklist.Exists(block.Hash()) {
	//		return uint64(i - minH), errors.New("block in blacklist")
	//	}
	//
	//	item := NewItem(block, nil)
	//
	//	err := q.AddItem(item)
	//	if err != nil {
	//		return uint64(i - minH), err
	//	}
	//	info.offset.Hash = item.Hash()
	//	info.offset.Height = item.Height()
	//}
	//
	//return uint64(headH - minH), errors.New("all in")
}
