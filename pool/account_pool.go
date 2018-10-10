package pool

import (
	"sync"
	"time"

	"errors"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_context/vmctxt_interface"
)

type accountPool struct {
	BCPool
	rw            *accountCh
	verifyTask    verifyTask
	loopTime      time.Time
	address       types.Address
	v             *accountVerifier
	f             *accountSyncer
	receivedIndex sync.Map
	log           log15.Logger
	pool          *pool
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

func (self *accountPoolBlock) PrevHash() types.Hash {
	return self.block.PrevHash
}

func newAccountPool(name string, rw *accountCh, v *ForkVersion, log log15.Logger) *accountPool {
	pool := &accountPool{}
	pool.Id = name
	pool.rw = rw
	pool.version = v
	pool.loopTime = time.Now()
	pool.log = log.New("name", name)
	return pool
}

func (self *accountPool) Init(
	tools *tools, pool *pool, v *accountVerifier) {
	self.pool = pool
	self.v = v
	self.BCPool.init(self.rw, tools)
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
	if self.verifyTask != nil && !self.verifyTask.done(self.rw.rw) {
		return nil
	}
	// lock other chain insert
	self.pool.RLock()
	defer self.pool.RUnLock()

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
	for i := minH; i <= headH; {
		block := self.getCurrentBlock(i)
		if block == nil {
			return self.v.newSuccessTask()
		}

		block.resetForkVersion()
		n++
		stat := self.v.verifyAccount(block)
		if !block.checkForkVersion() {
			block.resetForkVersion()
			return self.v.newSuccessTask()
		}
		result := stat.verifyResult()
		switch result {
		case verifier.PENDING:
			return stat.task()
		case verifier.FAIL:
			self.log.Error("account block verify fail. ",
				"hash", block.Hash(), "height", block.Height())
			return self.v.newFailTask()
		case verifier.SUCCESS:
			if block.Height() == current.tailHeight+1 {
				err, cnt := self.verifySuccess(stat.blocks)
				if err != nil {
					self.log.Error("account block write fail. ",
						"hash", block.Hash(), "height", block.Height(), "error", err)
					return self.v.newFailTask()
				}
				i = i + cnt
			} else {
				return self.v.newSuccessTask()
			}
		default:
			// shutdown process
			self.log.Crit("Unexpected things happened.",
				"hash", block.Hash(), "height", block.Height(), "result", result)
			return self.v.newFailTask()
		}
	}

	return self.v.newSuccessTask()
}
func (self *accountPool) verifySuccess(bs []*accountPoolBlock) (error, uint64) {
	cp := self.chainpool

	blocks, forked, err := genBlocks(cp, bs)
	if err != nil {
		return err, 0
	}

	err = cp.writeBlocksToChain(forked, blocks)
	if err != nil {
		return err, 0
	}
	cp.currentModifyToChain(forked)
	for _, b := range blocks {
		self.blockpool.afterInsert(b)
		self.afterInsertBlock(b)
	}
	return nil, uint64(len(bs))
}

// result,(need fork)
func genBlocks(cp *chainPool, bs []*accountPoolBlock) ([]commonBlock, *forkedChain, error) {
	current := cp.current

	var newChain *forkedChain
	var err error
	var result = []commonBlock{}

	for _, b := range bs {
		tmp := current.getHeightBlock(b.Height())
		if newChain != nil {
			// forked chain
			newChain.addHead(tmp)
		} else {
			if tmp == nil || tmp.Hash() != b.Hash() {
				// forked chain
				newChain, err = cp.forkFrom(current, b.Height()-1, b.PrevHash())
				if err != nil {
					return nil, nil, err
				}
				newChain.addHead(tmp)
			}
		}
		result = append(result, b)
	}

	if newChain == nil {
		return result, current, nil
	} else {
		return result, newChain, nil
	}
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

	stat := self.v.verifyDirectAccount(received, sendBlocks)
	result := stat.verifyResult()
	switch result {
	case verifier.PENDING:
		return errors.New("pending for something")
	case verifier.FAIL:
		return errors.New(stat.errMsg())
	case verifier.SUCCESS:
		fchain, blocks, err := self.genDirectBlocks(stat.blocks)
		if err != nil {
			return err
		}
		err = self.chainpool.writeBlocksToChain(fchain, blocks)
		if err != nil {
			return err
		}
		err = self.chainpool.currentModifyToChain(fchain)
		if err != nil {
			return err
		}
		return nil
	default:
		self.log.Crit("verify unexpected.")
		return errors.New("verify unexpected")
	}
}

func (self *accountPool) broadcastUnConfirmedBlocks() {
	blocks := self.rw.getUnConfirmedBlocks()
	self.f.broadcastBlocks(blocks)
}

func (self *accountPool) AddReceivedBlock(block *ledger.AccountBlock) {
	if block.IsReceiveBlock() {
		self.receivedIndex.Store(block.FromBlockHash, block)
	}
}
func (self *accountPool) afterInsertBlock(b commonBlock) {
	block := b.(*accountPoolBlock)
	if block.block.IsReceiveBlock() {
		self.receivedIndex.Delete(block.block.FromBlockHash)
	}
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
	if received == nil || received.Hash() != block.Hash {
		return false
	} else {
		return true
	}
	return ok
}
func (self *accountPool) getCurrentBlock(i uint64) *accountPoolBlock {
	b := self.chainpool.current.getBlock(i, false)
	if b != nil {
		return b.(*accountPoolBlock)
	} else {
		return nil
	}
}
func (self *accountPool) genDirectBlocks(blocks []*accountPoolBlock) (*forkedChain, []commonBlock, error) {
	var results []commonBlock
	fchain, err := self.chainpool.forkFrom(self.chainpool.current, blocks[0].Height()-1, blocks[0].PrevHash())
	if err != nil {
		return nil, nil, err
	}
	for _, b := range blocks {
		fchain.addHead(b)
		results = append(results, b)
	}
	return fchain, results, nil
}
