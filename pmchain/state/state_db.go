package chain_state

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/pmchain/dbutils"
	"math/big"
)

type StateDB struct {
	mvDB *multiVersionDB
}

func NewStateDB(chainDir string) (*StateDB, error) {
	mvDB, err := newMultiVersionDB(chainDir)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		mvDB: mvDB,
	}, nil
}

func (sDB *StateDB) Destroy() error {
	if err := sDB.mvDB.Destroy(); err != nil {
		return err
	}

	sDB.mvDB = nil
	return nil
}

func (sDB *StateDB) GetBalance(addr *types.Address, tokenTypeId *types.TokenTypeId) (*big.Int, error) {
	accountId := uint64(1)

	value, err := sDB.mvDB.GetValue(chain_dbutils.CreateBalanceKey(accountId, tokenTypeId))
	if err != nil {
		return nil, err
	}
	balance := big.NewInt(0).SetBytes(value)
	return balance, nil
}

func (sDB *StateDB) GetCode(addr *types.Address) ([]byte, error) {
	accountId := uint64(1)

	return sDB.mvDB.GetValue(chain_dbutils.CreateCodeKey(accountId))
}

func (sDB *StateDB) GetValue(addr *types.Address, key []byte) ([]byte, error) {
	accountId := uint64(1)
	return sDB.mvDB.GetValue(chain_dbutils.CreateStorageKeyKey(accountId, key))
}

func (sDB *StateDB) NewStorageIterator(addr *types.Address, prefix []byte) interfaces.StorageIterator {
	return nil
}
