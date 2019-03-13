package vm_db

import "github.com/vitelabs/go-vite/common/types"

type vmDB struct {
	chain   Chain
	address *types.Address

	latestSnapshotBlockHash *types.Hash
}

func NewVMDB(chain Chain) (VMDB, error) {
	return &vmDB{
		chain: chain,
	}, nil
}
