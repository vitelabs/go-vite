package pool

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/golang-collections/collections/stack"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

/**
loop:
1. make a queue for blocks plan
2. insert for queue
*/
func (self *pool) loopQueue() {
	for {
		q := self.makeQueue()
		size := q.Size()
		if size == 0 {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		err := self.insertQueue(q)
		if err != nil {
			fmt.Printf("insert queue err:%s\n", err)
			fmt.Printf("all queue:%s\n", q.Info())
			time.Sleep(time.Second * 2)
			os.Exit(0)
		}
	}
}

/**
make a queue from account pool and snapshot pool
*/
func (self *pool) makeQueue() Package {
	snapshotOffset := &offsetInfo{offset: &ledger.HashHeight{Height: self.pendingSc.CurrentChain().tailHeight, Hash: self.pendingSc.CurrentChain().tailHash}}

	p := NewSnapshotPackage(self.snapshotExists, self.accountExists, 50)
	for {
		newOffset, errAcc, tmpSb := self.makeSnapshotBlock(p, snapshotOffset)
		if tmpSb == nil {
			// just account
			if p.Size() > 0 {
				break
			}

			if errAcc != nil && rand.Intn(10) > 3 {
				self.snapshotPendingFix(newOffset, errAcc)
			} else {
				self.makeQueueFromAccounts(p)
			}
			break
		} else { // snapshot block
			err := self.makeQueueFromSnapshotBlock(p, tmpSb)
			if err != nil {
				fmt.Println("from snapshot", err)
				break
			}
			snapshotOffset.offset = newOffset
		}
	}
	return p
}

type completeSnapshotBlock struct {
	cur   *snapshotPoolBlock
	addrM map[types.Address]*stack.Stack
}

func (self *completeSnapshotBlock) isEmpty() bool {
	if len(self.addrM) > 0 {
		for _, v := range self.addrM {
			if v.Len() > 0 {
				return false
			}
		}
	}
	return true
}

func (self *pool) makeSnapshotBlock(p Package, info *offsetInfo) (*ledger.HashHeight, map[types.Address]*ledger.HashHeight, *completeSnapshotBlock) {
	if self.pendingSc.CurrentChain().size() == 0 {
		return nil, nil, nil
	}
	current := self.pendingSc.CurrentChain()
	block := current.getBlock(info.offset.Height+1, false)
	if block == nil {
		return nil, nil, nil
	}
	newOffset := &ledger.HashHeight{Hash: block.Hash(), Height: block.Height()}
	b := block.(*snapshotPoolBlock)
	result := &completeSnapshotBlock{cur: b}
	contents := b.block.SnapshotContent

	errorAcc := make(map[types.Address]*ledger.HashHeight)

	pending := false
	addrM := make(map[types.Address]*stack.Stack)
	for k, v := range contents {
		ac := self.selfPendingAc(k)
		acurr := ac.CurrentChain()
		ab := acurr.getBlock(v.Height, true)
		if ab == nil {
			errorAcc[k] = v
			pending = true
			continue
		}
		if ab.Hash() != v.Hash {
			fmt.Printf("account chain has forked. snapshot block[%d-%s], account block[%s-%d][%s<->%s]\n",
				b.block.Height, b.block.Hash, k, v.Height, v.Hash, ab.Hash())
			// todo switch account chain
			errorAcc[k] = v
			pending = true
			continue
		}
		if pending {
			continue
		}
		if ab.Height() > acurr.tailHeight {
			tmp := stack.New()
			for h := ab.Height(); h > acurr.tailHeight; h-- {
				currB := ac.getCurrentBlock(h)
				if p.Exists(currB.Hash()) {
					break
				}
				tmp.Push(currB)
			}
			if tmp.Len() > 0 {
				addrM[k] = tmp
			}
		}
	}

	if len(errorAcc) > 0 {
		return newOffset, errorAcc, nil
	}
	result.addrM = addrM
	return newOffset, nil, result
}

func (self *pool) makeQueueFromSnapshotBlock(p Package, b *completeSnapshotBlock) error {
	sum := 0
	for {
		for _, v := range b.addrM {
			for v.Len() > 0 {
				ab := v.Peek().(*accountPoolBlock)
				item := NewItem(ab, &ab.block.AccountAddress)
				err := p.AddItem(item)
				if err != nil {
					if err == MAX_ERROR {
						fmt.Printf("account[%s] max. %s\n", item.Hash(), err)
						return err
					}
					fmt.Printf("account[%s] add fail. %s\n", item.Hash(), err)
					break
				}
				sum += 1
				v.Pop()
			}
		}
		if sum == 0 {
			break
		} else {
			sum = 0
		}
	}
	if b.isEmpty() {
		item := NewItem(b.cur, nil)
		err := p.AddItem(item)
		if err != nil {
			fmt.Printf("add snapshot[%s] error. %s\n", item.Hash(), err)
			return err
		}
		return nil
	} else {
		return errors.WithMessage(REFER_ERROR, fmt.Sprintf("snapshot[%s] not finish.", b.cur.block.Hash))
	}
}
func (self *pool) makeQueueFromAccounts(p Package) {
	addrOffsets := make(map[types.Address]*offsetInfo)
	for {
		sum := uint64(0)
		self.pendingAc.Range(func(key, v interface{}) bool {
			cp := v.(*accountPool)
			offset := addrOffsets[key.(types.Address)]
			if offset == nil {
				offset = &offsetInfo{}
				addrOffsets[key.(types.Address)] = offset
			}

			num, _ := cp.makePackage(p, offset)
			sum += num
			return true
		})
		if sum == 0 {
			break
		}
	}
}

//func (self *pool) makePackages() (Packages, error) {
//	q := NewSnapshotPackage(self.snapshotExists, self.accountExists, 50)
//	addrOffsets := make(map[types.Address]*offsetInfo)
//	snapshotOffset := &offsetInfo{}
//
//	spkg, err := self.pendingSc.makePackage(self.snapshotExists, self.accountExists, snapshotOffset)
//	if err != nil {
//		return nil, err
//	}
//
//	for {
//		sum := uint64(0)
//		self.pendingAc.Range(func(key, v interface{}) bool {
//			cp := v.(*accountPool)
//			offset := addrOffsets[key.(types.Address)]
//			if offset == nil {
//				offset = &offsetInfo{}
//				addrOffsets[key.(types.Address)] = offset
//			}
//			num, _ := cp.makePackage(q, offset)
//			sum += num
//			return true
//		})
//
//		num, _ := self.pendingSc.makeQueue(q, snapshotOffset)
//		sum += num
//		if sum == 0 {
//			break
//		}
//	}
//	return q
//}

func (self *pool) insertQueue(q Package) error {
	levels := q.Levels()
	t0 := time.Now()

	defer func() {
		sub := time.Now().Sub(t0)
		queueResult := fmt.Sprintf("queue[%s][%d][%d]", sub, (int64(q.Size())*time.Second.Nanoseconds())/sub.Nanoseconds(), q.Size())
		fmt.Println(queueResult)
	}()

	for _, level := range levels {
		if level == nil {
			continue
		}
		err := self.insertLevel(level)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *pool) insertAccountBucket(bucket Bucket) error {
	latestSb := self.bc.GetLatestSnapshotBlock()
	err := self.selfPendingAc(*bucket.Owner()).tryInsertItems(bucket.Items(), latestSb)
	if err != nil {
		return err
	}
	return nil
}

func (self *pool) insertSnapshotBucket(bucket Bucket) error {
	// stop the world for snapshot insert
	self.Lock()
	defer self.UnLock()
	accBlocks, item, err := self.pendingSc.snapshotInsertItems(bucket.Items())
	if err != nil {
		return err
	}
	if accBlocks == nil || len(accBlocks) == 0 {
		return nil
	}

	for k, v := range accBlocks {
		err := self.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
	}
	return errors.Errorf("account blocks rollback for snapshot block[%s-%d] insert.", item.Hash(), item.Height())
}

var NotFound = errors.New("Not Found")

func (self *pool) accountExists(hash types.Hash) error {
	ab, err := self.bc.GetAccountBlockByHash(&hash)
	if err != nil {
		return err
	}
	if ab != nil {
		return nil
	}
	return NotFound
}

func (self *pool) accountBlockCheckAndFetch(snapshot *snapshotPoolBlock, hashH *ledger.HashHeight, address types.Address) (*ledger.HashHeight, error) {
	err := self.accountExists(hashH.Hash)
	if err == nil {
		return nil, nil
	}
	if err != nil && err != NotFound {
		return nil, err
	}
	hashHeight, err := self.PendingAccountTo(address, hashH, snapshot.Height())

	return hashHeight, err
}
func (self *pool) snapshotExists(hash types.Hash) error {
	sb, err := self.bc.GetSnapshotBlockByHash(&hash)
	if err != nil {
		return err
	}
	if sb != nil {
		return nil
	}
	return errors.New("Not Found")
}

type offsetInfo struct {
	offset      *ledger.HashHeight
	quotaUnused uint64
}

func (self offsetInfo) quotaEnough(b commonBlock) bool {
	accB := b.(*accountPoolBlock)
	quotaUsed := accB.block.Quota
	if quotaUsed > self.quotaUnused {
		return false
	}
	return true
}
func (self offsetInfo) quotaSub(b commonBlock) {
	accB := b.(*accountPoolBlock)
	quotaUsed := accB.block.Quota
	if quotaUsed > self.quotaUnused {
		self.quotaUnused = self.quotaUnused - quotaUsed
	} else {
		self.quotaUnused = 0
	}
}

// todo fix: not in same thread with loopCompact
func (self *pool) loopPendingSnapshot() {
	snapshots, err := self.pendingSc.getPendingForCurrent()
	if err != nil {
		self.log.Error("getPendingForCurrent error.", "error", err)
	}

	accounts := make(map[types.Address]*ledger.HashHeight)

	for _, b := range snapshots {
		block := b.(*snapshotPoolBlock)
		for addr, hashH := range block.block.SnapshotContent {
			hashH, err := self.accountBlockCheckAndFetch(block, hashH, addr)
			if err != nil {
				self.log.Error("account block check and fetch fail.", "err", err, "addr", addr, "hash", hashH.Hash, "height", hashH.Height)
			}

			_, ok := accounts[addr]
			if hashH != nil && !ok {
				accounts[addr] = hashH
			}
		}
	}
	if len(accounts) > 0 {
		self.pendingSc.forkAccounts(accounts)

		// todo fix
		self.pendingSc.fetchAccounts(accounts, snapshots[len(snapshots)-1].Height())
	}
}
