package chain_index

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"os"
	"path"
)

type store struct {
	dbDir string

	db *leveldb.DB
}

func NewStore(chainDir string) (Store, error) {
	dbDir := path.Join(chainDir, "indexes")

	db, err := leveldb.OpenFile(dbDir, nil)
	if err != nil {
		return nil, err
	}

	return &store{
		db:    db,
		dbDir: dbDir,
	}, nil
}
func (s *store) NewBatch() Batch {
	return new(leveldb.Batch)
}
func (s *store) Write(batch Batch) error {
	return s.db.Write(batch.(*leveldb.Batch), nil)
}

func (s *store) Get(key []byte) ([]byte, error) {
	return s.db.Get(key, nil)
}

func (s *store) Has(key []byte) (bool, error) {
	return s.db.Has(key, nil)
}

func (s *store) Close() error {
	return s.db.Close()
}

func (s *store) Clean() error {
	if closeErr := s.Close(); closeErr != nil {
		return errors.New(fmt.Sprintf("Close db failed, error is %s", closeErr))
	}

	if err := os.RemoveAll(s.dbDir); err != nil && err != os.ErrNotExist {
		return errors.New("Remove " + s.dbDir + " failed, error is " + err.Error())
	}

	s.db = nil

	return nil
}
