package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
)

func (iDB *IndexDB) IsAccountBlockExisted(hash *types.Hash) (bool, error) {
	key := chain_dbutils.CreateAccountBlockHashKey(hash)
	if ok := iDB.memDb.Has(key); ok {
		return ok, nil
	}

	return iDB.store.Has(key)
}

func (iDB *IndexDB) GetAccountBlockLocation(addr *types.Address, height uint64) (*chain_block.Location, error) {
	return nil, nil
}

func (iDB *IndexDB) GetAccountBlockLocationList(hash *types.Hash, count uint64) ([]*chain_block.Location, uint64, [2]uint64, error) {
	return nil, 0, [2]uint64{}, nil
}
func (iDB *IndexDB) GetConfirmHeightByHash(blockHash *types.Hash) (uint64, error) {
	return 0, nil
}

func (iDB *IndexDB) GetAccountHeightByHash(blockHash *types.Hash) (uint64, uint64, error) {
	return 0, 0, nil
}

func (iDB *IndexDB) GetReceivedBySend(sendBlockHash *types.Hash) (uint64, uint64, error) {
	return 0, 0, nil
}

func (iDB *IndexDB) IsReceived(sendBlockHash *types.Hash) (bool, error) {
	return false, nil
}

//func (iDB *IndexDB) GetAccountBlockLocationByHash(blockHash *types.Hash) (string, error) {
//	return "", nil
//}
