package vm_db

import (
	"errors"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type vmDb struct {
	interfaces.VmDb
	uns *Unsaved // the unsaved memory

	chain Chain

	// context
	address   *types.Address
	isGenesis bool

	latestSnapshotBlockHash *types.Hash
	latestSnapshotBlock     *ledger.SnapshotBlock // for cache

	prevAccountBlockHash *types.Hash
	prevAccountBlock     *ledger.AccountBlock // for cache

	callDepth *uint16 // for cache
}

func NewVmDb(chain Chain, address *types.Address, latestSnapshotBlockHash *types.Hash, prevAccountBlockHash *types.Hash) (*vmDb, error) {
	if address == nil {
		return nil, errors.New("address is nil")
	} else if latestSnapshotBlockHash == nil {
		return nil, errors.New("latestSnapshotBlockHash is nil")
	} else if prevAccountBlockHash == nil {
		return nil, errors.New("prevAccountBlockHash is nil")
	}

	return &vmDb{
		chain:     chain,
		address:   address,
		isGenesis: false,

		latestSnapshotBlockHash: latestSnapshotBlockHash,
		prevAccountBlockHash:    prevAccountBlockHash,
	}, nil
}

func (vdb *vmDb) unsaved() *Unsaved {
	if vdb.uns == nil {
		vdb.uns = NewUnsaved()
	}
	return vdb.uns
}

func (vdb *vmDb) CanWrite() bool {
	return vdb.isGenesis || (vdb.address != nil &&
		vdb.prevAccountBlockHash != nil &&
		vdb.latestSnapshotBlockHash != nil)
}

func NewNoContextVmDb(chain Chain) *vmDb {
	return &vmDb{
		chain: chain,
	}
}

func NewVmDbByAddr(chain Chain, address *types.Address) *vmDb {
	return &vmDb{
		chain:   chain,
		address: address,
	}
}

func NewGenesisVmDB(address *types.Address) *vmDb {
	return &vmDb{
		address:   address,
		isGenesis: true,
	}
}
