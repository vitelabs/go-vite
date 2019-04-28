package pool

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/vitelabs/go-vite/common"

	"github.com/vitelabs/go-vite/pool/batch"

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

	pool          *pool
	hashBlacklist Blacklist
}

func newAccountPoolBlock(block *ledger.AccountBlock,
	vmBlock vm_db.VmDb,
	version *common.Version,
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

func (self *accountPoolBlock) Owner() *types.Address {
	return &self.block.AccountAddress
}

func newAccountPool(name string, rw *accountCh, v *common.Version, hashBlacklist Blacklist, log log15.Logger) *accountPool {
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
	self.checkCurrent()
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
	// todo
	return nil
}

func (self *accountPool) findInPool(hash types.Hash, height uint64) bool {
	self.blockpool.pendingMu.Lock()
	defer self.blockpool.pendingMu.Unlock()
	return self.blockpool.containsHash(hash)
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
	stat := self.v.verifyAccount(received, latestSb)
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

func (self *accountPool) getCurrentBlock(i uint64) *accountPoolBlock {
	b := self.chainpool.tree.Main().GetKnot(i, false)
	if b != nil {
		return b.(*accountPoolBlock)
	} else {
		return nil
	}
}
func (self *accountPool) makePackage(q batch.Batch, info *offsetInfo, max uint64) (uint64, error) {
	// if current size is empty, do nothing.
	if self.chainpool.tree.Main().Size() <= 0 {
		return 0, errors.New("empty chainpool")
	}

	// lock other chain insert
	self.pool.RLockInsert()
	defer self.pool.RUnLockInsert()

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

		err := q.AddItem(block)
		if err != nil {
			return uint64(i - minH), err
		}
		info.offset.Hash = block.Hash()
		info.offset.Height = block.Height()
		info.quotaSub(block)
	}

	return uint64(headH - minH), errors.New("all in")
}

func (self *accountPool) tryInsertItems(p batch.Batch, items []batch.Item, latestSb *ledger.SnapshotBlock, version uint64) error {
	// if current size is empty, do nothing.
	if self.chainpool.tree.Main().Size() <= 0 {
		return errors.Errorf("empty chainpool, but item size:%d", len(items))
	}

	self.chainTailMu.Lock()
	defer self.chainTailMu.Unlock()

	cp := self.chainpool

	for i := 0; i < len(items); i++ {
		item := items[i]
		block := item.(*accountPoolBlock)
		self.log.Info(fmt.Sprintf("[%d]try to insert account block[%d-%s]%d-%d.", p.Id(), block.Height(), block.Hash(), i, len(items)))
		current := cp.tree.Main()
		tailHeight, tailHash := current.TailHH()
		if block.Height() == tailHeight+1 &&
			block.PrevHash() == tailHash {
			block.resetForkVersion()
			if block.forkVersion() != version {
				return errors.New("snapshot version update")
			}

			stat := self.v.verifyAccount(block, latestSb)
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
			err := cp.writeBlockToChain(stat.block)
			if err != nil {
				self.log.Error("account block write fail. ",
					"hash", block.Hash(), "height", block.Height(), "error", err)
				return err
			}
		} else {
			fmt.Println(self.address, block.block.IsSendBlock())
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
func (self *accountPool) genForSnapshotContents(p batch.Batch, b *snapshotPoolBlock, k types.Address, v *ledger.HashHeight) (bool, *stack.Stack) {
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
	err := tree.CheckTree(self.chainpool.tree)
	if err != nil {
		panic(err)
	}
}
