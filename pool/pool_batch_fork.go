package pool

import (
	"fmt"
	"time"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pool/batch"
)

func (pl *pool) insertTo(height uint64) error {
	for {
		err := pl.checkTarget(height)
		if err == nil {
			return nil
		}
		t1 := time.Now()
		q := pl.makeQueueOnly()
		size := q.Size()
		if size == 0 {
			return pl.checkTarget(height)
		}
		err = pl.insertQueueForFork(q)
		if err != nil {
			pl.log.Error(fmt.Sprintf("insert queue err:%s\n", err))
			pl.log.Error(fmt.Sprintf("all queue:%s\n", q.Info()))
			e := pl.checkTarget(height)
			if e != nil {
				tailHeight, _ := pl.pendingSc.CurrentChain().TailHH()
				block := pl.pendingSc.getCurrentBlock(tailHeight + 1)
				if block != nil {
					pl.hashBlacklist.AddAddTimeout(block.block.Hash, time.Minute*30)
				}
			}
			return e
		}
		t2 := time.Now()
		pl.log.Info(fmt.Sprintf("time duration:%s, size:%d", t2.Sub(t1), size))
	}
}

func (pl *pool) insertQueueForFork(q batch.Batch) error {
	t0 := time.Now()
	defer func() {
		sub := time.Now().Sub(t0)
		queueResult := fmt.Sprintf("[%d][fork] queue[%s][%d][%d]", q.Id(), sub, (int64(q.Size())*time.Second.Nanoseconds())/sub.Nanoseconds(), q.Size())
		fmt.Println(queueResult)
	}()
	return q.Batch(pl.insertSnapshotBucketForFork, pl.insertAccountsBucketForFork)
}

func (pl *pool) insertSnapshotBucketForFork(p batch.Batch, bucket batch.Bucket, version uint64) error {
	return pl.insertSnapshotBucket(p, bucket, version)
}

func (pl *pool) insertAccountsBucketForFork(p batch.Batch, bucket batch.Bucket, version uint64) error {
	return pl.insertAccountBucket(p, bucket, version)
}

func (pl *pool) checkTarget(height uint64) error {
	curHeight, _ := pl.pendingSc.CurrentChain().TailHH()
	if curHeight >= height {
		return nil
	}
	return errors.Errorf("target fail.[%d][%d]", height, curHeight)
}

func (pl *pool) makeQueueOnly() batch.Batch {
	tailHeight, tailHash := pl.pendingSc.CurrentChain().TailHH()
	snapshotOffset := &offsetInfo{offset: &ledger.HashHeight{Height: tailHeight, Hash: tailHash}}

	p := batch.NewBatch(pl.snapshotExists, pl.accountExists, pl.version.Val(), 50)
	for {
		newOffset, _, tmpSb := pl.makeSnapshotBlock(p, snapshotOffset)
		if tmpSb == nil {
			if p.Size() > 0 {
				break
			} else {
				return p
			}
		} else {
			// snapshot block
			err := pl.makeQueueFromSnapshotBlock(p, tmpSb)
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
		pl.log.Info(msg)
	}
	return p
}
