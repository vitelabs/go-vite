package generator

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_db"
)

type EnvPrepareForGenerator struct {
	LatestSnapshotHash *types.Hash
	LatestAccountHash  *types.Hash
}

func GetAddressStateForGenerator(chain chain.Chain, addr *types.Address) (*EnvPrepareForGenerator, error) {
	latestSnapshot := chain.GetLatestSnapshotBlock()
	if latestSnapshot == nil {
		return nil, ErrGetLatestSnapshotBlock
	}
	var prevAccHash types.Hash
	prevAccountBlock, err := chain.GetLatestAccountBlock(*addr)
	if err != nil {
		return nil, ErrGetLatestAccountBlock
	}
	if prevAccountBlock != nil {
		prevAccHash = prevAccountBlock.Hash
	}
	return &EnvPrepareForGenerator{
		LatestSnapshotHash: &latestSnapshot.Hash,
		LatestAccountHash:  &prevAccHash,
	}, nil
}

//todo
func RecoverVmContext(chain vm_db.Chain, block *ledger.AccountBlock, snapshotHash *types.Hash) (vmDbList vm_db.VmDb, resultErr error) {
	return nil, nil
}
