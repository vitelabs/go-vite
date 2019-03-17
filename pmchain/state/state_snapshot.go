package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"math/big"
)

func (sDB *StateDB) NewStateSnapshot(snapshotBlockHash *types.Hash, addr *types.Address) (interfaces.StateSnapshot, error) {
	return newStateSnapshot(sDB, snapshotBlockHash, addr)
}

type stateSnapshot struct {
	stateDB           *StateDB
	snapshotBlockHash *types.Hash
	addr              *types.Address
}

func newStateSnapshot(sDB *StateDB, snapshotBlockHash *types.Hash, addr *types.Address) (*stateSnapshot, error) {
	ss := &stateSnapshot{
		stateDB:           sDB,
		snapshotBlockHash: snapshotBlockHash,
		addr:              addr,
	}

	return ss, nil
}

func (ss *stateSnapshot) GetBalance(tokenId *types.TokenTypeId) (*big.Int, error) {

	return nil, nil
}

func (ss *stateSnapshot) GetCode() ([]byte, error) {
	return nil, nil
}

func (ss *stateSnapshot) GetValue([]byte) ([]byte, error) {
	return nil, nil
}

func (ss *stateSnapshot) NewStorageIterator(prefix []byte) interfaces.StorageIterator {
	return nil
}
