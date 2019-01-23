package database

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

func NewLevelDb(dbDir string) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dbDir, &opt.Options{
		BlockCacheCapacity: 128 * opt.MiB,
	})

	if err != nil {
		return nil, err
	}
	return db, nil
}
