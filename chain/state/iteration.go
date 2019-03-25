package chain_state

import (
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
)

// TODO
func (sDB *StateDB) NewStorageIterator(addr *types.Address, prefix []byte) (interfaces.StorageIterator, error) {
	return sDB.db.NewIterator(util.BytesPrefix(chain_utils.CreateStorageValueKey(addr, prefix)), nil), nil
}
