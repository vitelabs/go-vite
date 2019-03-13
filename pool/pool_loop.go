package pool

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-collections/collections/stack"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/helper"
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
		self.insertQueue(q, 5)
		//fmt.Println(q.Info())
	}
}

/**
make a queue from account pool and snapshot pool
*/
func (self *pool) makeQueue() Package {
	snapshotOffset := &offsetInfo{offset: &ledger.HashHeight{Height: self.pendingSc.CurrentChain().tailHeight, Hash: self.pendingSc.CurrentChain().tailHash}}

	p := NewSnapshotPackage(self.snapshotExists, self.accountExists, 50)
	for {
		tmpSb := self.makeSnapshotBlock(snapshotOffset)
		if tmpSb == nil {
			self.makeQueueFromAccounts(p)
			break
		} else {
			err := self.makeQueueFromSnapshotBlock(p, tmpSb)
			if err != nil {
				snapshotOffset.offset = &ledger.HashHeight{Hash: tmpSb.cur.block.Hash, Height: tmpSb.cur.block.Height}
			} else {
				break
			}
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

func (self *pool) makeSnapshotBlock(info *offsetInfo) *completeSnapshotBlock {
	if self.pendingSc.CurrentChain().size() == 0 {
		return nil
	}
	current := self.pendingSc.CurrentChain()
	block := current.getBlock(info.offset.Height+1, false)
	if block == nil {
		return nil
	}
	b := block.(*snapshotPoolBlock)
	result := &completeSnapshotBlock{cur: b}
	contents := b.block.SnapshotContent

	addrM := make(map[types.Address]*stack.Stack)
	for k, v := range contents {
		ac := self.selfPendingAc(k)
		acurr := ac.CurrentChain()
		ab := acurr.getBlock(v.Height, true)
		if ab == nil {
			return nil
		}
		if ab.Height() > acurr.tailHeight {
			tmp := stack.New()
			for h := ab.Height(); h > acurr.tailHeight; h-- {
				tmp.Push(ac.getCurrentBlock(h))
			}
			if tmp.Len() > 0 {
				addrM[k] = tmp
			}
		}
	}
	result.addrM = addrM
	return result
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
			return err
		}
		return nil
	} else {
		return REFER_ERROR
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

func (self *pool) insertQueue(q Package, N int) {
	var wg sync.WaitGroup
	closed := NewClosed()
	levels := q.Levels()
	for _, level := range levels {
		bs := level.Buckets()
		lenBs := len(bs)
		if lenBs == 0 {
			return
		}
		N = helper.MinInt(lenBs, 5)
		bucketCh := make(chan Bucket, lenBs)

		wg.Add(N)

		var num int32
		t1 := time.Now()
		for i := 0; i < N; i++ {
			common.Go(func() {
				defer wg.Done()
				for b := range bucketCh {
					if closed.IsClosed() {
						return
					}
					atomic.AddInt32(&num, int32(len(b.Items())))
					err := self.insertBucket(b)
					if err != nil {
						closed.Close()
						return
					}
				}
			})
		}

		levelInfo := ""
		for _, bucket := range bs {
			levelInfo += "|" + strconv.Itoa(len(bucket.Items()))
			if bucket.Owner() == nil {
				levelInfo += "S"
			}

			bucketCh <- bucket
		}
		close(bucketCh)
		wg.Wait()
		sub := time.Now().Sub(t1)
		levelInfo = "[" + sub.String() + "][" + strconv.Itoa(int((int64(num)*time.Second.Nanoseconds())/sub.Nanoseconds())) + "]" + "[" + strconv.Itoa(int(num)) + "]" + "->" + levelInfo
		//fmt.Println(levelInfo)
		if closed.IsClosed() {
			return
		}
	}
}

func (self *pool) insertBucket(bucket Bucket) error {
	owner := bucket.Owner()
	if owner == nil {
		err := self.pendingSc.snapshotTryInsertItems(bucket.Items())
		if err != nil {
			return err
		}
	} else {
		err := self.selfPendingAc(*owner).tryInsertItems(bucket.Items())
		if err != nil {
			return err
		}
	}
	return nil
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
	offset *ledger.HashHeight
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
