package vm_db

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (vdb *vmDb) GetPledgeBeneficialAmount(addr *types.Address) (*big.Int, error) {
	if vdb.latestSnapshotBlockHash == nil {
		return nil, errors.New("no context, vdb.latestSnapshotBlockHash is nil")
	}

	return vdb.chain.GetPledgeBeneficialAmount(*addr)
}
