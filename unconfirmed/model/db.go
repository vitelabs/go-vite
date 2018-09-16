// Data structures stored: key[DBKP_UNCONFIRMEDMETA.address.hash]=value[markType]
// markType: []byte("1"),represents true;[]byte("0"),represents false

package model

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
)

type UnconfirmedSet struct {
	db *leveldb.DB
}

func NewUnconfirmedSet(db *leveldb.DB) *UnconfirmedSet {
	return &UnconfirmedSet{
		db: db,
	}
}

func (ucf *UnconfirmedSet) GetCountByAddress(addr *types.Address) (count uint64, err error) {
	count = 0
	key, err := database.EncodeKey(database.DBKP_UNCONFIRMEDMETA, addr.Bytes(), "KEY_MAX")

	if err != nil {
		return 0, err
	}

	iter := ucf.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	for iter.Next() {
		count += 1
	}
	return count, nil
}

func (ucf *UnconfirmedSet) GetHashsByCount(count uint64, addr *types.Address) (hashs []*types.Hash, err error) {
	key, err := database.EncodeKey(database.DBKP_UNCONFIRMEDMETA, addr.Bytes(), "KEY_MAX")
	if err != nil {
		return nil, err
	}

	iter := ucf.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	for i := uint64(0); i < count; iter.Next() {
		key := iter.Key()
		hash, err := types.BytesToHash(key[len(key)-types.HashSize:])
		if err != nil {
			continue
		}
		hashs = append(hashs, &hash)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return hashs, nil
}

func (ucf *UnconfirmedSet) GetHashList(addr *types.Address) (hashs []*types.Hash, err error) {
	key, err := database.EncodeKey(database.DBKP_UNCONFIRMEDMETA, addr.Bytes(), "KEY_MAX")
	if err != nil {
		return nil, err
	}

	iter := ucf.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		hash, err := types.BytesToHash(key[len(key)-types.HashSize:])
		if err != nil {
			continue
		}
		hashs = append(hashs, &hash)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return hashs, nil
}

func (ucf *UnconfirmedSet) WriteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	key, err := database.EncodeKey(database.DBKP_UNCONFIRMEDMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	batch.Put(key, []byte("1"))
	return nil
}

func (ucf *UnconfirmedSet) DeleteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	key, err := database.EncodeKey(database.DBKP_UNCONFIRMEDMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	batch.Delete(key)
	return nil
}