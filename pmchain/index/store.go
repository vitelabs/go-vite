package chain_index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"path"
)

type store struct {
	db *leveldb.DB
}

func NewStore(chainDir string) (Store, error) {
	dbDir := path.Join(chainDir, "indexes")

	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}

	return &store{
		db: db,
	}, nil
}
func (s *store) NewBatch() Batch {
	return new(leveldb.Batch)
}
func (s *store) Write(batch Batch) error {
	return s.db.Write(batch.(*leveldb.Batch), nil)
}
