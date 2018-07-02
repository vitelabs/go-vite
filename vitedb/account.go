package vitedb

import (
	"log"
	"github.com/vitelabs/go-vite/ledger"
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

func (a *Account) GetAccountMetaByAddress (accountAddress []byte) (*ledger.AccountMeta, error) {
	return nil, nil
}