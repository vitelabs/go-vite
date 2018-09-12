package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Account struct {
	db *leveldb.DB
}

func NewAccount(db *leveldb.DB) *Account {
	return &Account{
		db: db,
	}
}

func (accountAccess *Account) GetAddressById(accountId uint64) (*types.Address, error) {
	keyAccountAddress, _ := database.EncodeKey(database.DBKP_ACCOUNTID_INDEX, accountId)
	data, dgErr := accountAccess.db.Get(keyAccountAddress, nil)

	if dgErr != nil {
		return nil, dgErr
	}

	b2Address, b2Err := types.BytesToAddress(data)
	if b2Err != nil {
		return nil, b2Err
	}
	return &b2Address, nil
}

func (accountAccess *Account) GetAccountByAddress(address *types.Address) (*ledger.Account, error) {
	keyAccountMeta, _ := database.EncodeKey(database.DBKP_ACCOUNT, address.Bytes())

	data, dgErr := accountAccess.db.Get(keyAccountMeta, nil)
	if dgErr != nil {
		return nil, dgErr
	}
	account := &ledger.Account{}
	dsErr := account.DbDeserialize(data)

	if dsErr != nil {
		return nil, dsErr
	}

	return account, nil
}
