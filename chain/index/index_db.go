package chain_index

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/db"
	"path"
)

type IndexDB struct {
	store *chain_db.Store

	latestAccountId uint64
	latestOnRoadId  uint64
}

func NewIndexDB(chainDir string) (*IndexDB, error) {

	var err error

	store, err := chain_db.NewStore(path.Join(chainDir, "index"), 0)
	if err != nil {
		return nil, err
	}

	iDB := &IndexDB{
		store: store,
	}

	iDB.latestAccountId, err = iDB.queryLatestAccountId()
	if err != nil {
		return nil, err
	}

	iDB.latestOnRoadId, err = iDB.queryLatestOnRoadId()
	if err != nil {
		return nil, err
	}

	return iDB, nil
}

func (iDB *IndexDB) CleanAllData() error {
	// clean latestAccountId
	iDB.latestAccountId = 0

	// clean latestOnRoadId
	iDB.latestOnRoadId = 0

	// clean store
	if err := iDB.store.Clean(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Clean failed, error is %s", err.Error()))
	}
	return nil
}

func (iDB *IndexDB) Store() *chain_db.Store {
	return iDB.store
}

func (iDB *IndexDB) Close() error {
	if err := iDB.store.Close(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Close failed, error is %s", err.Error()))
	}
	iDB.store = nil

	return nil
}
