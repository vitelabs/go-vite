package database

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

//, &opt.Options{

//BlockSize:           2 * opt.KiB,
//}
func NewLevelDb(dbDir string) (*leveldb.DB, error) {
	db, err := leveldb.OpenFile(dbDir, &opt.Options{
		WriteBuffer:        64 * opt.MiB,
		BlockCacheCapacity: 32 * opt.MiB,
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
