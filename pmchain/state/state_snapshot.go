package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"math/big"
)

type stateSnapshot struct {
	stateDB *StateDB
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

func (sDB *StateDB) NewStateSnapshot(blockHash *types.Hash) (interfaces.StateSnapshot, error) {
	return &stateSnapshot{
		stateDB: sDB,
	}, nil
}
