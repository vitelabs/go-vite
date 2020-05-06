package pool

import (
	"sync"
	"time"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/verifier"
	"github.com/vitelabs/go-vite/interval/version"
)

type snapshotPool struct {
	BCPool
	rwMu *sync.RWMutex
	//consensus consensus.AccountsConsensus
	closed chan struct{}
	wg     sync.WaitGroup
	pool   *pool
}

func newSnapshotPool(name string, v *version.Version) *snapshotPool {
	pool := &snapshotPool{}
	pool.Id = name
	pool.version = v
	pool.closed = make(chan struct{})
	return pool
}

func (self *snapshotPool) init(rw chainRw,
	verifier verifier.Verifier,
	syncer *fetcher,
	rwMu *sync.RWMutex,
	pool *pool) {
	self.rwMu = rwMu
	//self.consensus = accountsConsensus
	self.pool = pool
	self.BCPool.init(rw, verifier, syncer)
}

func (self *snapshotPool) loopCheckFork() {
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
	longest := self.LongestChain()
	current := self.CurrentChain()
	if longest.ChainId() == current.ChainId() {
		return
	}
	self.snapshotFork(longest, current)

}

func (self *snapshotPool) snapshotFork(longest Chain, current Chain) {
	log.Warn("[try]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())
	self.rwMu.Lock()
	defer self.rwMu.Unlock()
	log.Warn("[lock]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())

	k, f, err := self.getForkPoint(longest, current)
	if err != nil {
		log.Error("get snapshot forkPoint err. err:%v", err)
		return
	}
	forkPoint := f.(*common.SnapshotBlock)
	keyPoint := k.(*common.SnapshotBlock)

	startAcs, endAcs := self.getUnlockAccountSnapshot(forkPoint)

	err = self.pool.UnLockAccounts(startAcs, endAcs)
	if err != nil {
		log.Error("unlock accounts fail. err:%v", err)
		return
	}
	err = self.pool.ForkAccounts(keyPoint, forkPoint)
	if err != nil {
		log.Error("rollback accounts fail. err:%v", err)
		return
	}
	err = self.Rollback(forkPoint.Height(), forkPoint.Hash())
	if err != nil {
		log.Error("rollback snapshot fail. err:%v", err)
		return
	}
	err = self.CurrentModifyToChain(longest)
	if err != nil {
		log.Error("snapshot modify current fail. err:%v", err)
		return
	}
	self.version.Inc()
}

func (self *snapshotPool) loop() {
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			self.loopGenSnippetChains()
			self.loopAppendChains()
			self.loopFetchForSnippets()
			self.loopCheckCurrentInsert()
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (self *snapshotPool) loopCheckCurrentInsert() {
	if self.chainpool.current.size() == 0 {
		return
	}
	self.rwMu.RLock()
	defer self.rwMu.RUnlock()
	self.snapshotTryInsert()
}

func (self *snapshotPool) snapshotTryInsert() {
	pool := self.chainpool
	current := pool.current
	minH := current.tailHeight + 1
	headH := current.headHeight
L:
	for i := minH; i <= headH; i++ {
		wrapper := current.getBlock(i, false)
		block := wrapper.block

		if !wrapper.checkForkVersion() {
			wrapper.reset()
		}
		stat := pool.verifier.VerifyReferred(block)
		if !wrapper.checkForkVersion() {
			wrapper.reset()
			continue
		}
		result := stat.VerifyResult()
		switch result {
		case verifier.PENDING:
			self.insertVerifyPending(block, stat)
			break L
		case verifier.FAIL:
			log.Error("snapshot verify fail. block info:hash[%s],height[%d], %s",
				result, block.Hash(), block.Height(), stat.ErrMsg())
			self.insertVerifyFail(block, stat)
			break L
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := pool.writeToChain(current, wrapper)
				if err != nil {
					log.Error("insert snapshot chain fail. block info:hash[%s],height[%d], err:%v",
						block.Hash(), block.Height(), err)
					break L
				} else {
					self.blockpool.afterInsert(wrapper)
				}
			} else {
				break L
			}
		default:
			log.Fatal("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			break L
		}
	}
}
func (self *snapshotPool) Start() {
	self.wg.Add(1)
	go self.loop()
	self.wg.Add(1)
	go self.loopCheckFork()
	log.Info("snapshot_pool[%s] started.", self.Id)
}
func (self *snapshotPool) Stop() {
	close(self.closed)
	self.wg.Wait()
	log.Info("snapshot_pool[%s] stopped.", self.Id)
}

func (self *snapshotPool) insertVerifyFail(b common.Block, s verifier.BlockVerifyStat) {
	block := b.(*common.SnapshotBlock)
	stat := s.(*verifier.SnapshotBlockVerifyStat)
	results := stat.Results()

	for _, account := range block.Accounts {
		result := results[account.Addr]
		if result == verifier.FAIL {
			self.pool.ForkAccountTo(account)
		}
	}
	self.version.Inc()
}

func (self *snapshotPool) insertVerifyPending(b common.Block, s verifier.BlockVerifyStat) {
	block := b.(*common.SnapshotBlock)
	stat := s.(*verifier.SnapshotBlockVerifyStat)
	results := stat.Results()

	for _, account := range block.Accounts {
		result := results[account.Addr]
		if result == verifier.PENDING {
			self.pool.PendingAccountTo(account)
		}
	}
}

func (self *snapshotPool) getUnlockAccountSnapshot(block *common.SnapshotBlock) (map[string]*common.SnapshotPoint, map[string]*common.SnapshotPoint) {
	h := self.chainpool.diskChain.Head()
	head := h.(*common.SnapshotBlock)
	startAcs := make(map[string]*common.SnapshotPoint)
	endAcs := make(map[string]*common.SnapshotPoint)

	self.accounts(startAcs, endAcs, head)
	for i := head.Height() - 1; i > block.Height(); i-- {
		b := self.chainpool.diskChain.getBlock(i, false)
		if b != nil {
			block := b.block.(*common.SnapshotBlock)
			self.accounts(startAcs, endAcs, block)
		}
	}
	return startAcs, endAcs
}

func (self *snapshotPool) accounts(start map[string]*common.SnapshotPoint, end map[string]*common.SnapshotPoint, block *common.SnapshotBlock) {
	hs := block.Accounts
	for _, v := range hs {
		point := &common.SnapshotPoint{SnapshotHeight: block.Height(), SnapshotHash: block.Hash(), AccountHeight: v.Height, AccountHash: v.Hash}
		s := start[v.Addr]
		if s == nil {
			start[v.Addr] = point
		}
		end[v.Addr] = point
	}
}
