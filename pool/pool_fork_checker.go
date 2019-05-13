package pool

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/pool/tree"
)

func (pl *pool) checkFork() {
	longest, longestH, err := pl.pendingSc.checkFork()
	if err != nil {
		pl.log.Error("check fork error", "error", err)
		return
	}

	if longest == nil {
		return
	}
	current := pl.pendingSc.CurrentChain()
	curTailHeight, _ := current.TailHH()
	pl.log.Warn("[try]snapshot chain start fork.", "longest", longest.ID(), "current", current.ID(),
		"longestTail", longest.SprintTail(), "longestHead", longest.SprintHead(), "currentTail", current.SprintTail(), "currentHead", current.SprintHead())
	pl.LockInsert()
	defer pl.UnLockInsert()
	pl.LockRollback()
	defer pl.UnLockRollback()
	defer pl.version.Inc()
	pl.log.Warn("[lock]snapshot chain start fork.", "longest", longest.ID(), "current", current.ID())

	err = pl.snapshotFork(longest, longestH)
	if err != nil {
		pl.log.Error(fmt.Sprintf("fork snapshot fail. targetId:%s, targetHeight:%d, switch to current[%s].", longest.ID(), longestH, current.ID()))
		err = pl.snapshotFork(current, curTailHeight)
		if err != nil {
			panic(err)
		}
	}
}

func (pl *pool) snapshotFork(branch tree.Branch, targetHeight uint64) error {
	current := pl.pendingSc.CurrentChain()
	if current.ID() == branch.ID() {
		return nil
	}
	keyPoint, err := pl.findForkKeyPoint(branch)
	if err != nil {
		pl.log.Error("snapshot find fork key point error", "err", err)
		return err
	}

	tailH, _ := current.TailHH()
	if keyPoint.block.Height > tailH {
		err = pl.pendingSc.CurrentModifyToChain(branch)
		if err != nil {
			pl.log.Error("just snapshot current modify error", "err", err)
			return err
		}
		err := pl.modifyCurrentAccounts(keyPoint.block.Height)
		if err != nil {
			return err
		}
		return nil
	}

	err = pl.snapshotRollback(branch, keyPoint)
	if err != nil {
		pl.log.Error("snapshot rollback error", "err", err)
		return err
	}

	err = pl.pendingSc.CurrentModifyToChain(branch)
	if err != nil {
		pl.log.Error("snapshot current modify error", "err", err)
		return err
	}

	if cur := pl.pendingSc.CurrentChain(); cur.ID() != branch.ID() {
		pl.log.Error("modify to chain fail", "longestId", branch.ID(), "currentId", cur.ID())
		return err
	}
	err = pl.snapshotInsert(targetHeight)
	if err != nil {
		pl.log.Error("snapshot insert error", "err", err)
		return err
	}
	return nil
}

func (pl *pool) findForkKeyPoint(longest tree.Branch) (*snapshotPoolBlock, error) {
	current := pl.pendingSc.CurrentChain()

	pl.log.Warn("snapshot find fork point.", "longest", longest.ID(), "current", current.ID())

	k, forked, err := pl.pendingSc.chainpool.tree.FindForkPointFromMain(longest)
	if err != nil {
		pl.log.Error("get snapshot forkPoint err.", "err", err)
		return nil, err
	}
	if k == nil {
		pl.log.Error("keypoint is empty.", "forked", forked.Height())
		return nil, errors.New("key point is nil")
	}
	keyPoint := k.(*snapshotPoolBlock)
	pl.log.Info("fork point", "height", keyPoint.Height(), "hash", keyPoint.Hash())
	return keyPoint, nil
}

func (pl *pool) snapshotRollback(longest tree.Branch, keyPoint *snapshotPoolBlock) error {
	pl.log.Info("rollback snapshot chain", "height", keyPoint.Height(), "hash", keyPoint.Hash())

	snapshots, accounts, e := pl.pendingSc.rw.delToHeight(keyPoint.block.Height)
	if e != nil {
		return e
	}

	if len(snapshots) > 0 {
		err := pl.pendingSc.rollbackCurrent(snapshots)
		if err != nil {
			return err
		}
		pl.pendingSc.checkCurrent()
	}

	if len(accounts) > 0 {
		err := pl.ForkAccounts(accounts)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pl *pool) snapshotInsert(targetHeight uint64) error {
	err := pl.modifyCurrentAccounts(targetHeight)
	if err != nil {
		return err
	}

	return pl.insertTo(targetHeight)
}
func (pl *pool) modifyCurrentAccounts(targetHeight uint64) error {
	accounts, err := pl.pendingSc.genMaxAccounts(targetHeight)

	if err != nil {
		return err
	}

	for k, v := range accounts {
		err := pl.ForkAccountTo(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}
