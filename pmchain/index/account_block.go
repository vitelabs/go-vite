package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (iDB *IndexDB) IsAccountBlockExisted(hash *types.Hash) (bool, error) {
	key := createAccountBlockHashKey(hash)
	if ok := iDB.memDb.Has(key); ok {
		return ok, nil
	}

	return iDB.store.Has(key)
}

func (iDB *IndexDB) GetAccountBlockLocation(addr *types.Address, height uint64) (*chain_block.Location, error) {
	return nil, nil
}

func (iDB *IndexDB) GetAccountHeightByHash(blockHash *types.Hash) (uint64, uint64, error) {
	return 0, 0, nil
}

//func (iDB *IndexDB) GetAccountBlockLocationByHash(blockHash *types.Hash) (string, error) {
//	return "", nil
//}
