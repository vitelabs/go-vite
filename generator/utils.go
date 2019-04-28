package generator

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type EnvPrepareForGenerator struct {
	LatestSnapshotHash  *types.Hash
	LatestAccountHash   *types.Hash
	LatestAccountHeight uint64
}

func GetAddressStateForGenerator(chain chain.Chain, addr *types.Address) (*EnvPrepareForGenerator, error) {
	latestSnapshot := chain.GetLatestSnapshotBlock()
	if latestSnapshot == nil {
		return nil, ErrGetLatestSnapshotBlock
	}
	var prevAccHash types.Hash
	var prevAccHeight uint64 = 0
	prevAccountBlock, err := chain.GetLatestAccountBlock(*addr)
	if err != nil {
		return nil, ErrGetLatestAccountBlock
	}
	if prevAccountBlock != nil {
		prevAccHash = prevAccountBlock.Hash
		prevAccHeight = prevAccountBlock.Height
	}
	return &EnvPrepareForGenerator{
		LatestSnapshotHash:  &latestSnapshot.Hash,
		LatestAccountHash:   &prevAccHash,
		LatestAccountHeight: prevAccHeight,
	}, nil
}

//todo
func RecoverVmContext(chain vm_db.Chain, block *ledger.AccountBlock, snapshotHash *types.Hash) (vmDbList vm_db.VmDb, resultErr error) {
	return nil, nil
}

type VMGlobalStatus struct {
	c        Chain
	sb       *ledger.SnapshotBlock
	fromHash types.Hash
	seed     uint64
	setSeed  bool
}

func NewVMGlobalStatus(c Chain, sb *ledger.SnapshotBlock, fromHash types.Hash) *VMGlobalStatus {
	return &VMGlobalStatus{c: c, sb: sb, fromHash: fromHash, setSeed: false}
}
func (g *VMGlobalStatus) Seed() (uint64, error) {
	if g.setSeed {
		return g.seed, nil
	} else {
		s, err := g.c.GetSeed(g.sb, g.fromHash)
		if err == nil {
			g.seed = s
			g.setSeed = true
		}
		return s, err
	}
}
func (g *VMGlobalStatus) SnapshotBlock() *ledger.SnapshotBlock {
	return g.sb
}
