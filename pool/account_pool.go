package pool

import (
	"sync"
	"time"

	"errors"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
	"github.com/viteshan/naive-vite/common/log"
)

type accountPool struct {
	BCPool
	mu            sync.Locker // read lock, snapshot insert and account insert
	rw            *accountCh
	verifyTask    verifyTask
	loopTime      time.Time
	address       types.Address
	v             *accountVerifier
	f             *accountSyncer
	receivedIndex sync.Map
}

func newAccountPoolBlock(block *ledger.AccountBlock, vmBlock vmctxt_interface.VmDatabase, version *ForkVersion) *accountPoolBlock {
	return &accountPoolBlock{block: block, vmBlock: vmBlock, forkBlock: *newForkBlock(version)}
}

type accountPoolBlock struct {
	forkBlock
	block   *ledger.AccountBlock
	vmBlock vmctxt_interface.VmDatabase
}

func (self *accountPoolBlock) Height() uint64 {
	return self.block.Height
}

func (self *accountPoolBlock) Hash() types.Hash {
	return self.block.Hash
}

func (self *accountPoolBlock) PreHash() types.Hash {
	return self.block.PrevHash
}

func newAccountPool(name string, rw *accountCh, v *ForkVersion) *accountPool {
	pool := &accountPool{}
	pool.Id = name
	pool.rw = rw
	pool.version = v
	pool.loopTime = time.Now()
	return pool
}

func (self *accountPool) Init(
	tools *tools,
	mu sync.Locker) {

	self.mu = mu
	self.BCPool.init(self.rw, tools)
}

// 1. must be in diskchain
//func (self *accountPool) TryRollback(rollbackHeight uint64, rollbackHash types.Hash) ([]*ledger.AccountBlock, error) {
//	{ // check logic
//		w := self.chainpool.diskChain.getBlock(rollbackHeight, false)
//		if w == nil || w.Hash() != rollbackHash {
//			return nil, errors.New("error rollback cmd.")
//		}
//	}
//
//	head := self.chainpool.diskChain.Head()
//
//	var sendBlocks []*ledger.AccountBlock
//
//	headHeight := head.Height()
//	for i := headHeight; i > rollbackHeight; i-- {
//		w := self.chainpool.diskChain.getBlock(i, false)
//		if w == nil {
//			continue
//		}
//		block := w.block.(*ledger.AccountBlock)
//		if block.BlockType == common.SEND {
//			sendBlocks = append(sendBlocks, block)
//		}
//	}
//	return sendBlocks, nil
//}

// rollback to current
//func (self *accountPool) FindRollbackPointByReferSnapshot(snapshotHeight uint64, snapshotHash string) (bool, *ledger.AccountBlock, error) {
//	head := self.chainpool.diskChain.Head().(*ledger.AccountBlock)
//	if head.SnapshotHeight < snapshotHeight {
//		return false, nil, nil
//	}
//
//	accountBlock := self.rw.findAboveSnapshotHeight(snapshotHeight)
//	if accountBlock == nil {
//		return true, nil, nil
//	} else {
//		return true, accountBlock, nil
//	}
//}

//func (self *accountPool) FindRollbackPointForAccountHashH(height uint64, hash types.Hash) (bool, *ledger.AccountBlock, Chain, error) {
//	chain := self.whichChain(height, hash)
//	if chain == nil {
//		return false, nil, nil, nil
//	}
//	if chain.id() == self.chainpool.current.id() {
//		return false, nil, nil, nil
//	}
//	_, forkPoint, err := self.getForkPointByChains(chain, self.chainpool.current)
//	if err != nil {
//		return false, nil, nil, err
//	}
//	return true, forkPoint.(*ledger.AccountBlock), chain, nil
//}

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
func (self *accountPool) TryInsert() verifyTask {
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
	if self.verifyTask != nil && !self.verifyTask.done() {
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
func (self *accountPool) tryInsert() verifyTask {
	self.rMu.Lock()
	defer self.rMu.Unlock()
	cp := self.chainpool
	current := cp.current
	minH := current.tailHeight + 1
	headH := current.headHeight
	n := 0
	for i := minH; i <= headH; i++ {
		block := current.getBlock(i, false)
		block.resetForkVersion()
		n++
		stat := self.v.verifyAccount(block.(*accountPoolBlock))
		if !block.checkForkVersion() {
			block.resetForkVersion()
			//return verifier.NewSuccessTask()
		}
		result := stat.verifyResult()
		switch result {
		case verifier.PENDING:
			return stat.task()
		case verifier.FAIL:
			log.Error("account block verify fail. block info:account[%s],hash[%s],height[%d]",
				result, self.address.String(), block.Hash(), block.Height())
			return self.v.newFailTask()
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err := cp.writeToChain(current, block)
				if err != nil {
					log.Error("account block write fail. block info:account[%s],hash[%s],height[%d], err:%v",
						result, self.address.String(), block.Hash(), block.Height(), err)
					return self.v.newFailTask()
				} else {
					self.blockpool.afterInsert(block)
					self.afterInsertBlock(block)
				}
			} else {
				return self.v.newSuccessTask()
			}
		default:
			// shutdown process
			log.Fatal("Unexpected things happened. verify result is %d. block info:account[%s],hash[%s],height[%d]",
				result, self.address, block.Hash(), block.Height())
			return self.v.newFailTask()
		}
	}

	return self.v.newSuccessTask()
}

func (self *accountPool) insertAccountFailCallback(b commonBlock) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}

