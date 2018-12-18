package pool

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func (self *pool) loopQueue() {
	for {
		q := self.makeQueue()
		size := q.Size()
		if size == 0 {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		self.insert(q, 5)
		//fmt.Println(q.Info())
	}
}

func (self *pool) makeQueue() Queue {
	q := NewQueue(self.snapshotExists, self.accountExists, 50)
	addrOffsets := make(map[types.Address]*offsetInfo)
	snapshotOffset := &offsetInfo{}

	for {
		sum := uint64(0)
		self.pendingAc.Range(func(key, v interface{}) bool {
			cp := v.(*accountPool)
			offset := addrOffsets[key.(types.Address)]
			if offset == nil {
				offset = &offsetInfo{}
				addrOffsets[key.(types.Address)] = offset
			}
			num, _ := cp.makeQueue(q, offset)
			sum += num
			return true
		})

		num, _ := self.pendingSc.makeQueue(q, snapshotOffset)
		sum += num
		if sum == 0 {
			break
		}
	}
	return q
}

func (self *pool) insert(q Queue, N int) {
	var wg sync.WaitGroup
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

		var closedOnce sync.Once
		closed := make(chan struct{})
		var num int32
		t1 := time.Now()
		for i := 0; i < N; i++ {
			common.Go(func() {
				defer wg.Done()
				for b := range bucketCh {
					select {
					case <-closed:
						return
					default:
						atomic.AddInt32(&num, int32(len(b.Items())))
						err := self.insertBucket(b)
						if err != nil {
							closedOnce.Do(func() {
								self.log.Error("err insert", "err", err)
								close(closed)
							})
							return
						}
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
		fmt.Println(levelInfo)
		select {
		case <-closed:
			return
		default:
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

func (self *pool) accountExists(hash types.Hash) error {
	ab, err := self.bc.GetAccountBlockByHash(&hash)
	if err != nil {
		return err
	}
	if ab != nil {
		return nil
	}
	return errors.New("Not Found")
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
