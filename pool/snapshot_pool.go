package pool

import (
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/pool/batch"

	"github.com/vitelabs/go-vite/pool/tree"

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
	closed chan struct{}
	wg     sync.WaitGroup
	pool   *pool
	rw     *snapshotCh
	v      *snapshotVerifier
	f      *snapshotSyncer

	nextFetchTime        time.Time
	hashBlacklist        Blacklist
	newSnapshotBlockCond *common.CondTimer
}

func newSnapshotPoolBlock(block *ledger.SnapshotBlock, version *common.Version, source types.BlockSource) *snapshotPoolBlock {
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

func (self *snapshotPoolBlock) Owner() *types.Address {
	return nil
}

func newSnapshotPool(
	name string,
	version *common.Version,
	v *snapshotVerifier,
	f *snapshotSyncer,
	rw *snapshotCh,
	hashBlacklist Blacklist,
	cond *common.CondTimer,
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
	pool.hashBlacklist = hashBlacklist
	pool.newSnapshotBlockCond = cond
	return pool
}

func (self *snapshotPool) init(
	tools *tools,
	pool *pool) {
	//self.consensus = accountsConsensus
	self.pool = pool
	self.BCPool.init(tools)
}

func (self *snapshotPool) checkFork() (tree.Branch, uint64, error) {
	current := self.CurrentChain()
	minHeight := self.pool.realSnapshotHeight(current)

	self.log.Debug("current chain.", "id", current.Id(), "realH", minHeight, "headH", current.SprintHead())

	longers := self.LongerChain(minHeight)

	var longest tree.Branch
	longestH := minHeight
	for _, l := range longers {
		headHeight, _ := l.HeadHH()
		if headHeight < longestH {
			continue
		}
		lH := self.pool.realSnapshotHeight(l)
		self.log.Debug("find chain.", "id", l.Id(), "realH", lH, "headH", headHeight)
		if lH > longestH {
			longestH = lH
			longest = l
			self.log.Info("find more longer chain.", "id", l.Id(), "realH", lH, "headH", headHeight, "tailH", l.SprintTail())
		}
	}

	if longest == nil {
		return nil, 0, nil
	}

	if longest.Id() == current.Id() {
		return nil, 0, nil
	}
	curHeadH, _ := current.HeadHH()
	if longestH-self.LIMIT_LONGEST_NUM < curHeadH {
		return nil, 0, nil
	}
	return longest, longestH, nil
}

func (self *snapshotPool) snapshotFork(longest tree.Branch, current tree.Branch) error {
	defer monitor.LogTime("pool", "snapshotFork", time.Now())
	self.log.Warn("[try]snapshot chain start fork.", "longest", longest.Id(), "current", current.Id(),
		"longestTail", longest.SprintTail(), "longestHead", longest.SprintHead(), "currentTail", current.SprintTail(), "currentHead", current.SprintHead())
	self.pool.LockInsert()
	defer self.pool.UnLockInsert()
	self.pool.LockRollback()
	defer self.pool.UnLockRollback()
	self.log.Warn("[lock]snapshot chain start fork.", "longest", longest.Id(), "current", current.Id())

	k, forked, err := self.chainpool.tree.FindForkPointFromMain(longest)
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
		self.checkCurrent()
	}

	if len(accounts) > 0 {
		err = self.pool.ForkAccounts(accounts)
		if err != nil {
			return err
		}
	}

	self.log.Debug("snapshotFork longest modify", "id", longest.Id(), "Tail", longest.SprintTail())
	err = self.CurrentModifyToChain(longest)
	if err != nil {
		return err
	}
	self.version.Inc()
	return nil
}

func (self *snapshotPool) loop() {
	self.wg.Add(1)
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			self.loopCompactSnapshot()
			self.newSnapshotBlockCond.Wait()
		}
	}
}

func (self *snapshotPool) loopCompactSnapshot() int {
	self.chainHeadMu.Lock()
	defer self.chainHeadMu.Unlock()
	sum := 0
	self.loopGenSnippetChains()
	sum += self.loopAppendChains()
	now := time.Now()
	if now.After(self.nextFetchTime) {
		self.nextFetchTime = now.Add(time.Millisecond * 200)
		self.loopFetchForSnippets()
		self.loopFetchForSnapshot()
	}
	self.checkCurrent()
	return sum
}

