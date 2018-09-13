package access

import (
	"encoding/binary"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
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

func (sc *SnapshotChain) WriteSnapshotHash(batch *leveldb.Batch, hash *types.Hash, height uint64) {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCKHASH, hash.Bytes())
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	batch.Put(key, heightBytes)
}

func (sc *SnapshotChain) WriteSnapshotContent(batch *leveldb.Batch, height uint64, snapshotContent ledger.SnapshotContent) error {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTCONTENT, height)
	data, sErr := snapshotContent.DbSerialize()
	if sErr != nil {
		return sErr
	}
	batch.Put(key, data)
	return nil
}

func (sc *SnapshotChain) WriteSnapshotBlock(batch *leveldb.Batch, snapshotBlock *ledger.SnapshotBlock) error {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK, snapshotBlock.Height)
	data, sErr := snapshotBlock.DbSerialize()
	if sErr != nil {
		return sErr
	}
	batch.Put(key, data)
	return nil
}

func (sc *SnapshotChain) GetLatestBlock() (*ledger.SnapshotBlock, error) {
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK)

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
	key, _ := database.EncodeKey(database.DBKP_SNAPSHOTBLOCK)

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
