package chain_index

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/utils"
	"path"
)

type IndexDB struct {
	chain Chain

	store *chain_db.Store

	latestAccountId uint64
	latestOnRoadId  uint64
}

func NewIndexDB(chain Chain, chainDir string) (*IndexDB, error) {
	var err error

	store, err := chain_db.NewStore(path.Join(chainDir, "index"), 0)
	if err != nil {
		return nil, err
	}

	iDB := &IndexDB{
		chain: chain,
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

	// clean store
	if err := iDB.store.Clean(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Clean failed, error is %s", err.Error()))
	}
	return nil
}

func (iDB *IndexDB) QueryLatestLocation() (*chain_file_manager.Location, error) {
	value, err := iDB.store.Get(chain_utils.CreateIndexDbLatestLocationKey())
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	if len(value) <= 0 {
		return nil, nil
	}
	return chain_utils.DeserializeLocation(value), nil
}

func (iDB *IndexDB) Destroy() error {
	if err := iDB.store.Close(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Close failed, error is %s", err.Error()))
	}
	iDB.store = nil

	return nil
}
