package database

import (
	"github.com/syndtr/goleveldb/leveldb"
)

//, &opt.Options{

//BlockSize:           2 * opt.KiB,
//}
func NewLevelDb(dbDir string) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}
