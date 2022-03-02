package vm_db

import (
	"errors"
	"math/big"

	"github.com/vitelabs/go-vite/v2/common/types"
)

func (vdb *vmDb) GetStakeBeneficialAmount(addr *types.Address) (*big.Int, error) {
	if vdb.latestSnapshotBlockHash == nil {
		return nil, errors.New("no context, vdb.latestSnapshotBlockHash is nil")
	}

	return vdb.chain.GetStakeBeneficialAmount(*addr)
}