func (self *snapshotPool) snapshotInsertItems(p batch.Batch, items []batch.Item, version uint64) (map[types.Address][]commonBlock, batch.Item, error) {
	// lock current chain tail
	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()

	pool := self.chainpool
	current := pool.tree.Root()

	for i, item := range items {
		block := item.(*snapshotPoolBlock)
		self.log.Info(fmt.Sprintf("[%d]try to insert snapshot block[%d-%s]%d-%d.", p.Id(), block.Height(), block.Hash(), i, len(items)))
		tailHeight, tailHash := current.HeadHH()
		if block.Height() == tailHeight+1 &&
			block.PrevHash() == tailHash {
			block.resetForkVersion()
			if block.forkVersion() != version {
				return nil, nil, errors.Errorf("[%d]snapshot[s] version update", p.Id())
			}
			stat := self.v.verifySnapshot(block)
			if !block.checkForkVersion() {
				block.resetForkVersion()
				return nil, item, errors.New("new fork version")
			}

			switch stat.verifyResult() {
			case verifier.FAIL:
				self.log.Warn("add snapshot block to blacklist.", "hash", block.Hash(), "height", block.Height())
				self.hashBlacklist.AddAddTimeout(block.Hash(), time.Second*10)
				// todo
				panic(stat.errMsg())
				return nil, item, errors.New("fail verifier")
			case verifier.PENDING:
				self.log.Error("snapshot db.", "hash", block.Hash(), "height", block.Height())
				// todo
				panic(stat.errMsg())
				return nil, item, errors.New("fail verifier db.")
			}
			accBlocks, err := self.snapshotWriteToChain(block)
			if err != nil {
				return nil, item, err
			}
			self.log.Info(fmt.Sprintf("[%d]insert snapshot block[%d-%s]%d-%d success.", p.Id(), block.Height(), block.Hash(), i, len(items)))
			if len(accBlocks) > 0 {
				return accBlocks, item, err
			}
		} else {
			return nil, item, errors.New("tail not match")
		}
	}
	return nil, nil, nil
}

func (self *snapshotPool) snapshotWriteToChain(block *snapshotPoolBlock) (map[types.Address][]commonBlock, error) {
	height := block.Height()
	hash := block.Hash()
	delAbs, err := self.rw.insertSnapshotBlock(block)
	if err == nil {
		self.chainpool.tree.RootHeadAdd(block)
		//self.fixReferInsert(chain, self.diskChain, height)
		return delAbs, nil
	} else {
		self.log.Error(fmt.Sprintf("waiting pool insert forkedChain fail. height:[%d], hash:[%s]", height, hash), "err", err)
		return nil, err
	}
}

func (self *snapshotPool) Start() {
	self.closed = make(chan struct{})
	//common.Go(self.loop)
	//common.Go(self.loopCheckFork)
	self.log.Info("snapshot_pool started.")
}
func (self *snapshotPool) Stop() {
	close(self.closed)
	self.wg.Wait()
	self.log.Info("snapshot_pool stopped.")
}

