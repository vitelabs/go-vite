package pool

import (
	"fmt"
	"time"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/pool/batch"
)

func (self *pool) insertTo(height uint64) error {
	for {
		err := self.checkTarget(height)
		if err == nil {
			return nil
		}
		t1 := time.Now()
		q := self.makeQueueOnly()
		size := q.Size()
		if size == 0 {
			return self.checkTarget(height)
		}
		err = self.insertQueue(q)
		if err != nil {
			self.log.Error(fmt.Sprintf("insert queue err:%s\n", err))
			self.log.Error(fmt.Sprintf("all queue:%s\n", q.Info()))
			e := self.checkTarget(height)
			if e != nil {
				tailHeight, _ := self.pendingSc.CurrentChain().TailHH()
				block := self.pendingSc.getCurrentBlock(tailHeight + 1)
				if block != nil {
					self.hashBlacklist.AddAddTimeout(block.block.Hash, time.Minute*30)
				}
			}
			return e
		}
		t2 := time.Now()
		self.log.Info(fmt.Sprintf("time duration:%s, size:%d", t2.Sub(t1), size))
	}
}

func (self *pool) checkTarget(height uint64) error {
	curHeight, _ := self.pendingSc.CurrentChain().TailHH()
	if curHeight >= height {
		return nil
	}
	return errors.Errorf("target fail.[%d][%d]", height, curHeight)
}

func (self *pool) makeQueueOnly() batch.Batch {
	tailHeight, tailHash := self.pendingSc.CurrentChain().TailHH()
	snapshotOffset := &offsetInfo{offset: &ledger.HashHeight{Height: tailHeight, Hash: tailHash}}

	p := batch.NewBatch(self.snapshotExists, self.accountExists, self.version.Val(), 50)
	for {
		newOffset, _, tmpSb := self.makeSnapshotBlock(p, snapshotOffset)
		if tmpSb == nil {
			if p.Size() > 0 {
				break
			}
			return p
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
