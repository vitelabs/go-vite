package chain_index

import (
	"fmt"
	"github.com/allegro/bigcache"
	"github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/chain/db"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/crypto"
	"github.com/vitelabs/go-vite/log15"
	"path"
)

type IndexDB struct {
	store *chain_db.Store

	latestAccountId uint64
	cache           *bigcache.BigCache
	accountCache    *lru.Cache

	log log15.Logger

	onRoadData map[types.Address]map[types.Hash]struct{} // FIXME, template test

	chain Chain
}

func NewIndexDB(chainDir string, chain Chain) (*IndexDB, error) {
	id, _ := types.BytesToHash(crypto.Hash256([]byte("indexDb")))

	var err error
	store, err := chain_db.NewStore(path.Join(chainDir, "index"), id)
	if err != nil {
		return nil, err
	}

	iDB := &IndexDB{
		store: store,
		log:   log15.New("module", "indexDB"),
		chain: chain,
	}

	iDB.latestAccountId, err = iDB.queryLatestAccountId()
	if err != nil {
		return nil, err
	}

	if err = iDB.newCache(); err != nil {
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

func (iDB *IndexDB) Store() *chain_db.Store {
	return iDB.store
}

func (iDB *IndexDB) Close() error {
	if err := iDB.cache.Close(); err != nil {
		return errors.New(fmt.Sprintf("iDB.cache.Close failed, error is %s", err.Error()))
	}
	iDB.cache = nil

	if err := iDB.store.Close(); err != nil {
		return errors.New(fmt.Sprintf("iDB.store.Close failed, error is %s", err.Error()))
	}
	iDB.store = nil

	return nil
}
