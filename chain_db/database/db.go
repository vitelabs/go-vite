package database

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/vitelabs/go-vite/log15"
)

var dbLog = log15.New("module", "chain_db/database")

func NewLevelDb(filename string) *leveldb.DB {
	comparer := new(dbComparer)
	options := &opt.Options{
		Comparer: comparer,
	}
	db, err := leveldb.OpenFile(filename, options)
	if err != nil {
		dbLog.Error(err.Error())
		return nil
	}
	return db
}
