package vitedb

import (
	"log"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Unconfirmed struct {
	db *DataBase
}

var _unconfirmed *Unconfirmed

func GetUnconfirmed () *Unconfirmed {
	db, err := GetLDBDataBase(DB_BLOCK)
	if err != nil {
		log.Fatal(err)
	}

	if _unconfirmed == nil {
		_unconfirmed = &Unconfirmed{
			db: db,
		}
	}

	return _unconfirmed
}

func (ucf *Unconfirmed) GetUnconfirmedAccount (accountAddress *types.Address) ([]*ledger.UnconfirmedAccount, error) {
	//key, err := createKey(DBKP_UNCONFIRMED, accountAddress)
	//if err != nil {
	//	return nil, err
	//}
	//data, dbErr := ucf.db.Leveldb.Get(key, nil)
	//if dbErr != nil {
	//	return nil, dbErr
	//}
	return nil, nil
}