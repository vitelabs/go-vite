package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/helper"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type Account struct {
	db  *leveldb.DB
	log log15.Logger
}

func NewAccount(db *leveldb.DB) *Account {
	return &Account{
		db:  db,
		log: log15.New("module", "ledger/access/account"),
	}
}

func (accountAccess *Account) GetAccountByAddress(address *types.Address) (*ledger.Account, error) {
	keyAccountMeta, _ := helper.EncodeKey(database.DBKP_ACCOUNT, address.Bytes())

	data, dgErr := accountAccess.db.Get(keyAccountMeta, nil)
	if dgErr != nil {
		accountAccess.log.Error("GetAccountMetaByAddress func db.Get()", "dgErr", dgErr)
		return nil, dgErr
	}
	account := &ledger.Account{}
	dsErr := account.DbDeSerialize(data)

	if dsErr != nil {
		accountAccess.log.Error(dsErr.Error())
		return nil, dsErr
	}

	return account, nil
}
