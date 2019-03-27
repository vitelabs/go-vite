package chain_index

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain/file_manager"
	"github.com/vitelabs/go-vite/chain/pending"
	"github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/interfaces"
)

type IndexDB struct {
	chain Chain

	store Store
	memDb *chain_pending.MemDB

	latestAccountId uint64
	latestOnRoadId  uint64
}

func NewIndexDB(chain Chain, chainDir string) (*IndexDB, error) {
	var err error

	store, err := NewStore(chainDir)
	if err != nil {
		return nil, err
	}

	iDB := &IndexDB{
		chain: chain,
		store: store,
		memDb: chain_pending.NewMemDB(),
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
	// clean memory
	iDB.memDb.Clean()

	// clean latestAccountId
	iDB.latestAccountId = 0

	// clean store
	if err := iDB.store.Clean(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Clean failed, error is %s", err.Error()))
	}
	return nil
}

func (iDB *IndexDB) NewIterator(slice *util.Range) interfaces.StorageIterator {
	return dbutils.NewMergedIterator([]interfaces.StorageIterator{
		iDB.memDb.NewIterator(slice),
		iDB.store.NewIterator(slice),
	}, iDB.memDb.IsDelete)
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
	iDB.memDb = nil
	if err := iDB.store.Close(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Close failed, error is %s", err.Error()))
	}
	iDB.store = nil
	return nil
}

func (iDB *IndexDB) getValue(key []byte) ([]byte, error) {
	value, ok := iDB.memDb.Get(key)
	if !ok {
		var err error
		value, err = iDB.store.Get(key)
		if err != nil {
			if err == leveldb.ErrNotFound {
				return nil, nil
			}
			return nil, err
		}
	}

	return value, nil
}

func (iDB *IndexDB) hasValue(key []byte) (bool, error) {
	if ok, deleted := iDB.memDb.Has(key); ok {
		return ok, nil

	} else if deleted {
		return false, nil

	}

	return iDB.store.Has(key)
}
func (iDB *IndexDB) hasValueByPrefix(prefix []byte) (bool, error) {
	if ok := iDB.memDb.HasByPrefix(prefix); ok {
		return ok, nil
	}

	iter := iDB.store.NewIterator(util.BytesPrefix(prefix))
	defer iter.Release()

	result := false
	for iter.Next() {
		key := iter.Key()
		if ok := iDB.memDb.IsDelete(key); !ok {
			result = true
			break
		}
	}

	if err := iter.Error(); err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return result, nil

}
