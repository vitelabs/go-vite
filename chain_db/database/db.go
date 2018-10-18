package database

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func NewLevelDb(dbDir string) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dbDir, &opt.Options{
		CompactionTableSize: 4,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
