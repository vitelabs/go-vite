package access

import (
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/db_helper"
	"github.com/vitelabs/go-vite/ledger"
)

type SnapshotChain struct {
	db *leveldb.DB
}

func NewSnapshotChain(db *leveldb.DB) *SnapshotChain {
	return &SnapshotChain{
		db: db,
	}
}

func (sc *SnapshotChain) GetLatestBlock() (*ledger.SnapshotBlock, error) {
	key, ckErr := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, "KEY_MAX")
	if ckErr != nil {
		return nil, ckErr
	}

	iter := sc.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, errors.New("GetLatestBlock failed. Because the SnapshotChain has no block")
	}

	sb := &ledger.SnapshotBlock{}
	sdErr := sb.DbDeserialize(iter.Value())

	if sdErr != nil {
		return nil, sdErr
	}

	return sb, nil
}

func (sc *SnapshotChain) GetGenesesBlock() (*ledger.SnapshotBlock, error) {
	key, ckErr := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, "KEY_MAX")
	if ckErr != nil {
		return nil, ckErr
	}

	iter := sc.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Next() {
		return nil, errors.New("GetGenesesBlock failed. Because the SnapshotChain has no block")
	}

	sb := &ledger.SnapshotBlock{}
	sdErr := sb.DbDeserialize(iter.Value())

	if sdErr != nil {
		return nil, sdErr
	}

	return sb, nil
}
