package vm

import (
	"github.com/vitelabs/go-vite/v2/common/helper"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type TestGlobalStatus struct {
	seed          uint64
	snapshotBlock *ledger.SnapshotBlock
	randSource    helper.Source64
	setRandSeed   bool
}

func NewTestGlobalStatus(seed uint64, snapshotBlock *ledger.SnapshotBlock) *TestGlobalStatus {
	return &TestGlobalStatus{seed: seed, snapshotBlock: snapshotBlock}
}

func (g *TestGlobalStatus) Seed() (uint64, error) {
	return g.seed, nil
}
func (g *TestGlobalStatus) Random() (uint64, error) {
	if g.setRandSeed {
		return g.randSource.Uint64(), nil
	}
	g.randSource = helper.NewSource64(int64(g.seed))
	g.setRandSeed = true
	return g.randSource.Uint64(), nil
}
func (g *TestGlobalStatus) SnapshotBlock() *ledger.SnapshotBlock {
	return g.snapshotBlock
}
