package database

import (
	"github.com/syndtr/goleveldb/leveldb"
)

func NewLevelDb(dbDir string) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}
