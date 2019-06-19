package pool

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/golang-collections/collections/stack"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/pool/batch"
	"github.com/vitelabs/go-vite/pool/tree"
	"github.com/vitelabs/go-vite/verifier"
	"github.com/vitelabs/go-vite/vm_db"
)

type accountPool struct {
	BCPool
	poolContext
	rw            *accountCh
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

func (accB *accountPoolBlock) ReferHashes() (keys []types.Hash, accounts []types.Hash, snapshot *types.Hash) {
	if accB.block.IsReceiveBlock() {
		accounts = append(accounts, accB.block.FromBlockHash)
	}
	if accB.Height() > types.GenesisHeight {
		accounts = append(accounts, accB.PrevHash())
	}
	keys = append(keys, accB.Hash())
	if len(accB.block.SendBlockList) > 0 {
		for _, sendB := range accB.block.SendBlockList {
			keys = append(keys, sendB.Hash)
		}
	}
	return keys, accounts, snapshot
}

func (accB *accountPoolBlock) Height() uint64 {
	return accB.block.Height
}

func (accB *accountPoolBlock) Hash() types.Hash {
	return accB.block.Hash
}

func (accB *accountPoolBlock) PrevHash() types.Hash {
	return accB.block.PrevHash
}

func (accB *accountPoolBlock) Owner() *types.Address {
	return &accB.block.AccountAddress
}

func newAccountPool(name string, rw *accountCh, v *common.Version, hashBlacklist Blacklist, log log15.Logger) *accountPool {
	pool := &accountPool{}
	pool.ID = name
	pool.rw = rw
	pool.version = v
	pool.loopTime = time.Now()
	pool.log = log.New("account", name)
	pool.hashBlacklist = hashBlacklist
	return pool
}

func (accP *accountPool) Init(
	tools *tools, pool *pool, v *accountVerifier, f *accountSyncer) {
	accP.pool = pool
	accP.v = v
	accP.f = f
	accP.BCPool.init(tools)
}

/**
1. compact for data
	1.1. free blocks
	1.2. snippet chain
2. fetch block for snippet chain.
*/
func (accP *accountPool) Compact() int {
	accP.chainHeadMu.Lock()
	defer accP.chainHeadMu.Unlock()
	//	this is a rate limiter
	now := time.Now()
	sum := 0

	defer monitor.LogTime("pool", "accountSnippet", now)
	accP.loopTime = now
	sum = sum + accP.loopGenSnippetChains()
	sum = sum + accP.loopAppendChains()

	if now.After(accP.loopFetchTime.Add(time.Millisecond * 200)) {
		defer monitor.LogTime("pool", "loopFetchForSnippets", now)
		accP.loopFetchTime = now
		sum = sum + accP.loopFetchForSnippets()
		accP.checkCurrent()
	}
	return sum
}

/**
try insert block to real chain.
*/
func (accP *accountPool) pendingAccountTo(h *ledger.HashHeight, sHeight uint64) (*ledger.HashHeight, error) {
	accP.chainHeadMu.Lock()
	defer accP.chainHeadMu.Unlock()
	accP.chainTailMu.Lock()
	defer accP.chainTailMu.Unlock()

	targetChain := accP.findInTree(h.Hash, h.Height)
	if targetChain != nil {
		current := accP.CurrentChain()

		if targetChain.ID() == current.ID() {
			return nil, nil
		}

		_, forkPoint, err := accP.chainpool.tree.FindForkPointFromMain(targetChain)
		if err != nil {
			return nil, err
		}
		tailHeight, _ := current.TailHH()
		// key point in disk chain
		if forkPoint.Height() < tailHeight {
			return h, nil
		}
		accP.log.Info("PendingAccountTo->CurrentModifyToChain", "addr", accP.address, "hash", h.Hash, "height", h.Height, "targetChain",
			targetChain.ID(), "targetChainTailt", targetChain.SprintTail(), "targetChainHead", targetChain.SprintHead(),
			"forkPoint", fmt.Sprintf("[%s-%d]", forkPoint.Hash(), forkPoint.Height()))
		err = accP.CurrentModifyToChain(targetChain)
		if err != nil {
			accP.log.Error("PendingAccountTo->CurrentModifyToChain err", "err", err, "targetId", targetChain.ID())
			panic(err)
		}
		return nil, nil
	}
	return nil, nil
}

func (accP *accountPool) verifySuccess(bs *accountPoolBlock) (uint64, error) {
	cp := accP.chainpool

	err := accP.rw.insertBlock(bs)
	if err != nil {
		return 0, err
	}

	cp.insertNotify(bs)

	if err != nil {
		return 0, err
	}
	return 1, nil
}

func (accP *accountPool) findInPool(hash types.Hash, height uint64) bool {
	accP.blockpool.pendingMu.Lock()
	defer accP.blockpool.pendingMu.Unlock()
	return accP.blockpool.containsHash(hash)
}

func (accP *accountPool) findInTree(hash types.Hash, height uint64) tree.Branch {
	return accP.chainpool.tree.FindBranch(height, hash)
}

func (accP *accountPool) findInTreeDisk(hash types.Hash, height uint64, disk bool) tree.Branch {
	cur := accP.CurrentChain()
	targetHash := cur.GetHash(height, disk)
	if targetHash != nil && *targetHash == hash {
		return cur
	}

	for _, c := range accP.chainpool.allChain() {
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

func (accP *accountPool) AddDirectBlocks(received *accountPoolBlock) error {
	latestSb := accP.rw.getLatestSnapshotBlock()
	//accP.rMu.Lock()
	//defer accP.rMu.Unlock()
	accP.chainHeadMu.Lock()
	defer accP.chainHeadMu.Unlock()

	accP.chainTailMu.Lock()
	defer accP.chainTailMu.Unlock()

	current := accP.CurrentChain()
	tailHeight, tailHash := current.TailHH()
	if received.Height() != tailHeight+1 ||
		received.PrevHash() != tailHash {
		return errors.Errorf("account head not match[%d-%s][%s]", received.Height(), received.PrevHash(), current.SprintTail())
	}

	accP.checkCurrent()
	stat := accP.v.verifyAccount(received, latestSb)
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

		accP.log.Debug("AddDirectBlocks", "height", received.Height(), "hash", received.Hash())
		_, err := accP.verifySuccess(stat.block)
		if err != nil {
			return err
		}
		return nil
	default:
		accP.log.Crit("verify unexpected.")
		return errors.New("verify unexpected")
	}
}

func (accP *accountPool) getCurrentBlock(i uint64) *accountPoolBlock {
	b := accP.chainpool.tree.Main().GetKnot(i, false)
	if b != nil {
		return b.(*accountPoolBlock)
	}
	return nil
}

var ErrEmptyMain = errors.New("empty chainpool")
var ErrCurrentChainModify = errors.New("current chain modify")
var ErrMax = errors.New("arrived to max")
var ErrBlackList = errors.New("block in blacklist")
var ErrQuotaNotEnough = errors.New("block quota not enough")
var ErrAllIn = errors.New("all in")

func (accP *accountPool) makePackage(q batch.Batch, info *offsetInfo, max uint64) (uint64, error) {
	// if current size is empty, do nothing.
	if accP.chainpool.tree.Main().Size() <= 0 {
		return 0, ErrEmptyMain
	}

	// lock other chain insert
	accP.pool.RLockInsert()
	defer accP.pool.RUnLockInsert()

	accP.chainHeadMu.Lock()
	defer accP.chainHeadMu.Unlock()

	accP.chainTailMu.Lock()
	defer accP.chainTailMu.Unlock()

	// choose current
	current := accP.chooseAndSwitchCurrentForMake(info)

	if info.offset == nil {
		tailHeight, tailHash := current.TailHH()
		info.offset = &ledger.HashHeight{Hash: tailHash, Height: tailHeight}
		info.quotaUnused = accP.rw.getQuotaUnused()
	} else {
		block := current.GetKnot(info.offset.Height+1, false)
		if block == nil || block.PrevHash() != info.offset.Hash {
			return uint64(0), ErrCurrentChainModify
		}
	}

	minH := info.offset.Height + 1
	headH, _ := current.HeadHH()
	for i := minH; i <= headH; i++ {
		if i-minH >= max {
			return uint64(i - minH), ErrMax
		}
		block := accP.getCurrentBlock(i)
		if block == nil {
			return uint64(i - minH), ErrCurrentChainModify
		}
		if accP.hashBlacklist.Exists(block.Hash()) {
			return uint64(i - minH), ErrBlackList
		}
		// check quota
		used, unused, enought := info.quotaEnough(block)
		if !enought {
			// todo remove
			return uint64(i - minH), ErrQuotaNotEnough
		}
		accP.log.Debug(fmt.Sprintf("[%s][%d][%s]quota info [used:%d][unused:%d]\n", block.block.AccountAddress, block.Height(), block.Hash(), used, unused))
		// check request block confirmed time for response block
		if err := accP.checkSnapshotSuccess(block); err != nil {
			return uint64(i - minH), err
		}

		err := q.AddAItem(block, nil)
		if err != nil {
			return uint64(i - minH), err
		}
		info.offset.Hash = block.Hash()
		info.offset.Height = block.Height()
		info.quotaSub(block)
	}

	return uint64(headH - minH), ErrAllIn
}

// choose branch, add random strategy
func (accP *accountPool) chooseAndSwitchCurrentForMake(info *offsetInfo) tree.Branch {
	main := accP.chainpool.tree.Main()
	if info.offset == nil {
		return main
	}
	uTime := main.UTime()
	if time.Now().After(uTime.Add(time.Second * 5)) {
		brothers := accP.chainpool.tree.Brothers(main)
		if len(brothers) == 0 {
			return main
		}
		randN := rand.Intn(len(brothers))
		branch := brothers[randN]
		accP.log.Info("current modify for random", "targetId", branch.ID(), "TargetTail", branch.SprintTail(), "currentId", main.ID(), "CurrentTail", main.SprintTail())
		err := accP.chainpool.tree.SwitchMainTo(branch)
		if err != nil {
			accP.log.Error("switch main fail", "err", err)
			return accP.chainpool.tree.Main()
		}
		return accP.chainpool.tree.Main()
	}
	return main
}

func (accP *accountPool) tryInsertItems(p batch.Batch, items []batch.Item, latestSb *ledger.SnapshotBlock, version uint64) error {
	accP.chainTailMu.Lock()
	defer accP.chainTailMu.Unlock()

	cp := accP.chainpool

	for i := 0; i < len(items); i++ {
		item := items[i]
		block := item.(*accountPoolBlock)
		accP.log.Info(fmt.Sprintf("[%d]try to insert account block[%d-%s]%d-%d.", p.Id(), block.Height(), block.Hash(), i, len(items)))
		current := cp.tree.Root()
		tailHeight, tailHash := current.HeadHH()
		if block.Height() == tailHeight+1 &&
			block.PrevHash() == tailHash {
			block.resetForkVersion()
			if block.forkVersion() != version {
				return errors.New("snapshot version update")
			}

			stat := accP.v.verifyAccount(block, latestSb)
			if !block.checkForkVersion() {
				block.resetForkVersion()
				return errors.New("new fork version")
			}
			switch stat.verifyResult() {
			case verifier.FAIL:
				accP.log.Warn("add account block to blacklist.", "hash", block.Hash(), "height", block.Height(), "err", stat.err)
				accP.hashBlacklist.AddAddTimeout(block.Hash(), time.Second*10)
				return errors.Wrap(stat.err, "fail verifier")
			case verifier.PENDING:
				accP.log.Error("snapshot db.", "hash", block.Hash(), "height", block.Height())
				return errors.Wrap(stat.err, "fail verifier db.")
			}
			err := cp.writeBlockToChain(stat.block)
			if err != nil {
				accP.log.Error("account block write fail. ",
					"hash", block.Hash(), "height", block.Height(), "error", err)
				return err
			}
		} else {
			return errors.New("tail not match")
		}
		accP.log.Info(fmt.Sprintf("[%d]try to insert account block[%d-%s]%d-%d [latency:%s]success.", p.Id(), block.Height(), block.Hash(), i, len(items), block.Latency()))
	}
	return nil
}
func (accP *accountPool) checkSnapshotSuccess(block *accountPoolBlock) error {
	if block.block.IsReceiveBlock() {
		if !types.IsContractAddr(block.block.AccountAddress) {
			return nil
		}
		num, e := accP.rw.needSnapshot(block.block.AccountAddress)
		if e != nil {
			return e
		}
		if num > 0 {
			b, err := accP.rw.getConfirmedTimes(block.block.FromBlockHash)
			if err != nil {
				return err
			}
			if b >= uint64(num) {
				return nil
			}
			return errors.New("send block need to snapshot")
		}
	}
	return nil
}
func (accP *accountPool) genForSnapshotContents(p batch.Batch, b *snapshotPoolBlock, k types.Address, v *ledger.HashHeight) (bool, *stack.Stack) {
	accP.chainTailMu.Lock()
	defer accP.chainTailMu.Unlock()
	acurr := accP.CurrentChain()
	tailHeight, _ := acurr.TailHH()
	targetHash := acurr.GetHash(v.Height, true)
	if targetHash == nil {
		return true, nil
	}
	if *targetHash != v.Hash {
		accP.log.Info(fmt.Sprintf("account chain has forked. snapshot block[%d-%s], account block[%s-%d][%s<->%s]\n",
			b.block.Height, b.block.Hash, k, v.Height, v.Hash, *targetHash))
		// todo switch account chain

		return true, nil
	}

	if v.Height > tailHeight {
		// account block is in pool.
		tmp := stack.New()
		for h := v.Height; h > tailHeight; h-- {
			currB := accP.getCurrentBlock(h)
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

func (accP *accountPool) shouldDestroy() bool {
	accP.chainHeadMu.Lock()
	defer accP.chainHeadMu.Unlock()

	accP.chainTailMu.Lock()
	defer accP.chainTailMu.Unlock()
	if accP.blockpool.size() > 0 {
		return false
	}

	if len(accP.chainpool.snippetChains) > 0 {
		return false
	}

	if accP.chainpool.tree.Size() > 0 {
		return false
	}
	if !time.Now().After(accP.chainpool.tree.Main().UTime().Add(time.Minute * 8)) {
		return false
	}
	return true
}
