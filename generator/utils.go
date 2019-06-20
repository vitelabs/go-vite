package generator

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

// EnvPrepareForGenerator carries the info about the latest state of the world.
type EnvPrepareForGenerator struct {
	LatestSnapshotHash   *types.Hash
	LatestSnapshotHeight uint64
	LatestAccountHash    *types.Hash
	LatestAccountHeight  uint64
}

type stateChain interface {
	GetLatestSnapshotBlock() *ledger.SnapshotBlock
	GetLatestAccountBlock(addr types.Address) (*ledger.AccountBlock, error)
}

// GetAddressStateForGenerator returns the latest state of the world including the account's and snapshot's.
func GetAddressStateForGenerator(chain stateChain, addr *types.Address) (*EnvPrepareForGenerator, error) {
	latestSnapshot := chain.GetLatestSnapshotBlock()
	if latestSnapshot == nil {
		return nil, ErrGetLatestSnapshotBlock
	}
	var prevAccHash types.Hash
	var prevAccHeight uint64
	prevAccountBlock, err := chain.GetLatestAccountBlock(*addr)
	if err != nil {
		return nil, ErrGetLatestAccountBlock
	}
	if prevAccountBlock != nil {
		prevAccHash = prevAccountBlock.Hash
		prevAccHeight = prevAccountBlock.Height
	}
	return &EnvPrepareForGenerator{
		LatestSnapshotHash:   &latestSnapshot.Hash,
		LatestSnapshotHeight: latestSnapshot.Height,
		LatestAccountHash:    &prevAccHash,
		LatestAccountHeight:  prevAccHeight,
	}, nil
}

// VMGlobalStatus provides data about random seed.
type VMGlobalStatus struct {
	c           chain
	sb          *ledger.SnapshotBlock
	fromHash    types.Hash
	seed        uint64
	setSeed     bool
	randSource  helper.Source64
	setRandSeed bool
}

// NewVMGlobalStatus needs method to get the seed from the snapshot block.
func NewVMGlobalStatus(c chain, sb *ledger.SnapshotBlock, fromHash types.Hash) *VMGlobalStatus {
	return &VMGlobalStatus{c: c, sb: sb, fromHash: fromHash, setSeed: false}
}

// Seed returns the random seed.
func (g *VMGlobalStatus) Seed() (uint64, error) {
	if g.setSeed {
		return g.seed, nil
	}
	s, err := g.c.GetSeed(g.sb, g.fromHash)
	if err == nil {
		g.seed = s
		g.setSeed = true
	}
	return s, err
}

// Random returns a new random number
func (g *VMGlobalStatus) Random() (uint64, error) {
	if g.setRandSeed {
		return g.randSource.Uint64(), nil
	}
	s, err := g.c.GetSeed(g.sb, g.fromHash)
	if err != nil {
		return 0, err
	}
	g.randSource = helper.NewSource64(int64(s))
	g.setRandSeed = true
	return g.randSource.Uint64(), nil

}

// SnapshotBlock returns the SnapshotBlock to which the seed referred.
func (g *VMGlobalStatus) SnapshotBlock() *ledger.SnapshotBlock {
	return g.sb
}
