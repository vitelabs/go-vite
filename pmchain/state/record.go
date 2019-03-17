package chain_state

import "github.com/vitelabs/go-vite/common/types"

func (sDB *StateDB) recordSnapshot(snapshotBlockHash *types.Hash, keyList [][]byte) error {
	return nil
}

func (sDB *StateDB) getKeyListBySbHashList(snapshotBlockHash []*types.Hash) ([][]byte, error) {
	return nil, nil
}
