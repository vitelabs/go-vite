package pool

import (
	"fmt"

	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
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
	defer pl.rollbackVersion.Inc()
	defer pl.version.Inc()
	pl.log.Warn("[lock]snapshot chain start fork.", "longest", longest.ID(), "current", current.ID())

	err = pl.snapshotFork(longest, longestH)
	if err != nil {
		pl.log.Error(fmt.Sprintf("fork snapshot fail. targetId:%s, targetHeight:%d, switch to current[%s].", longest.ID(), longestH, current.ID()), "err", err)
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

	_, forked, err := pl.pendingSc.chainpool.tree.FindForkPointFromMain(longest)
	if err != nil {
		pl.log.Error("get snapshot forkPoint err.", "err", err)
		return nil, err
	}
	if forked == nil {
		pl.log.Error("forked point is empty.")
		return nil, errors.New("key point is nil")
	}
	keyPoint := forked.(*snapshotPoolBlock)
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
	rollbackV  uint64
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
	return errors.Errorf("check Irreversible Principle Fail, %s, keyPoint:%d", info.String(), keyPoint.Height())
}

func (pl *pool) updateIrreversibleBlock() error {
	nodeCnt := pl.cs.SBPReader().GetNodeCount()
	if nodeCnt < 3 {
		return nil
	}
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
		pl.log.Info(fmt.Sprintf("first update irreversible, %s", info.String()))
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
	pl.log.Info(fmt.Sprintf("update irreversible, %s", result.String()))
	return nil
}

func (pl *pool) getLatestIrreversibleBlock(lastProofPoint *ledger.SnapshotBlock) (*irreversibleInfo, error) {
	nodeCnt := pl.cs.SBPReader().GetNodeCount()
	ti := pl.cs.SBPReader().GetPeriodTimeIndex()

	lastIdx := uint64(0)
	if lastProofPoint != nil {
		block, err := pl.bc.GetSnapshotHeaderByHeight(lastProofPoint.Height)
		if err != nil {
			return nil, err
		}
		if block != nil && block.Hash == lastProofPoint.Hash {
			lastIdx = ti.Time2Index(*block.Timestamp)
		}
	}
	return pl.getLatestIrreversible(lastIdx, nodeCnt, ti)
}

func (pl *pool) getLatestIrreversible(lastIdx uint64, nodeCnt int, ti core.TimeIndex) (*irreversibleInfo, error) {
	if nodeCnt < 3 {
		return nil, nil
	}
	irreversibleCnt := uint64(nodeCnt/3*2 + 1)

	head := pl.bc.GetLatestSnapshotBlock()
	latest := head
	for {
		if latest.Height <= irreversibleCnt {
			return &irreversibleInfo{point: nil, proofPoint: head, rollbackV: pl.rollbackVersion.Val()}, nil
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
		if point == nil {
			return nil, errors.New("block not exist.")
		}

		if pl.checkIrreversible(point, latest, irreversibleCnt) {
			return &irreversibleInfo{point: point, proofPoint: head, rollbackV: pl.rollbackVersion.Val()}, nil
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
		if block == nil {
			return false
		}
		addrs[block.Producer()] = struct{}{}
	}
	return uint64(len(addrs)) >= irreversibleCnt
}

func (pl *pool) GetIrreversibleBlock() *ledger.SnapshotBlock {
	pl.updateIrreversibleBlock()
	info := pl.pendingSc.irreversible
	if info != nil {
		return info.point
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
