package vm_db

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type vmDB struct {
	unsaved *Unsaved
	chain   Chain

	address *types.Address

	latestSnapshotBlockHash *types.Hash
	latestSnapshotBlock     *ledger.SnapshotBlock // for cache

	prevAccountBlockHash *types.Hash
	prevAccountBlock     *ledger.AccountBlock
}

func NewVMDB(chain Chain, address *types.Address, latestSnapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash) (VMDB, error) {
	if address == nil {
		return nil, errors.New("address is nil")
	} else if latestSnapshotBlockHash == nil {
		return nil, errors.New("latestSnapshotBlockHash is nil")
	} else if prevAccountBlockHash == nil {
		return nil, errors.New("prevAccountBlockHash is nil")
	}

	return &vmDB{
		unsaved: NewUnsaved(),
		chain:   chain,
		address: address,

		latestSnapshotBlockHash: latestSnapshotBlockHash,
		prevAccountBlockHash:    prevAccountBlockHash,
	}, nil
}

func NewNoContextVMDB(chain Chain) VMDB {
	return &vmDB{
		chain: chain,
	}
}
