package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
)

type OnRoad struct {
	db *leveldb.DB
}

func NewOnRoad(db *leveldb.DB) *OnRoad {
	return &OnRoad{
		db: db,
	}
}

func (or *OnRoad) GetMeta(addr *types.Address, hash *types.Hash) ([]byte, error) {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}
	value, err := or.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}
	return value, nil
}
