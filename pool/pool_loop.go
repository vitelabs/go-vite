package pool

import (
	"fmt"
	"math/rand"
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
	self.wg.Add(1)
	defer self.wg.Done()
	for {
		select {
		case <-self.closed:
			return
		default:
			t1 := time.Now()
			q := self.makeQueue()
			size := q.Size()
			if size == 0 {
				time.Sleep(2 * time.Millisecond)
				continue
			}
			err := self.insertQueue(q)
			if err != nil {
				self.log.Error(fmt.Sprintf("insert queue err:%s\n", err))
				self.log.Error(fmt.Sprintf("all queue:%s\n", q.Info()))
				//time.Sleep(time.Second
				self.log.Error("pool auto stop")
			}
			t2 := time.Now()
			self.log.Info(fmt.Sprintf("time duration:%s, size:%d", t2.Sub(t1), size))
		}
	}
}

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
func (self *pool) makeQueue() Package {
	tailHeight, tailHash := self.pendingSc.CurrentChain().TailHH()
	snapshotOffset := &offsetInfo{offset: &ledger.HashHeight{Height: tailHeight, Hash: tailHash}}

	p := NewSnapshotPackage(self.snapshotExists, self.accountExists, self.version.Val(), 50)
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
	sHash     types.Hash
	sHeight   uint64
	sPrevHash types.Hash
	addrM     map[types.Address]*ledger.HashHeight
}

func (self *pool) makeSnapshotBlock(p Package, info *offsetInfo) (*ledger.HashHeight, *snapshotPending, *completeSnapshotBlock) {
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
		pendingS := &snapshotPending{sHash: b.Hash(), sHeight: b.Height(), sPrevHash: b.PrevHash(), addrM: errorAcc}
		return newOffset, pendingS, nil

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
		queueResult := fmt.Sprintf("[%d]queue[%s][%d][%d]", q.Id(), sub, (int64(q.Size())*time.Second.Nanoseconds())/sub.Nanoseconds(), q.Size())
		fmt.Println(queueResult)
	}()

	for i, level := range levels {
		if level == nil {
			continue
		}
		self.log.Info(fmt.Sprintf("[%d]insert queue level[%d][%t] insert.", q.Id(), i, level.Snapshot()))
		err := self.insertLevel(q, level, q.Version())
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *pool) insertAccountBucket(p Package, bucket Bucket, version int) error {
	self.RLock()
	defer self.RUnLock()
	latestSb := self.bc.GetLatestSnapshotBlock()
	err := self.selfPendingAc(*bucket.Owner()).tryInsertItems(p, bucket.Items(), latestSb, version)
	if err != nil {
		return err
	}
	return nil
}

func (self *pool) insertSnapshotBucket(p Package, bucket Bucket, version int) error {
	// stop the world for snapshot insert
	self.Lock()
	defer self.UnLock()
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
	return errors.Errorf("account blocks rollback for snapshot block[%s-%s-%d] insert.", item.ownerWrapper, item.Hash(), item.Height())
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
