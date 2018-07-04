package vitedb

import (
	"log"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"github.com/vitelabs/go-vite/common"
)

type Account struct {
	db *DataBase
}

var _account *Account

func (account Account) GetInstance () *Account {
	if _account == nil {
		db, err:= GetLDBDataBase(DB_BLOCK)
		if err != nil {
			log.Fatal(err)
		}

		_account = &Account{
			db: db,
		}
	}

	return _account


}

func (account *Account) GetAccountMetaByAddress (hexAddress *common.Address) (*ledger.AccountMeta, error) {
	keyAccountMeta, ckErr := createKey(DBKP_ACCOUNTMETA, hexAddress.String())
	if ckErr != nil {
		return nil, ckErr
	}
	data, dgErr := account.db.Get(keyAccountMeta)
	if dgErr != nil {
		log.Fatalln("GetAccountMetaByAddress func db.Get() error:", dgErr)
		return nil, dgErr
	}
	accountMeter := &ledger.AccountMeta{}
	dsErr := accountMeter.DbDeserialize(data)
	if dsErr != nil {
		log.Fatal(dsErr)
		return nil, dsErr
	}
	return accountMeter, nil
}

func (account *Account) GetAddressById (accountId *big.Int) (*common.Address, error) {
	keyAccountAddress, ckErr := createKey(DBKP_ACCOUNTID_INDEX, accountId)
	if ckErr != nil {
		return nil, ckErr
	}
	data, dgErr := account.db.Get(keyAccountAddress)
	if dgErr != nil {
		log.Fatalln("GetAddressById func db.Get() error:", dgErr)
		return nil, dgErr
	}
	b2Address, b2Err := common.BytesToAddress(data)
	if b2Err != nil {
		return nil, b2Err
	}
	return &b2Address, nil
}