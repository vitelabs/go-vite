package batch

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/log15"
)

type batchExecutor struct {
	p           Batch
	snapshotFn  BucketExecutorFn
	accountFn   BucketExecutorFn
	maxParallel int
	log         log15.Logger
}

func newBatchExecutor(p Batch, snapshotFn BucketExecutorFn, accountFn BucketExecutorFn) *batchExecutor {
	executor := &batchExecutor{p: p, snapshotFn: snapshotFn, accountFn: accountFn}
	executor.log = log15.New("module", "pool/batch", "batchId", p.Id())
	executor.maxParallel = 5
	return executor
}

func (self *batchExecutor) execute() error {
	levels := self.p.Levels()
	for i, level := range levels {
		if level == nil {
			continue
		}
		self.log.Info(fmt.Sprintf("insert queue level[%d][%t] insert.", i, level.Snapshot()))
		err := self.insertLevel(level)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *batchExecutor) insertLevel(l Level) error {
	if l.Snapshot() {
		return self.insertSnapshotLevel(l)
	} else {
		return self.insertAccountLevel(l)
	}
}
func (self *batchExecutor) insertSnapshotLevel(l Level) error {
	t1 := time.Now()
	num := 0
	defer func() {
		sub := time.Now().Sub(t1)
		levelInfo := fmt.Sprintf("\tlevel[%d][%d][%s][%d]->%dS", l.Index(), (int64(num)*time.Second.Nanoseconds())/sub.Nanoseconds(), sub, num, num)
		fmt.Println(levelInfo)
	}()
	version := self.p.Version()
	for _, b := range l.Buckets() {
		num = num + len(b.Items())
		return self.snapshotFn(self.p, b, version)
	}
	return nil
}

func (self *batchExecutor) insertAccountLevel(l Level) error {
	version := self.p.Version()
	bs := l.Buckets()
	lenBs := len(bs)
	if lenBs == 0 {
		return nil
	}

	N := helper.MinInt(lenBs, self.maxParallel)
	bucketCh := make(chan Bucket, lenBs)

	var wg sync.WaitGroup
	wg.Add(N)

	var num int32
	t1 := time.Now()
	var globalErr error
	for i := 0; i < N; i++ {
		common.Go(func() {
			defer wg.Done()
			for b := range bucketCh {
				if globalErr != nil {
					return
				}
				err := self.accountFn(self.p, b, version)
				atomic.AddInt32(&num, int32(len(b.Items())))
				if err != nil {
					globalErr = err
					fmt.Printf("error[%s] for insert account block.\n", err)
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
	levelInfo = fmt.Sprintf("\tlevel[%d][%d][%s][%d]->%s, %s", l.Index(), (int64(num)*time.Second.Nanoseconds())/sub.Nanoseconds(), sub, num, levelInfo, globalErr)
	fmt.Println(levelInfo)

	if globalErr != nil {
		return globalErr
	}
	return nil
}
