package pool

import (
	"sync"
	"time"

	"math/rand"

	"fmt"

	"encoding/base64"

	"github.com/pkg/errors"
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
	loopFetchTime time.Time
	address       types.Address
	v             *accountVerifier
	f             *accountSyncer
	receivedIndex sync.Map
	pool          *pool
}

func newAccountPoolBlock(block *ledger.AccountBlock,
	vmBlock vmctxt_interface.VmDatabase,
	version *ForkVersion,
	source types.BlockSource) *accountPoolBlock {
	return &accountPoolBlock{
		forkBlock: *newForkBlock(version, source),
		block:     block,
		vmBlock:   vmBlock,
		recover:   (&recoverStat{}).init(10, time.Hour),
		failStat:  (&recoverStat{}).init(10, time.Second*30),
		delStat:   (&recoverStat{}).init(100, time.Minute*10),
		fail:      false,
	}
}

type accountPoolBlock struct {
	forkBlock
	block    *ledger.AccountBlock
	vmBlock  vmctxt_interface.VmDatabase
	recover  *recoverStat
	failStat *recoverStat
	delStat  *recoverStat
	fail     bool
}

func (self *accountPoolBlock) ReferHashes() (accounts []types.Hash, snapshot *types.Hash) {
	if self.block.IsReceiveBlock() {
		accounts = append(accounts, self.block.FromBlockHash)
	}
	if self.Height() > types.GenesisHeight {
		accounts = append(accounts, self.PrevHash())
	}
	snapshot = &self.block.SnapshotHash
	return accounts, snapshot
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
func (self *accountPoolBlock) Source() types.BlockSource {
	return self.source
}

func newAccountPool(name string, rw *accountCh, v *ForkVersion, log log15.Logger) *accountPool {
	pool := &accountPool{}
	pool.Id = name
	pool.rw = rw
	pool.version = v
	pool.loopTime = time.Now()
	pool.log = log.New("account", name)
	return pool
}

func (self *accountPool) Init(
	tools *tools, pool *pool, v *accountVerifier, f *accountSyncer) {
	self.pool = pool
	self.v = v
	self.f = f
	self.BCPool.init(tools)
}

/**
1. compact for data
	1.1. free blocks
	1.2. snippet chain
2. fetch block for snippet chain.
*/
func (self *accountPool) Compact() int {
	// if an insert operation is in progress, do nothing.
	if !self.compactLock.TryLock() {
		return 0
	} else {
		defer self.compactLock.UnLock()
	}

	defer func() {
		if err := recover(); err != nil {
			var e error
			switch t := err.(type) {
			case error:
				e = errors.WithStack(t)
			case string:
				e = errors.New(t)
			default:
				e = errors.Errorf("unknown type,%+v", err)
			}

			self.log.Warn("Compact start recover.", "err", err, "withstack", fmt.Sprintf("%+v", e))
			fmt.Printf("%+v", e)
			defer self.log.Warn("Compact end recover.")
			self.pool.RLock()
			defer self.pool.RUnLock()
			self.rMu.Lock()
			defer self.rMu.Unlock()
			self.initPool()
		}
	}()

	defer monitor.LogTime("pool", "accountCompact", time.Now())
	self.pool.RLock()
	defer self.pool.RUnLock()
	defer monitor.LogTime("pool", "accountCompactRMu", time.Now())
	self.rMu.Lock()
	defer self.rMu.Unlock()
	//	this is a rate limiter
	now := time.Now()
	sum := 0
	if now.After(self.loopTime.Add(time.Millisecond * 2)) {
		defer monitor.LogTime("pool", "accountSnippet", now)
		self.loopTime = now
		sum = sum + self.loopGenSnippetChains()
		sum = sum + self.loopAppendChains()
	}
	if now.After(self.loopFetchTime.Add(time.Millisecond * 200)) {
		defer monitor.LogTime("pool", "loopFetchForSnippets", now)
		self.loopFetchTime = now
		sum = sum + self.loopFetchForSnippets()
	}
	return sum
}

func (self *accountPool) LockForInsert() {
	// if an compact operation is in progress, do nothing.
	self.compactLock.Lock()
	// lock other chain insert
	self.pool.RLock()
	self.rMu.Lock()
}

func (self *accountPool) UnLockForInsert() {
	self.compactLock.UnLock()
	self.pool.RUnLock()
	self.rMu.Unlock()
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

	// recover logic
	defer func() {
		if err := recover(); err != nil {
			var e error
			switch t := err.(type) {
			case error:
				e = errors.WithStack(t)
			case string:
				e = errors.New(t)
			default:
				e = errors.Errorf("unknown type, %+v", err)
			}
			self.log.Warn("tryInsert start recover.", "err", err, "stack", fmt.Sprintf("%+v", e))
			fmt.Printf("%+v", e)
			defer self.log.Warn("tryInsert end recover.")
			self.initPool()
		}
	}()

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
			self.log.Warn("snapshot fork happen. account should verify again.", "blockHash", block.Hash(), "blockHeight", block.Height())
			return self.v.newSuccessTask()
		}
		result := stat.verifyResult()
		switch result {
		case verifier.PENDING:
			monitor.LogEvent("pool", "AccountPending")
			t := stat.task()
			if t == nil || len(t.requests()) == 0 {
				monitor.LogEvent("pool", "AccountPendingNotFound")
			}

			err := self.verifyPending(block)
			if err != nil {
				self.log.Error("account pending fail. ",
					"hash", block.Hash(), "height", block.Height(), "err", err)
			}
			return t
		case verifier.FAIL:
			monitor.LogEvent("pool", "AccountFail")
			err := self.verifyFail(block)
			self.log.Error("account block verify fail. ",
				"hash", block.Hash(), "height", block.Height(), "err", stat.errMsg(), "err2", err)
			return self.v.newFailTask()
		case verifier.SUCCESS:
			monitor.LogEvent("pool", "AccountSuccess")

			if len(stat.blocks) == 0 {
				self.log.Error("account block fail. ",
					"hash", block.Hash(), "height", block.Height(), "error", "stat.blocks is empty.")
				return self.v.newFailTask()
			}
			if block.Height() == current.tailHeight+1 {
				err, cnt := self.verifySuccess(stat.blocks)
				if err != nil {
					self.log.Error("account block write fail. ",
						"hash", block.Hash(), "height", block.Height(), "error", err)
					return self.v.newFailTask()
				}
				i = i + cnt
			} else {
				self.log.Error("account block forked", "height", block.Height())
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

	self.log.Debug("verifySuccess", "id", forked.id(), "TailHeight", forked.tailHeight, "HeadHeight", forked.headHeight)
	err = cp.currentModifyToChain(forked)
	if err != nil {
		return err, 0
	}
	err = cp.writeBlocksToChain(forked, blocks)
	if err != nil {
		return err, 0
	}
	for _, b := range blocks {
		self.blockpool.afterInsert(b)
		self.afterInsertBlock(b)
	}
	return nil, uint64(len(bs))
}

func (self *accountPool) verifyPending(b *accountPoolBlock) error {
	if !b.recover.inc() {
		b.recover.reset()
		monitor.LogEvent("pool", "accountPendingFail")
		return self.modifyToOther(b)
	}
	return nil
}
func (self *accountPool) verifyFail(b *accountPoolBlock) error {
	if b.fail {
		if !b.delStat.inc() {
			self.log.Warn("account block delete.", "hash", b.Hash(), "height", b.Height())
			self.CurrentModifyToEmpty()
		}
	} else {
		if !b.failStat.inc() {
			byt, _ := b.block.DbSerialize()
			self.log.Warn("account block verify fail.", "hash", b.Hash(), "height", b.Height(), "byt", base64.StdEncoding.EncodeToString(byt))
			b.fail = true
		}
	}
	return self.modifyToOther(b)
}
func (self *accountPool) modifyToOther(b *accountPoolBlock) error {
	cp := self.chainpool
	cur := cp.current

	cs := cp.findOtherChainsByTail(cur, cur.tailHash, cur.tailHeight)

	if len(cs) == 0 {
		return nil
	}

	monitor.LogEvent("pool", "accountVerifyFailModify")
	r := rand.Intn(len(cs))

	err := cp.currentModifyToChain(cs[r])

	return err
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
			err := newChain.canAddHead(b)
			if err != nil {
				return nil, nil, err
			}
			// forked chain
			newChain.addHead(tmp)
		} else {
			if tmp == nil || tmp.Hash() != b.Hash() {
				// forked chain
				newChain, err = cp.forkFrom(current, b.Height()-1, b.PrevHash())
				if err != nil {
					return nil, nil, err
				}
				err := newChain.canAddHead(b)
				if err != nil {
					return nil, nil, err
				}
				newChain.addHead(b)
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
	for _, c := range self.chainpool.allChain() {
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
	block := self.chainpool.current.getBlock(height, false)
	if block != nil && block.Hash() == hash {
		return self.chainpool.current
	}
	for _, c := range self.chainpool.allChain() {
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

func (self *accountPool) findInTreeDisk(hash types.Hash, height uint64, disk bool) *forkedChain {
	self.rMu.Lock()
	defer self.rMu.Unlock()

	block := self.chainpool.current.getBlock(height, disk)
	if block != nil && block.Hash() == hash {
		return self.chainpool.current
	}

	for _, c := range self.chainpool.allChain() {
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
		self.log.Debug("AddDirectBlocks", "id", fchain.id(), "TailHeight", fchain.tailHeight, "HeadHeight", fchain.headHeight)
		err = self.chainpool.currentModifyToChain(fchain)
		if err != nil {
			return err
		}
		err = self.chainpool.writeBlocksToChain(fchain, blocks)
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
		err := fchain.canAddHead(b)
		if err != nil {
			return nil, nil, err
		}
		fchain.addHead(b)
		results = append(results, b)
	}
	return fchain, results, nil
}
func (self *accountPool) deleteBlock(block *accountPoolBlock) {

}
func (self *accountPool) makeQueue(q Queue, info *offsetInfo) (uint64, error) {
	// if current size is empty, do nothing.
	if self.chainpool.current.size() <= 0 {
		return 0, errors.New("empty chainpool")
	}

	// if an compact operation is in progress, do nothing.
	self.compactLock.Lock()
	defer self.compactLock.UnLock()

	// lock other chain insert
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

		item := NewItem(block, &self.address)

		err := q.AddItem(item)
		if err != nil {
			return uint64(i - minH), err
		}
		info.offset.Hash = item.Hash()
		info.offset.Height = item.Height()
	}

	return uint64(headH - minH), errors.New("all in")
}

func (self *accountPool) tryInsertItems(items []*Item) error {
	// if current size is empty, do nothing.
	if self.chainpool.current.size() <= 0 {
		return errors.New("empty chainpool")
	}

	// if an compact operation is in progress, do nothing.
	self.compactLock.Lock()
	defer self.compactLock.UnLock()

	// lock other chain insert
	self.pool.RLock()
	defer self.pool.RUnLock()

	self.rMu.Lock()
	defer self.rMu.Unlock()

	cp := self.chainpool
	current := cp.current

	for i := 0; i < len(items); i++ {
		item := items[i]
		block := item.commonBlock

		if block.Height() == current.tailHeight+1 &&
			block.PrevHash() == current.tailHash {
			block.resetForkVersion()
			stat := self.v.verifyAccount(block.(*accountPoolBlock))
			if !block.checkForkVersion() {
				block.resetForkVersion()
				return errors.New("new fork version")
			}
			err, num := self.verifySuccess(stat.blocks)
			if err != nil {
				self.log.Error("account block write fail. ",
					"hash", block.Hash(), "height", block.Height(), "error", err)
				return err
			}
			i = i + int(num) - 1
		} else {
			fmt.Println(self.address, item.commonBlock.(*accountPoolBlock).block.IsSendBlock())
			return errors.New("tail not match")
		}
	}
	return nil
}