func (self *accountPool) insertAccountSuccessCallback(b commonBlock) {
	log.Info("do nothing. height:%d, hash:%s, pool:%s", b.Height(), b.Hash(), self.Id)
}
func (self *accountPool) FindInChain(hash types.Hash, height uint64) bool {
	for _, c := range self.chainpool.chains {
		b := c.getBlock(height, false)
		if b == nil {
			continue
		} else {
			if b.Hash() == hash {
				return true
			}
		}
	}
	return false
}

func (self *accountPool) findInPool(hash types.Hash, height uint64) bool {
	for _, c := range self.chainpool.chains {
		b := c.getBlock(height, false)
		if b == nil {
			continue
		} else {
			if b.Hash() == hash {
				return true
			}
		}
	}
	return false
}

func (self *accountPool) findInTree(hash types.Hash, height uint64) *forkedChain {
	for _, c := range self.chainpool.chains {
		b := c.getBlock(height, false)

		if b == nil {
			continue
		} else {
			if b.Hash() == hash {
				return c
			}
		}
	}
	return nil
}
func (self *accountPool) AddDirectBlocks(received *accountPoolBlock, sendBlocks []*accountPoolBlock) error {
	self.rMu.Lock()
	defer self.rMu.Unlock()

	stat := self.v.verifyAccount(received)
	result := stat.verifyResult()
	switch result {
	case verifier.PENDING:
		return errors.New("pending for something")
	case verifier.FAIL:
		return errors.New(stat.errMsg())
	case verifier.SUCCESS:
		err := self.rw.insertBlocks(received, sendBlocks)
		if err != nil {
			return err
		}
		head := self.chainpool.diskChain.Head()
		self.chainpool.insertNotify(head)
		return nil
	default:
		log.Fatal("verify unexpected.")
		return errors.New("verify unexpected")
	}
}

func (self *accountPool) broadcastUnConfirmedBlocks() {
	blocks := self.rw.getUnConfirmedBlocks()
	self.f.broadcastBlocks(blocks)
}

func (self *accountPool) getFirstTimeoutUnConfirmedBlock(snapshotHead *ledger.SnapshotBlock) *ledger.AccountBlock {
	block := self.rw.getFirstUnconfirmedBlock()
	// todo need timeout tools
	self.v.verifyTimeout(snapshotHead.Timestamp, block)
	if block.Timestamp.Before(time.Now().Add(-time.Hour * 24)) {
		return block
	}
	return nil
}
func (self *accountPool) AddSendBlock(block *ledger.AccountBlock) {
	if block.IsReceiveBlock() {
		self.receivedIndex.Store(block.FromBlockHash, block)
	}
}
func (self *accountPool) afterInsertBlock(b commonBlock) {
	block := b.(*accountPoolBlock)
	self.receivedIndex.Delete(block.block.FromBlockHash)
}

func (self *accountPool) ExistInCurrent(fromHash types.Hash) bool {
	// received in pool
	b, ok := self.receivedIndex.Load(fromHash)
	if !ok {
		return false
	}

	block := b.(*ledger.AccountBlock)
	h := block.Height
	// block in current
	received := self.chainpool.current.getBlock(h, false)
	if received == nil {
		return false
	} else {
		return true
	}
	return ok
}
