package pool

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/pool/tree"
)

func (self *pool) checkFork() {
	longest, longestH, err := self.pendingSc.checkFork()
	if err != nil {
		self.log.Error("check fork error", "error", err)
		return
	}

	if longest == nil {
		return
	}
	current := self.pendingSc.CurrentChain()
	curTailHeight, _ := current.TailHH()
	self.log.Warn("[try]snapshot chain start fork.", "longest", longest.Id(), "current", current.Id(),
		"longestTail", longest.SprintTail(), "longestHead", longest.SprintHead(), "currentTail", current.SprintTail(), "currentHead", current.SprintHead())
	self.LockInsert()
	defer self.UnLockInsert()
	self.LockRollback()
	defer self.UnLockRollback()
	defer self.version.Inc()
	self.log.Warn("[lock]snapshot chain start fork.", "longest", longest.Id(), "current", current.Id())

	err = self.snapshotFork(longest, longestH)
	if err != nil {
		self.log.Error(fmt.Sprintf("fork snapshot fail. targetId:%s, targetHeight:%d, switch to current[%s].", longest.Id(), longestH, current.Id()))
		err = self.snapshotFork(current, curTailHeight)
		if err != nil {
			panic(err)
		}
	}
}

func (self *pool) snapshotFork(branch tree.Branch, targetHeight uint64) error {
	err := self.snapshotRollback(branch)
	if err != nil {
		self.log.Error("snapshot rollback error", "err", err)
		return err
	}

	err = self.pendingSc.CurrentModifyToChain(branch)
	if err != nil {
		self.log.Error("snapshot current modify error", "err", err)
		return err
	}

	if cur := self.pendingSc.CurrentChain(); cur.Id() != branch.Id() {
		self.log.Error("modify to chain fail.", "longestId", branch.Id(), "currentId", cur.Id())
		return err
	}
	err = self.snapshotInsert(targetHeight)
	if err != nil {
		self.log.Error("snapshot insert error", "err", err)
		return err
	}
	return nil
}

func (self *pool) snapshotRollback(longest tree.Branch) error {
	current := self.pendingSc.CurrentChain()

	self.log.Warn("snapshot chain rollback.", "longest", longest.Id(), "current", current.Id())

	k, forked, err := self.pendingSc.chainpool.tree.FindForkPointFromMain(longest)
	if err != nil {
		self.log.Error("get snapshot forkPoint err.", "err", err)
		return err
	}
	if k == nil {
		self.log.Error("keypoint is empty.", "forked", forked.Height())
		return errors.New("key point is nil.")
	}
	keyPoint := k.(*snapshotPoolBlock)
	self.log.Info("fork point", "height", keyPoint.Height(), "hash", keyPoint.Hash())

	snapshots, accounts, e := self.pendingSc.rw.delToHeight(keyPoint.block.Height)
	if e != nil {
		return e
	}

	if len(snapshots) > 0 {
		err = self.pendingSc.rollbackCurrent(snapshots)
		if err != nil {
			return err
		}
		self.pendingSc.checkCurrent()
	}

	if len(accounts) > 0 {
		err = self.ForkAccounts(accounts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *pool) snapshotInsert(targetHeight uint64) error {
	err := self.modifyCurrentAccounts(targetHeight)
	if err != nil {
		return err
	}

	return self.insertTo(targetHeight)
}
func (self *pool) modifyCurrentAccounts(targetHeight uint64) error {
	accounts, err := self.pendingSc.genMaxAccounts(targetHeight)

	if err != nil {
		return err
	}

	for k, v := range accounts {
		err := self.ForkAccountTo(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
func (self *pool) genMaxAccounts(u uint64) {
}
