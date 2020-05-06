package pool

import (
	"errors"
	"sync"

	"time"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/log"
	"github.com/vitelabs/go-vite/interval/monitor"
	"github.com/vitelabs/go-vite/interval/verifier"
	"github.com/vitelabs/go-vite/interval/version"
)

type accountPool struct {
	BCPool
	mu         sync.Locker // read lock, snapshot insert and account insert
	rw         *accountCh
	verifyTask verifier.Task

	loopTime time.Time
}

func newAccountPool(name string, rw *accountCh, v *version.Version) *accountPool {
	pool := &accountPool{}
	pool.Id = name
	pool.rw = rw
	pool.version = v
	pool.loopTime = time.Now()
	return pool
}

func (self *accountPool) Init(
	verifier verifier.Verifier,
	syncer *fetcher,
	mu sync.Locker) {

	self.mu = mu
	self.BCPool.init(self.rw, verifier, syncer)
}

// 1. must be in diskchain
func (self *accountPool) TryRollback(rollbackHeight uint64, rollbackHash string) ([]*common.AccountStateBlock, error) {
	{ // check logic
		w := self.chainpool.diskChain.getBlock(rollbackHeight, false)
		if w == nil || w.block.Hash() != rollbackHash {
			return nil, errors.New("error rollback cmd.")
		}
	}

	head := self.chainpool.diskChain.Head()

	var sendBlocks []*common.AccountStateBlock

	headHeight := head.Height()
	for i := headHeight; i > rollbackHeight; i-- {
		w := self.chainpool.diskChain.getBlock(i, false)
		if w == nil {
			continue
		}
		block := w.block.(*common.AccountStateBlock)
		if block.BlockType == common.SEND {
			sendBlocks = append(sendBlocks, block)
		}
	}
	return sendBlocks, nil
}

// rollback to current
func (self *accountPool) FindRollbackPointByReferSnapshot(snapshotHeight uint64, snapshotHash string) (bool, *common.AccountStateBlock, error) {
	head := self.chainpool.diskChain.Head().(*common.AccountStateBlock)
	if head.SnapshotHeight < snapshotHeight {
		return false, nil, nil
	}

	accountBlock := self.rw.findAboveSnapshotHeight(snapshotHeight)
	if accountBlock == nil {
		return true, nil, nil
	} else {
		return true, accountBlock, nil
	}
}

func (self *accountPool) FindRollbackPointForAccountHashH(height uint64, hash string) (bool, *common.AccountStateBlock, Chain, error) {
	chain := self.whichChain(height, hash)
	if chain == nil {
		return false, nil, nil, nil
	}
	if chain.id() == self.chainpool.current.id() {
		return false, nil, nil, nil
	}
	_, forkPoint, err := self.getForkPointByChains(chain, self.chainpool.current)
	if err != nil {
		return false, nil, nil, err
	}
	return true, forkPoint.(*common.AccountStateBlock), chain, nil
}

func (self *accountPool) loop() int {
	//if !self.compactLock.TryLock() {
	//	return 0
	//} else {
	//	defer self.compactLock.UnLock()
	//}
	//
	//now := time.Now()
	//if now.After(self.loopTime.Add(time.Millisecond * 200)) {
	//	self.loopTime = now
	//	sum := 0
	//	sum = sum + self.loopGenSnippetChains()
	//	sum = sum + self.loopAppendChains()
	//	sum = sum + self.loopFetchForSnippets()
	//	sum = sum + self.TryInsert()
	//	return sum
	//}
	return 0
}

/**
1. compact for data
	1.1. free blocks
	1.2. snippet chain
2. fetch block for snippet chain.
*/
func (self *accountPool) Compact() int {
	// If no new data arrives, do nothing.
	if len(self.blockpool.freeBlocks) == 0 {
		return 0
	}
	// if an insert operation is in progress, do nothing.
	if !self.compactLock.TryLock() {
		return 0
	} else {
		defer self.compactLock.UnLock()
	}

	//	this is a rate limiter
	now := time.Now()
	if now.After(self.loopTime.Add(time.Millisecond * 200)) {
		defer monitor.LogTime("pool", "accountCompact", now)
		self.loopTime = now
		sum := 0
		sum = sum + self.loopGenSnippetChains()
		sum = sum + self.loopAppendChains()
		sum = sum + self.loopFetchForSnippets()
		return sum
	}
	return 0
}

/**
try insert block to real chain.
*/
func (self *accountPool) TryInsert() verifier.Task {
	// if current size is empty, do nothing.
	if self.chainpool.current.size() <= 0 {
		return nil
	}

	// if an compact operation is in progress, do nothing.
	if !self.compactLock.TryLock() {
		return nil
	} else {
		defer self.compactLock.UnLock()
	}

	// if last verify task has not done
	if self.verifyTask != nil && !self.verifyTask.Done() {
		return nil
	}
	// lock other chain insert
	self.mu.Lock()
	defer self.mu.Unlock()

	// try insert block to real chain
	defer monitor.LogTime("pool", "accountTryInsert", time.Now())
	task := self.tryInsert()
	self.verifyTask = task
	if task != nil {
		return task
	} else {
		return nil
	}
}

/**
1. fail    something is wrong.
2. pending
	2.1 pending for snapshot
	2.2 pending for other account chain(specific block height)
3. success



fail: If no fork version increase, don't do anything.
pending:
	pending(2.1): If snapshot height is not reached, fetch snapshot block, and wait..
	pending(2.2): If other account chain height is not reached, fetch other account block, and wait.
success:
	really insert to chain.
*/
func (self *accountPool) tryInsert() verifier.Task {
	self.rMu.Lock()
	defer self.rMu.Unlock()
	cp := self.chainpool
	current := cp.current
	minH := current.tailHeight + 1
	headH := current.headHeight
	n := 0
	for i := minH; i <= headH; i++ {
		wrapper := current.getBlock(i, false)
		block := wrapper.block
		wrapper.reset()
		n++
		stat := cp.verifier.VerifyReferred(block)
		if !wrapper.checkForkVersion() {
			wrapper.reset()
			return verifier.NewSuccessTask()
		}
		result := stat.VerifyResult()
		switch result {
		case verifier.PENDING:
			return stat.Task()
		case verifier.FAIL:
			log.Error("account block verify fail. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			return verifier.NewFailTask()
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := cp.writeToChain(current, wrapper)
				if err != nil {
					log.Error("account block write fail. block info:account[%s],hash[%s],height[%d], err:%v",
						result, block.Signer(), block.Hash(), block.Height(), err)
					return verifier.NewFailTask()
				} else {
					self.blockpool.afterInsert(wrapper)
				}
			} else {
				return verifier.NewSuccessTask()
			}
		default:
			// shutdown process
			log.Fatal("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, block.Signer(), block.Hash(), block.Height())
			return verifier.NewFailTask()
		}
	}

	return verifier.NewSuccessTask()
}

func (self *accountPool) insertAccountFailCallback(b common.Block, s verifier.BlockVerifyStat) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}

func (self *accountPool) insertAccountSuccessCallback(b common.Block, s verifier.BlockVerifyStat) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}
func (self *accountPool) FindInChain(hash string, height uint64) bool {

	for _, c := range self.chainpool.chains {
		b := c.heightBlocks[height]
		if b == nil {
			continue
		} else {
			if b.block.Hash() == hash {
				return true
			}
		}
	}
	return false
}
