package chain_index

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/allegro/bigcache"
	lru "github.com/hashicorp/golang-lru"

	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	chain_db "github.com/vitelabs/go-vite/v2/ledger/chain/db"
	"github.com/vitelabs/go-vite/v2/log15"
)

type IndexDB struct {
	store *chain_db.Store

	latestAccountId uint64

	cache *bigcache.BigCache

	sendCreateBlockHashCache *lru.Cache

	accountCache *lru.Cache

	log log15.Logger
}

func NewIndexDB(chainDir string) (*IndexDB, error) {

	store, err := chain_db.NewStore(path.Join(chainDir, "index"), "indexDb")
	if err != nil {
		return nil, err
	}

	iDB := &IndexDB{
		store: store,
		log:   log15.New("module", "indexDB"),
	}

	store.RegisterAfterRecover(iDB.InitAccountId)

	iDB.InitAccountId()

	if err := iDB.newCache(); err != nil {
		return nil, err
	}

	return iDB, nil
}

func (iDB *IndexDB) Init(c Chain) error {
	return iDB.initCache(c)
}

func (iDB *IndexDB) InitAccountId() {
	var err error
	iDB.latestAccountId, err = iDB.queryLatestAccountId()
	if err != nil {
		panic(err)
	}

}

func (iDB *IndexDB) CleanAllData() error {
	// clean latestAccountId
	iDB.latestAccountId = 0

	// clean store
	if err := iDB.store.Clean(); err != nil {
		return fmt.Errorf("iDB.store.Clean failed, error is %s", err.Error())
	}
	return nil
}

func (iDB *IndexDB) Store() *chain_db.Store {
	return iDB.store
}

func (iDB *IndexDB) Close() error {
	if err := iDB.cache.Close(); err != nil {
		return fmt.Errorf("iDB.cache.Close failed, error is %s", err.Error())
	}
	iDB.cache = nil

	if err := iDB.store.Close(); err != nil {
		return fmt.Errorf("iDB.store.Close failed, error is %s", err.Error())
	}
	iDB.store = nil

	return nil
}

func (iDB *IndexDB) GetStatus() []interfaces.DBStatus {
	iDBCacheStatus, err := json.Marshal(iDB.cache.Stats())
	if err != nil {
		iDBCacheStatus = []byte("Error: " + err.Error())
	}

	statusList := iDB.store.GetStatus()

	return []interfaces.DBStatus{{
		Name:   "indexDB.cache",
		Count:  uint64(iDB.cache.Len()),
		Size:   uint64(iDB.cache.Capacity()),
		Status: string(iDBCacheStatus),
	}, {
		Name:   "indexDB.sendCreateBlockHashCache",
		Count:  uint64(iDB.sendCreateBlockHashCache.Len()),
		Size:   uint64(iDB.sendCreateBlockHashCache.Len() * (types.HashSize + 8)),
		Status: "",
	}, {
		Name:   "indexDB.accountCache",
		Count:  uint64(iDB.accountCache.Len()),
		Size:   uint64(iDB.accountCache.Len() * types.AddressSize),
		Status: "",
	}, {
		Name:   "indexDB.store.mem",
		Count:  uint64(statusList[0].Count),
		Size:   uint64(statusList[0].Size),
		Status: statusList[0].Status,
	}, {
		Name:   "indexDB.store.levelDB",
		Count:  uint64(statusList[1].Count),
		Size:   uint64(statusList[1].Size),
		Status: statusList[1].Status,
	}}
}
