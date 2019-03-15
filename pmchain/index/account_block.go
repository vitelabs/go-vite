package chain_index

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/pmchain/block"
)

func (iDB *IndexDB) GetAccountBlockLocation(addr *types.Address, height uint64) (*chain_block.Location, error) {
	return nil, nil
}

//func (iDB *IndexDB) GetAccountBlockLocationByHash(blockHash *types.Hash) (string, error) {
//	return "", nil
//}
