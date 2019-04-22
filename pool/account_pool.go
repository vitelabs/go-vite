package pool

import (
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/pool/tree"

	"github.com/golang-collections/collections/stack"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_db"
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
	hashBlacklist Blacklist
}

func newAccountPoolBlock(block *ledger.AccountBlock,
	vmBlock vm_db.VmDb,
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
	vmBlock  vm_db.VmDb
	recover  *recoverStat
	failStat *recoverStat
	delStat  *recoverStat
	fail     bool
}

func (self *accountPoolBlock) ReferHashes() (keys []types.Hash, accounts []types.Hash, snapshot *types.Hash) {
	if self.block.IsReceiveBlock() {
		accounts = append(accounts, self.block.FromBlockHash)
	}
	if self.Height() > types.GenesisHeight {
		accounts = append(accounts, self.PrevHash())
	}
	keys = append(keys, self.Hash())
	if len(self.block.SendBlockList) > 0 {
		for _, sendB := range self.block.SendBlockList {
			keys = append(keys, sendB.Hash)
		}
	}
	// todo add send hashes for RS block
	return keys, accounts, snapshot
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

func newAccountPool(name string, rw *accountCh, v *ForkVersion, hashBlacklist Blacklist, log log15.Logger) *accountPool {
	pool := &accountPool{}
	pool.Id = name
	pool.rw = rw
	pool.version = v
	pool.loopTime = time.Now()
	pool.log = log.New("account", name)
	pool.hashBlacklist = hashBlacklist
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
	self.chainHeadMu.Lock()
	defer self.chainHeadMu.Unlock()
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

/**
try insert block to real chain.
*/
func (self *accountPool) pendingAccountTo(h *ledger.HashHeight, sHeight uint64) (*ledger.HashHeight, error) {
	self.chainHeadMu.Lock()
	defer self.chainHeadMu.Unlock()
	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()

	targetChain := self.findInTree(h.Hash, h.Height)
	if targetChain != nil {
		current := self.CurrentChain()

		if targetChain.Id() == current.Id() {
			return nil, nil
		}

		_, forkPoint, err := self.chainpool.tree.FindForkPointFromMain(targetChain)
		if err != nil {
			return nil, err
		}
		tailHeight, _ := current.TailHH()
		// key point in disk chain
		if forkPoint.Height() < tailHeight {
			return h, nil
		}
		self.log.Info("PendingAccountTo->CurrentModifyToChain", "addr", self.address, "hash", h.Hash, "height", h.Height, "targetChain",
			targetChain.Id(), "targetChainTailt", targetChain.SprintTail(), "targetChainHead", targetChain.SprintHead(),
			"forkPoint", fmt.Sprintf("[%s-%d]", forkPoint.Hash(), forkPoint.Height()))
		err = self.CurrentModifyToChain(targetChain, h)
		if err != nil {
			self.log.Error("PendingAccountTo->CurrentModifyToChain err", "err", err, "targetId", targetChain.Id())
			panic(err)
		}
		return nil, nil
	}
	return nil, nil
}

/**
1. fail    something is wrong.
2. db
	2.1 db for snapshot
	2.2 db for other account chain(specific block height)
3. success



fail: If no fork version increase, don't do anything.
db:
	db(2.1): If snapshot height is not reached, fetch snapshot block, and wait..
	db(2.2): If other account chain height is not reached, fetch other account block, and wait.
success:
	really insert to chain.
*/
//func (self *accountPool) tryInsert() verifyTask {
//	self.rMu.Lock()
//	defer self.rMu.Unlock()
//
//	//// recover logic
//	//defer func() {
//	//	if err := recover(); err != nil {
//	//		var e error
//	//		switch t := err.(type) {
//	//		case error:
//	//			e = errors.WithStack(t)
//	//		case string:
//	//			e = errors.New(t)
//	//		default:
//	//			e = errors.Errorf("unknown type, %+v", err)
//	//		}
//	//		self.log.Warn("tryInsert start recover.", "err", err, "stack", fmt.Sprintf("%+v", e))
//	//		fmt.Printf("%+v", e)
//	//		defer self.log.Warn("tryInsert end recover.")
//	//		self.initPool()
//	//	}
//	//}()
//
//	cp := self.chainpool
//	current := cp.current
//	minH := current.tailHeight + 1
//	headH := current.headHeight
//	n := 0
//	for i := minH; i <= headH; {
//		block := self.getCurrentBlock(i)
//		if block == nil {
//			return self.v.newSuccessTask()
//		}
//
//		block.resetForkVersion()
//		n++
//		stat := self.v.verifyAccount(block)
//		if !block.checkForkVersion() {
//			block.resetForkVersion()
//			self.log.Warn("snapshot fork happen. account should verify again.", "blockHash", block.Hash(), "blockHeight", block.Height())
//			return self.v.newSuccessTask()
//		}
//		result := stat.verifyResult()
//		switch result {
//		case verifier.PENDING:
//			monitor.LogEvent("pool", "AccountPending")
//			t := stat.task()
//			if t == nil || len(t.requests()) == 0 {
//				monitor.LogEvent("pool", "AccountPendingNotFound")
//			}
//
//			err := self.verifyPending(block)
//			if err != nil {
//				self.log.Error("account db fail. ",
//					"hash", block.Hash(), "height", block.Height(), "err", err)
//			}
//			return t
//		case verifier.FAIL:
//			monitor.LogEvent("pool", "AccountFail")
//			err := self.verifyFail(block)
//			self.log.Error("account block verify fail. ",
//				"hash", block.Hash(), "height", block.Height(), "err", stat.errMsg(), "err2", err)
//			return self.v.newFailTask()
//		case verifier.SUCCESS:
//			monitor.LogEvent("pool", "AccountSuccess")
//
//			if len(stat.blocks) == 0 {
//				self.log.Error("account block fail. ",
//					"hash", block.Hash(), "height", block.Height(), "error", "stat.blocks is empty.")
//				return self.v.newFailTask()
//			}
//			if block.Height() == current.tailHeight+1 {
//				err, cnt := self.verifySuccess(stat.blocks)
//				if err != nil {
//					self.log.Error("account block write fail. ",
//						"hash", block.Hash(), "height", block.Height(), "error", err)
//					return self.v.newFailTask()
//				}
//				i = i + cnt
//			} else {
//				self.log.Error("account block forked", "height", block.Height())
//				return self.v.newSuccessTask()
//			}
//		default:
//			// shutdown process
//			self.log.Crit("Unexpected things happened.",
//				"hash", block.Hash(), "height", block.Height(), "result", result)
//			return self.v.newFailTask()
//		}
//	}
//
//	return self.v.newSuccessTask()
//}
func (self *accountPool) verifySuccess(bs *accountPoolBlock) (error, uint64) {
	cp := self.chainpool

	err := self.rw.insertBlock(bs)
	if err != nil {
		return err, 0
	}

	cp.insertNotify(bs)

	if err != nil {
		return err, 0
	}
	self.blockpool.afterInsert(bs)
	self.afterInsertBlock(bs)
	return nil, 1
}

func (self *accountPool) verifyPending(b *accountPoolBlock) error {
	if !b.recover.inc() {
		b.recover.reset()
		monitor.LogEvent("pool", "accountPendingFail")
		// todo
		//return self.modifyToOther(b)
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
			byt, _ := b.block.Serialize()
			self.log.Warn("account block verify fail.", "hash", b.Hash(), "height", b.Height(), "byt", base64.StdEncoding.EncodeToString(byt))
			b.fail = true
		}
	}
	//return self.modifyToOther(b)
	// todo
	return nil
}

//func (self *accountPool) modifyToOther(b *accountPoolBlock) error {
//	cp := self.chainpool
//	cur := cp.current
//
//	cs := cp.findOtherChainsByTail(cur, cur.tailHash, cur.tailHeight)
//
//	if len(cs) == 0 {
//		return nil
//	}
//
//	monitor.LogEvent("pool", "accountVerifyFailModify")
//	r := rand.Intn(len(cs))
//
//	err := cp.currentModifyToChain(cs[r])
//
//	return err
//}

// result,(need fork)
func genBlocks(cp *chainPool, bs []*accountPoolBlock) ([]commonBlock, tree.Branch, error) {
	current := cp.tree.Main()

	var newChain tree.Branch
	var err error
	var result = []commonBlock{}

	for _, b := range bs {
		tmp := current.GetKnot(b.Height(), false)
		if newChain != nil {
			//err := newChain.canAddHead(b)
			//if err != nil {
			//	return nil, nil, err
			//}
			// forked chain
			newChain.AddHead(tmp)
		} else {
			if tmp == nil || tmp.Hash() != b.Hash() {
				// forked chain
				newChain, err = cp.forkFrom(current, b.Height()-1, b.PrevHash())
				if err != nil {
					return nil, nil, err
				}
				//err := newChain.canAddHead(b)
				//if err != nil {
				//	return nil, nil, err
				//}
				newChain.AddHead(b)
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
	self.blockpool.pendingMu.Lock()
	defer self.blockpool.pendingMu.Unlock()
	return self.blockpool.contains(hash, height)
}

func (self *accountPool) findInTree(hash types.Hash, height uint64) tree.Branch {
	return self.chainpool.tree.FindBranch(height, hash)
}

func (self *accountPool) findInTreeDisk(hash types.Hash, height uint64, disk bool) tree.Branch {
	cur := self.CurrentChain()
	block := cur.GetKnot(height, disk)
	if block != nil && block.Hash() == hash {
		return cur
	}

	for _, c := range self.chainpool.allChain() {
		b := c.GetKnot(height, false)

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

func (self *accountPool) findInTreeDiskTmp(hash types.Hash, height uint64, disk bool, sHeight uint64) tree.Branch {
	cur := self.CurrentChain()
	block := cur.GetKnot(height, disk)
	if block != nil && block.Hash() == hash {
		return cur
	}

	for _, c := range self.chainpool.allChain() {
		b := c.GetKnot(height, false)

		if b == nil {
			continue
		} else {
			if b.Hash() == hash {
				return c
			}
		}
	}
	if sHeight == 117 && block != nil {
		fmt.Printf("%s-%d-%s-%s\n", self.address, height, hash, block.Hash())
	}
	return nil
}

func (self *accountPool) AddDirectBlocks(received *accountPoolBlock) error {
	latestSb := self.rw.getLatestSnapshotBlock()
	//self.rMu.Lock()
	//defer self.rMu.Unlock()
	self.chainHeadMu.Lock()
	defer self.chainHeadMu.Unlock()

	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()

	current := self.CurrentChain()
	tailHeight, tailHash := current.TailHH()
	if received.Height() != tailHeight+1 ||
		received.PrevHash() != tailHash {
		return errors.Errorf("account head not match[%d-%s][%s]", received.Height(), received.PrevHash(), current.SprintTail())
	}

	self.checkCurrent()
	stat := self.v.verifyDirectAccount(received, latestSb)
	result := stat.verifyResult()
	switch result {
	case verifier.PENDING:
		msg := fmt.Sprintf("db for directly adding account block[%s-%s-%d].", received.block.AccountAddress, received.Hash(), received.Height())
		return errors.New(msg)
	case verifier.FAIL:
		if stat.err != nil {
			return stat.err
		}
		return errors.Errorf("directly adding account block[%s-%s-%d] fail.", received.block.AccountAddress, received.Hash(), received.Height())
	case verifier.SUCCESS:

		self.log.Debug("AddDirectBlocks", "height", received.Height(), "hash", received.Hash())
		err, _ := self.verifySuccess(stat.block)
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
	received := self.chainpool.tree.Main().GetKnot(h, false)
	if received == nil || received.Hash() != block.Hash {
		return false
	} else {
		return true
	}
	return ok
}
func (self *accountPool) getCurrentBlock(i uint64) *accountPoolBlock {
	b := self.chainpool.tree.Main().GetKnot(i, false)
	if b != nil {
		return b.(*accountPoolBlock)
	} else {
		return nil
	}
}
func (self *accountPool) genDirectBlocks(blocks []*accountPoolBlock) (tree.Branch, []commonBlock, error) {
	var results []commonBlock
	fchain, err := self.chainpool.forkFrom(self.chainpool.tree.Main(), blocks[0].Height()-1, blocks[0].PrevHash())
	if err != nil {
		return nil, nil, err
	}
	for _, b := range blocks {
		//err := fchain.canAddHead(b)
		//if err != nil {
		//	return nil, nil, err
		//}
		fchain.AddHead(b)
		results = append(results, b)
	}
	return fchain, results, nil
}
func (self *accountPool) deleteBlock(block *accountPoolBlock) {

}
func (self *accountPool) makePackage(q Package, info *offsetInfo, max uint64) (uint64, error) {
	// if current size is empty, do nothing.
	if self.chainpool.tree.Main().Size() <= 0 {
		return 0, errors.New("empty chainpool")
	}

	// lock other chain insert
	self.pool.RLock()
	defer self.pool.RUnLock()

	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()

	cp := self.chainpool
	current := cp.tree.Main()

	if info.offset == nil {
		tailHeight, tailHash := current.TailHH()
		info.offset = &ledger.HashHeight{Hash: tailHash, Height: tailHeight}
		info.quotaUnused = self.rw.getQuotaUnused()
	} else {
		block := current.GetKnot(info.offset.Height+1, false)
		if block == nil || block.PrevHash() != info.offset.Hash {
			return uint64(0), errors.New("current chain modify.")
		}
	}

	minH := info.offset.Height + 1
	headH, _ := current.HeadHH()
	for i := minH; i <= headH; i++ {
		if i-minH >= max {
			return uint64(i - minH), errors.New("arrived to max")
		}
		block := self.getCurrentBlock(i)
		if block == nil {
			return uint64(i - minH), errors.New("current chain modify")
		}
		if self.hashBlacklist.Exists(block.Hash()) {
			return uint64(i - minH), errors.New("block in blacklist")
		}
		// check quota
		used, unused, enought := info.quotaEnough(block)
		if !enought {
			// todo remove
			return uint64(i - minH), errors.New("block quota not enough.")
		}
		self.log.Debug(fmt.Sprintf("[%s][%d][%s]quota info [used:%d][unused:%d]\n", block.block.AccountAddress, block.Height(), block.Hash(), used, unused))
		// check request block confirmed time for response block
		if err := self.checkSnapshotSuccess(block); err != nil {
			return uint64(i - minH), err
		}
		item := NewItem(block, &self.address)

		err := q.AddItem(item)
		if err != nil {
			return uint64(i - minH), err
		}
		info.offset.Hash = item.Hash()
		info.offset.Height = item.Height()
		info.quotaSub(block)
	}

	return uint64(headH - minH), errors.New("all in")
}

func (self *accountPool) tryInsertItems(p Package, items []*Item, latestSb *ledger.SnapshotBlock, version int) error {
	// if current size is empty, do nothing.
	if self.chainpool.tree.Main().Size() <= 0 {
		return errors.Errorf("empty chainpool, but item size:%d", len(items))
	}

	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()

	cp := self.chainpool

	for i := 0; i < len(items); i++ {
		item := items[i]
		block := item.commonBlock
		self.log.Info(fmt.Sprintf("[%d]try to insert account block[%d-%s]%d-%d.", p.Id(), block.Height(), block.Hash(), i, len(items)))
		current := cp.tree.Main()
		tailHeight, tailHash := current.TailHH()
		if block.Height() == tailHeight+1 &&
			block.PrevHash() == tailHash {
			block.resetForkVersion()
			if block.forkVersion() != version {
				return errors.New("snapshot version update")
			}

			stat := self.v.verifyAccount(block.(*accountPoolBlock), latestSb)
			if !block.checkForkVersion() {
				block.resetForkVersion()
				return errors.New("new fork version")
			}
			switch stat.verifyResult() {
			case verifier.FAIL:
				self.log.Warn("add account block to blacklist.", "hash", block.Hash(), "height", block.Height(), "err", stat.err)
				self.hashBlacklist.AddAddTimeout(block.Hash(), time.Second*10)
				return errors.Wrap(stat.err, "fail verifier")
			case verifier.PENDING:
				self.log.Error("snapshot db.", "hash", block.Hash(), "height", block.Height())
				return errors.Wrap(stat.err, "fail verifier db.")
			}
			err := cp.writeBlockToChain(current, stat.block)
			if err != nil {
				self.log.Error("account block write fail. ",
					"hash", block.Hash(), "height", block.Height(), "error", err)
				return err
			}
			i = i + 1
		} else {
			fmt.Println(self.address, item.commonBlock.(*accountPoolBlock).block.IsSendBlock())
			return errors.New("tail not match")
		}
		self.log.Info(fmt.Sprintf("[%d]try to insert account block[%d-%s]%d-%d [latency:%s]success.", p.Id(), block.Height(), block.Hash(), i, len(items), block.Latency()))
	}
	return nil
}
func (self *accountPool) checkSnapshotSuccess(block *accountPoolBlock) error {
	if block.block.IsReceiveBlock() {
		num, e := self.rw.needSnapshot(block.block.AccountAddress)
		if e != nil {
			return e
		}
		if num > 0 {
			b, err := self.rw.getConfirmedTimes(block.block.FromBlockHash)
			if err != nil {
				return err
			}
			if b >= uint64(num) {
				return nil
			} else {
				return errors.New("send block need to snapshot.")
			}
		}
	}
	return nil
}
func (self *accountPool) genForSnapshotContents(p Package, b *snapshotPoolBlock, k types.Address, v *ledger.HashHeight) (bool, *stack.Stack) {
	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()
	acurr := self.CurrentChain()
	tailHeight, _ := acurr.TailHH()
	ab := acurr.GetKnot(v.Height, true)
	if ab == nil {
		return true, nil
	}
	if ab.Hash() != v.Hash {
		fmt.Printf("account chain has forked. snapshot block[%d-%s], account block[%s-%d][%s<->%s]\n",
			b.block.Height, b.block.Hash, k, v.Height, v.Hash, ab.Hash())
		// todo switch account chain

		return true, nil
	}

	if ab.Height() > tailHeight {
		// account block is in pool.
		tmp := stack.New()
		for h := ab.Height(); h > tailHeight; h-- {
			currB := self.getCurrentBlock(h)
			if p.Exists(currB.Hash()) {
				break
			}
			tmp.Push(currB)
		}
		if tmp.Len() > 0 {
			return false, tmp
		}
	}
	return false, nil
}
func (self *BCPool) checkCurrent() {
	main := self.CurrentChain()
	tailHeight, tailHash := main.TailHH()
	headHeight, headHash := self.chainpool.diskChain.HeadHH()
	if headHeight != tailHeight || headHash != tailHash {
		panic(fmt.Sprintf("pool[%s] tail[%d-%s], chain head[%d-%s]",
			main.Id(), tailHeight, tailHash, headHeight, headHash))
	}
}
