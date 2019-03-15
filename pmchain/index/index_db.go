package chain_index

import (
	"fmt"
	"github.com/pkg/errors"
)

type IndexDB struct {
	latestAccountId uint64

	store Store
	memDb MemDB
}

func NewIndexDB(chainDir string) (*IndexDB, error) {
	store, err := NewStore(chainDir)
	if err != nil {
		return nil, err
	}

	iDB := &IndexDB{
		store: store,
		memDb: newMemDb(),
	}
	latestAccountId, err := iDB.queryLatestAccountId()
	if err != nil {
		return nil, err
	}

	iDB.latestAccountId = latestAccountId

	return iDB, nil
}

func (iDB *IndexDB) CleanUnconfirmedIndex() {
	iDB.memDb.Clean()
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
func (iDB *IndexDB) Destroy() error {
	iDB.memDb = nil
	if err := iDB.store.Close(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Close failed, error is %s", err.Error()))
	}
	iDB.store = nil
	return nil
}
