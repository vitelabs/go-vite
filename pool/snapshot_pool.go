package pool

import (
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/viteshan/naive-vite/common/log"
)

type snapshotPool struct {
	BCPool
	rwMu *sync.RWMutex
	//consensus consensus.AccountsConsensus
	closed chan struct{}
	wg     sync.WaitGroup
	pool   *pool
	rw     *snapshotCh
	v      *snapshotVerifier
	f      *snapshotSyncer
}

func newSnapshotPoolBlock(block *ledger.SnapshotBlock, version *ForkVersion) *snapshotPoolBlock {
	return &snapshotPoolBlock{block: block, forkBlock: *newForkBlock(version)}
}

type snapshotPoolBlock struct {
	forkBlock
	block *ledger.SnapshotBlock
}

func (self *snapshotPoolBlock) Height() uint64 {
	return self.block.Height
}

func (self *snapshotPoolBlock) Hash() types.Hash {
	return self.block.Hash
}

func (self *snapshotPoolBlock) PreHash() types.Hash {
	return self.block.PrevHash
}

func newSnapshotPool(
	name string,
	version *ForkVersion,
	v *snapshotVerifier,
	f *snapshotSyncer,
	rw *snapshotCh,

) *snapshotPool {
	pool := &snapshotPool{}
	pool.Id = name
	pool.version = version
	pool.closed = make(chan struct{})
	pool.rw = rw
	pool.v = v
	pool.f = f
	return pool
}

func (self *snapshotPool) init(
	tools *tools,
	rwMu *sync.RWMutex,
	pool *pool) {
	self.rwMu = rwMu
	//self.consensus = accountsConsensus
	self.pool = pool
	self.BCPool.init(self.rw, tools)
}

func (self *snapshotPool) loopCheckFork() {
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
	longest := self.LongestChain()
	current := self.CurrentChain()
	if longest.ChainId() == current.ChainId() {
		return
	}
	self.snapshotFork(longest, current)

}

func (self *snapshotPool) snapshotFork(longest Chain, current Chain) error {
	log.Warn("[try]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())
	self.rwMu.Lock()
	defer self.rwMu.Unlock()
	log.Warn("[lock]snapshot chain start fork.longest chain:%s, currentchain:%s", longest.ChainId(), current.ChainId())

	k, _, err := self.getForkPoint(longest, current)
	if err != nil {
		log.Error("get snapshot forkPoint err. err:%v", err)
		return err
	}
	keyPoint := k.(*snapshotPoolBlock)

	snapshots, accounts, e := self.rw.delToHeight(keyPoint.block.Height - 1)
	if e != nil {
		return e
	}

	err = self.rollbackCurrent(snapshots)
	if err != nil {
		return err
	}
	err = self.pool.ForkAccounts(accounts)
	if err != nil {
		return err
	}

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
	stat, block := self.snapshotTryInsert()

	if stat != nil {
		if stat.verifyResult() == verifier.FAIL {
			self.insertVerifyFail(block.(*snapshotPoolBlock), stat)
		} else if stat.verifyResult() == verifier.PENDING {
			self.insertVerifyPending(block.(*snapshotPoolBlock), stat)
		}
	}
}

func (self *snapshotPool) stw() {
	self.rwMu.Lock()

}
func (self *snapshotPool) unStw() {
	self.rwMu.Unlock()
}
func (self *snapshotPool) snapshotTryInsert() (*poolSnapshotVerifyStat, commonBlock) {
	self.rwMu.RLock()
	defer self.rwMu.RUnlock()

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
			log.Error("snapshot verify fail. block info:hash[%s],height[%d], %s",
				result, block.Hash(), block.Height(), stat.errMsg())
			return stat, block
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := pool.writeToChain(current, block)
				if err != nil {
					log.Error("insert snapshot chain fail. block info:hash[%s],height[%d], err:%v",
						block.Hash(), block.Height(), err)
					break L
				} else {
					self.blockpool.afterInsert(block)
				}
			} else {
				break L
			}
		default:
			log.Fatal("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Hash(), block.Height())
			break L
		}
	}
	return nil, nil
}
func (self *snapshotPool) Start() {
	go self.loop()
	go self.loopCheckFork()
	log.Info("snapshot_pool[%s] started.", self.Id)
}
func (self *snapshotPool) Stop() {
	close(self.closed)
	self.wg.Wait()
	log.Info("snapshot_pool[%s] stopped.", self.Id)
}

func (self *snapshotPool) insertVerifyFail(b *snapshotPoolBlock, stat *poolSnapshotVerifyStat) {
	self.rwMu.Lock()
	defer self.rwMu.Unlock()

	block := b.block
	results := stat.results

	for k, account := range block.SnapshotContent {
		result := results[k]
		if result == verifier.FAIL {
			self.pool.ForkAccountTo(k, account)
		}
	}
	self.version.Inc()
}

func (self *snapshotPool) insertVerifyPending(b *snapshotPoolBlock, stat *poolSnapshotVerifyStat) {
	block := b.block

	results := stat.results

	for k, account := range block.SnapshotContent {
		result := results[k]
		if result == verifier.PENDING {
			self.pool.PendingAccountTo(k, account)
		}
	}
}
