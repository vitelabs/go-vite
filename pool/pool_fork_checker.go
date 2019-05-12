package pool

import (
	"fmt"

	"github.com/vitelabs/go-vite/common/types"

	"github.com/vitelabs/go-vite/ledger"

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
	err = pl.updateIrreversibleBlock()
	if err != nil {
		return err
	}

	err = pl.checkIrreversiblePrinciple(keyPoint)
	if err != nil {
		return err
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

type irreversibleInfo struct {
	point      *ledger.SnapshotBlock
	proofPoint *ledger.SnapshotBlock
}

func (irreversible irreversibleInfo) String() string {
	result := ""
	if irreversible.point != nil {
		result += fmt.Sprintf("point[%d-%s-%s]", irreversible.point.Height, irreversible.point.Hash, irreversible.point.Timestamp)
	}

	if irreversible.proofPoint != nil {
		result += fmt.Sprintf("proofPoint[%d-%s-%s]", irreversible.proofPoint.Height, irreversible.proofPoint.Hash, irreversible.proofPoint.Timestamp)
	}
	return result
}

func (pl *pool) checkIrreversiblePrinciple(keyPoint *snapshotPoolBlock) error {
	info := pl.pendingSc.irreversible
	if info == nil {
		return nil
	}
	if info.proofPoint == nil {
		return nil
	}
	block, err := pl.bc.GetSnapshotHeaderByHeight(info.proofPoint.Height)
	if err != nil {
		return err
	}
	if block.Hash != info.proofPoint.Hash {
		return errors.New("proof hash fail, " + info.String())
	}
	if info.point == nil {
		return nil
	}

	if info.point.Height < keyPoint.Height() {
		return nil
	}
	return errors.Errorf("check Irreversible Principle Fail, %s, keyPoint:%s"+info.String(), keyPoint.Height())
}

func (pl *pool) updateIrreversibleBlock() error {
	last := pl.pendingSc.irreversible
	if last == nil {
		info, err := pl.getLatestIrreversibleBlock(nil)
		if err != nil {
			pl.log.Error("first get latest irreversible fail", "err", err)
			return err
		}
		if info == nil {
			pl.log.Error("first get latest irreversible fail, result is nil")
			return errors.New("can't get irreversible info")
		}
		pl.pendingSc.irreversible = info
		pl.log.Info("first update irreversible, %s", info.String())
		return nil
	}

	result, err := pl.getLatestIrreversibleBlock(last.proofPoint)
	if err != nil {
		pl.log.Error("get latest irreversible fail", "err", err)
		return err
	}

	if result == nil {
		return nil
	}

	pl.pendingSc.irreversible = result
	pl.log.Info("update irreversible, %s", result.String())
	return nil
}

func (pl *pool) getLatestIrreversibleBlock(lastProofPoint *ledger.SnapshotBlock) (*irreversibleInfo, error) {
	nodeCnt := pl.cs.SBPReader().GetNodeCount()
	irreversibleCnt := uint64(nodeCnt/3*2 + 1)
	ti := pl.cs.SBPReader().GetPeriodTimeIndex()

	lastIdx := uint64(0)
	if lastProofPoint != nil {
		block, err := pl.bc.GetSnapshotHeaderByHeight(lastProofPoint.Height)
		if err != nil {
			return nil, err
		}
		if block.Hash == lastProofPoint.Hash {
			lastIdx = ti.Time2Index(*block.Timestamp)
		}
	}

	pl.cs.SBPReader().GetPeriodTimeIndex()
	latest := pl.bc.GetLatestSnapshotBlock()
	for {
		if latest.Height <= irreversibleCnt {
			return &irreversibleInfo{point: nil, proofPoint: latest}, nil
		}

		endTime := latest.Timestamp

		index := ti.Time2Index(*endTime)

		if index < lastIdx {
			return nil, nil
		}

		stime, _ := ti.Index2Time(index)
		block, err := pl.bc.GetSnapshotHeaderBeforeTime(&stime)
		if err != nil {
			return nil, err
		}
		point, err := pl.bc.GetSnapshotBlockByHeight(block.Height + 1)
		if err != nil {
			return nil, err
		}

		if pl.checkIrreversible(point, latest, irreversibleCnt) {
			return &irreversibleInfo{point: point, proofPoint: latest}, nil
		}
		latest = block
	}
}

func (pl *pool) checkIrreversible(point *ledger.SnapshotBlock, proofPoint *ledger.SnapshotBlock, irreversibleCnt uint64) bool {
	if 1+proofPoint.Height-point.Height < irreversibleCnt {
		return false
	}
	addrs := make(map[types.Address]struct{})
	addrs[proofPoint.Producer()] = struct{}{}
	addrs[point.Producer()] = struct{}{}
	for i := proofPoint.Height - 1; i > point.Height; i-- {
		block, err := pl.bc.GetSnapshotHeaderByHeight(i)
		if err != nil {
			return false
		}
		addrs[block.Producer()] = struct{}{}
	}
	return uint64(len(addrs)) >= irreversibleCnt
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
