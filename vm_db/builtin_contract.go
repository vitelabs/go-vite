package vm_db

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

func (db *vmDb) GetPledgeAmount(addr *types.Address) (*big.Int, error) {
	if db.latestSnapshotBlockHash == nil {
		return nil, errors.New("No context, db.latestSnapshotBlockHash is nil")
	}

	return db.chain.GetPledgeAmount(db.latestSnapshotBlockHash, addr)
}
