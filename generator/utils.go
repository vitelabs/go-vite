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
