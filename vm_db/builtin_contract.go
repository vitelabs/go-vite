package vm_db

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (db *vmDb) GetPledgeAmount(addr *types.Address) (*big.Int, error) {
	if db.latestSnapshotBlockHash == nil {
		return nil, errors.New("no context, db.latestSnapshotBlockHash is nil")
	}

	return db.chain.GetPledgeAmount(addr)
}