func (self *snapshotPool) AddDirectBlock(block *snapshotPoolBlock) (map[types.Address][]commonBlock, error) {
	self.chainHeadMu.Lock()
	defer self.chainHeadMu.Unlock()

	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()
	current := self.CurrentChain()
	tailHeight, tailHash := current.TailHH()
	if block.Height() != tailHeight+1 ||
		block.PrevHash() != tailHash {
		return nil, errors.Errorf("snapshot head not match[%d-%s][%s]", block.Height(), block.PrevHash(), current.SprintTail())
	}
	stat := self.v.verifySnapshot(block)
	result := stat.verifyResult()
	switch result {
	case verifier.PENDING:
		return nil, errors.New("pending for something")
	case verifier.FAIL:
		return nil, errors.New(stat.errMsg())
	case verifier.SUCCESS:
		abs, err := self.rw.insertSnapshotBlock(block)
		if err != nil {
			return nil, err
		}
		self.chainpool.insertNotify(block)
		return abs, nil
	default:
		self.log.Crit("verify unexpected.")
		return nil, errors.New("verify unexpected")
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

//func (self *snapshotPool) makeQueue(q Package, info *offsetInfo) (uint64, error) {
//	self.pool.RLock()
//	defer self.pool.RUnLock()
//	self.rMu.Lock()
//	defer self.rMu.Unlock()
//
//	cp := self.chainpool
//	current := cp.current
//
//	if info.offset == nil {
//		info.offset = &ledger.HashHeight{Hash: current.tailHash, Height: current.tailHeight}
//	} else {
//		block := current.getBlock(info.offset.Height+1, false)
//		if block == nil || block.PrevHash() != info.offset.Hash {
//			return uint64(0), errors.New("current chain modify.")
//		}
//	}
//
//	minH := info.offset.Height + 1
//	headH := current.headHeight
//	for i := minH; i <= headH; i++ {
//		block := self.getCurrentBlock(i)
//		if block == nil {
//			return uint64(i - minH), errors.New("current chain modify")
//		}
//
//		if self.hashBlacklist.Exists(block.Hash()) {
//			return uint64(i - minH), errors.New("block in blacklist")
//		}
//
//		item := NewItem(block, nil)
//
//		err := q.AddItem(item)
//		if err != nil {
//			return uint64(i - minH), err
//		}
//		info.offset.Hash = item.Hash()
//		info.offset.Height = item.Height()
//	}
//
//	return uint64(headH - minH), errors.New("all in")
//
//}
func (self *snapshotPool) getCurrentBlock(i uint64) *snapshotPoolBlock {
	b := self.CurrentChain().GetKnot(i, false)
	if b != nil {
		return b.(*snapshotPoolBlock)
	} else {
		return nil
	}
}

//func (self *snapshotPool) getPendingForCurrent() ([]commonBlock, error) {
//	begin := self.chainpool.current.tailHeight + 1
//	blocks := self.chainpool.getCurrentBlocks(begin, begin+10)
//	err := self.checkChain(blocks)
//	if err != nil {
//		return nil, err
//	}
//
//	return blocks, nil
//}
func (self *snapshotPool) fetchAccounts(accounts map[types.Address]*ledger.HashHeight, sHeight uint64, sHash types.Hash) {
	for addr, hashH := range accounts {
		ac := self.pool.selfPendingAc(addr)
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
func (self *snapshotPool) genMaxAccounts(targetHeight uint64) (map[types.Address]*ledger.HashHeight, error) {
	result := make(map[types.Address]*ledger.HashHeight)
	cur := self.CurrentChain()
	tailHeight, _ := cur.TailHH()
	headHeight, _ := cur.HeadHH()
	for i := targetHeight; i > tailHeight; i-- {
		knot := cur.GetKnot(i, true)
		if knot == nil {
			return nil, errors.Errorf("can't find block[%d] from current[%s][%d-%d][%d]", i, cur.Id(), tailHeight, headHeight, targetHeight)
		}
		block := knot.(*snapshotPoolBlock).block
		for k, v := range block.SnapshotContent {
			if r := result[k]; r == nil {
				result[k] = &ledger.HashHeight{Hash: v.Hash, Height: v.Height}
			}
		}
	}
	return result, nil
}

//func (self *snapshotPool) makePackage(snapshotF SnapshotExistsFunc, accountF AccountExistsFunc, info *offsetInfo) (*snapshotPackage, error) {
//	self.pool.RLock()
//	defer self.pool.RUnLock()
//	self.rMu.Lock()
//	defer self.rMu.Unlock()
//
//	cp := self.chainpool
//	current := cp.current
//
//	if info.offset == nil {
//		info.offset = &ledger.HashHeight{Hash: current.tailHash, Height: current.tailHeight}
//	}
//
//	if current.size() == 0 {
//		return NewSnapshotPackage2(snapshotF, accountF, 50, nil), nil
//	}
//	block := current.getBlock(info.offset.Height+1, false)
//	if block == nil || block.PrevHash() != info.offset.Hash {
//		return nil, errors.New("current chain modify.")
//	}
//
//	c := block.(*snapshotPoolBlock)
//	return NewSnapshotPackage2(snapshotF, accountF, 50, c.block), nil
//
//	//minH := info.offset.Height + 1
//	//headH := current.headHeight
//	//for i := minH; i <= headH; i++ {
//	//	block := self.getCurrentBlock(i)
//	//	if block == nil {
//	//		return uint64(i - minH), errors.New("current chain modify")
//	//	}
//	//
//	//	if self.hashBlacklist.Exists(block.Hash()) {
//	//		return uint64(i - minH), errors.New("block in blacklist")
//	//	}
//	//
//	//	item := NewItem(block, nil)
//	//
//	//	err := q.AddItem(item)
//	//	if err != nil {
//	//		return uint64(i - minH), err
//	//	}
//	//	info.offset.Hash = item.Hash()
//	//	info.offset.Height = item.Height()
//	//}
//	//
//	//return uint64(headH - minH), errors.New("all in")
//}
