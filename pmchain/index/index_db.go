package chain_index

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/dbutils"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/pmchain/block"
	"github.com/vitelabs/go-vite/pmchain/pending"
	"github.com/vitelabs/go-vite/pmchain/utils"
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
	}, iDB.memDb.DeletedKeys())
}

func (iDB *IndexDB) QueryLatestLocation() (*chain_block.Location, error) {
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
	if ok := iDB.memDb.Has(key); ok {
		return ok, nil
	}

	return iDB.store.Has(key)
}
func (iDB *IndexDB) hasValueByPrefix(prefix []byte) (bool, error) {
	if ok := iDB.memDb.HasByPrefix(prefix); ok {
		return ok, nil
	}

	iter := iDB.store.NewIterator(util.BytesPrefix(prefix))
	defer iter.Release()

	iter.Next()

	if err := iter.Error(); err != nil {
		if err == leveldb.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil

}
