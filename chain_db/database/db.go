package database

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/vitelabs/go-vite/log15"
)

var dbLog = log15.New("module", "chain_db/database")

func NewLevelDb(dbDir string) *leveldb.DB {
	comparer := NewDbComparer("vite.cmp.v1")
	options := &opt.Options{
		Comparer: comparer,
	}
	db, err := leveldb.OpenFile(dbDir, options)
	if err != nil {
		dbLog.Error(err.Error())
		return nil
	}
	return db
}
