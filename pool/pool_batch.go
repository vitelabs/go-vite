package pool

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/vitelabs/go-vite/pool/batch"

	"github.com/golang-collections/collections/stack"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

/**
loop:
1. make a batch for blocks plan
2. insert batch
*/
func (self *pool) insert() {
	t1 := time.Now()
	q := self.makeQueue()
	size := q.Size()
	if size == 0 {
		return
	}
	err := self.insertQueue(q)
	if err != nil {
		self.log.Error(fmt.Sprintf("insert queue err:%s\n", err))
		self.log.Error(fmt.Sprintf("all queue:%s\n", q.Info()))
		//time.Sleep(time.Second
		//self.log.Error("pool auto stop")
	}
	t2 := time.Now()
	self.log.Info(fmt.Sprintf("time duration:%s, size:%d", t2.Sub(t1), size))
}

/**
make a queue from account pool and snapshot pool
*/
func (self *pool) makeQueue() batch.Batch {
	tailHeight, tailHash := self.pendingSc.CurrentChain().TailHH()
	snapshotOffset := &offsetInfo{offset: &ledger.HashHeight{Height: tailHeight, Hash: tailHash}}

	p := batch.NewBatch(self.snapshotExists, self.accountExists, self.version.Val(), 50)
	for {
		newOffset, pendingForSb, tmpSb := self.makeSnapshotBlock(p, snapshotOffset)
		if tmpSb == nil {
			// just account
			if p.Size() > 0 {
				break
			}

			if pendingForSb != nil && rand.Intn(10) > 3 {
				self.snapshotPendingFix(p, newOffset, pendingForSb)
			} else {
				self.makeQueueFromAccounts(p)
				if p.Size() > 0 {
					// todo remove
					msg := fmt.Sprintf("[%d]just make accounts[%d].", p.Id(), p.Size())
					fmt.Println(msg)
					self.log.Info(msg)
					return p
				}
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
	if p.Size() > 0 {
		msg := fmt.Sprintf("[%d]make from snapshot, accounts[%d].", p.Id(), p.Size())
		fmt.Println(msg)
		self.log.Info(msg)
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

type snapshotPending struct {
	snapshot *snapshotPoolBlock
	addrM    map[types.Address]*ledger.HashHeight
}

func (self *pool) makeSnapshotBlock(p batch.Batch, info *offsetInfo) (*ledger.HashHeight, *snapshotPending, *completeSnapshotBlock) {
	if self.pendingSc.CurrentChain().Size() == 0 {
		return nil, nil, nil
	}
	current := self.pendingSc.CurrentChain()
	block := current.GetKnot(info.offset.Height+1, false)
	if block == nil {
		return nil, nil, nil
	}
	// fork happen
	if block.PrevHash() != info.offset.Hash {
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
		var tmp *stack.Stack
		pending, tmp = ac.genForSnapshotContents(p, b, k, v)
		if pending {
			errorAcc[k] = v
			continue
		}
		if tmp != nil {
			addrM[k] = tmp
		}
	}

	if len(errorAcc) > 0 {
		pendingS := &snapshotPending{snapshot: b, addrM: errorAcc}
		return newOffset, pendingS, nil

	}
	result.addrM = addrM
	return newOffset, nil, result
}

func (self *pool) makeQueueFromSnapshotBlock(p batch.Batch, b *completeSnapshotBlock) error {
	sum := 0
	for {
		for _, v := range b.addrM {
			for v.Len() > 0 {
				ab := v.Peek().(*accountPoolBlock)
				err := p.AddItem(ab)
				if err != nil {
					if err == batch.MAX_ERROR {
						fmt.Printf("account[%s] max. %s\n", ab.Hash(), err)
						return err
					}
					fmt.Printf("account[%s] add fail. %s\n", ab.Hash(), err)
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
		err := p.AddItem(b.cur)
		if err != nil {
			fmt.Printf("add snapshot[%s] error. %s\n", b.cur.Hash(), err)
			return err
		}
		return nil
	} else {
		return errors.WithMessage(batch.REFER_ERROR, fmt.Sprintf("snapshot[%s] not finish.", b.cur.block.Hash))
	}
}
func (self *pool) makeQueueFromAccounts(p batch.Batch) {
	addrOffsets := make(map[types.Address]*offsetInfo)
	max := uint64(100)
	total := uint64(0)
	for {
		sum := uint64(0)
		self.pendingAc.Range(func(key, v interface{}) bool {
			cp := v.(*accountPool)
			offset := addrOffsets[key.(types.Address)]
			if offset == nil {
				offset = &offsetInfo{}
				addrOffsets[key.(types.Address)] = offset
			}
			if total >= max {
				return false
			}
			num, _ := cp.makePackage(p, offset, max-total)
			sum += num
			total += num
			return true
		})
		if total >= max {
			break
		}
		if sum == 0 {
			break
		}
	}
}

func (self *pool) insertQueue(q batch.Batch) error {
	t0 := time.Now()
	defer func() {
		sub := time.Now().Sub(t0)
		queueResult := fmt.Sprintf("[%d]queue[%s][%d][%d]", q.Id(), sub, (int64(q.Size())*time.Second.Nanoseconds())/sub.Nanoseconds(), q.Size())
		fmt.Println(queueResult)
	}()
	return q.Batch(self.insertSnapshotBucketForTree, self.insertAccountBucketForTree)
}

func (self *pool) insertSnapshotBucketForTree(p batch.Batch, bucket batch.Bucket, version uint64) error {
	// stop the world for snapshot insert
	self.LockInsert()
	defer self.UnLockInsert()
	return self.insertSnapshotBucket(p, bucket, version)
}

func (self *pool) insertAccountBucketForTree(p batch.Batch, bucket batch.Bucket, version uint64) error {
	self.RLockInsert()
	defer self.RUnLockInsert()
	return self.insertAccountBucket(p, bucket, version)
}

func (self *pool) insertAccountBucket(p batch.Batch, bucket batch.Bucket, version uint64) error {
	self.RLockInsert()
	defer self.RUnLockInsert()
	latestSb := self.bc.GetLatestSnapshotBlock()
	err := self.selfPendingAc(*bucket.Owner()).tryInsertItems(p, bucket.Items(), latestSb, version)
	if err != nil {
		return err
	}
	return nil
}

func (self *pool) insertSnapshotBucket(p batch.Batch, bucket batch.Bucket, version uint64) error {
	accBlocks, item, err := self.pendingSc.snapshotInsertItems(p, bucket.Items(), version)
	if err != nil {
		return err
	}
	self.pendingSc.checkCurrent()
	if accBlocks == nil || len(accBlocks) == 0 {
		return nil
	}

	for k, v := range accBlocks {
		err := self.selfPendingAc(k).rollbackCurrent(v)
		if err != nil {
			return err
		}
		self.selfPendingAc(k).checkCurrent()
	}
	return errors.Errorf("account blocks rollback for snapshot block[%s-%d] insert.", item.Hash(), item.Height())
}

var NotFound = errors.New("Not Found")

func (self *pool) accountExists(hash types.Hash) error {
	ab, err := self.bc.GetAccountBlockByHash(hash)
	if err != nil {
		return err
	}
	if ab != nil {
		return nil
	}
	return NotFound
}

func (self *pool) snapshotExists(hash types.Hash) error {
	sb, err := self.bc.GetSnapshotHeaderByHash(hash)
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

func (self offsetInfo) quotaEnough(b commonBlock) (uint64, uint64, bool) {
	accB := b.(*accountPoolBlock)
	quotaUsed := accB.block.Quota
	if quotaUsed > self.quotaUnused {
		return quotaUsed, self.quotaUnused, false
	}
	return quotaUsed, self.quotaUnused, true
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
