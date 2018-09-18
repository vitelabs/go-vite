// Data structures stored: key[DBKP_UNCONFIRMEDMETA.address.hash]=value[markType]
// markType: []byte("1"),represents true;[]byte("0"),represents false

package vitedb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type UnconfirmedDB struct {
	db *DataBase
}

func NewUnconfirmedDB() *UnconfirmedDB {
	db, err := GetLDBDataBase(DB_LEDGER)
	if err != nil {
		log15.Root().Crit(err.Error())
	}
	return &UnconfirmedDB{db: db}
}

func (ucf *UnconfirmedDB) GetCountByAddress(addr *types.Address) (count uint64, err error) {
	count = 0

	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes(), nil)
	if err != nil {
		return 0, err
	}

	iter := ucf.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	for iter.Next() {
		count += 1
	}
	return count, nil
}

func (ucf *UnconfirmedDB) GetHashsByCount(count uint64, addr *types.Address) (hashs []*types.Hash, err error) {
	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes(), nil)
	if err != nil {
		return nil, err
	}

	iter := ucf.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
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

func (ucf *UnconfirmedDB) GetHashList(addr *types.Address) (hashs []*types.Hash, err error) {
	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes(), nil)
	if err != nil {
		return nil, err
	}

	iter := ucf.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
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

func (ucf *UnconfirmedDB) WriteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	batch.Put(key, []byte("1"))
	return nil
}

func (ucf *UnconfirmedDB) DeleteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	batch.Delete(key)
	return nil
}

//func (ucf *UnconfirmedDB) GetUnconfirmedMeta(addr *types.Address) (*ledger.UnconfirmedMeta, error) {
//	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes())
//	if err != nil {
//		return nil, err
//	}
//	data, err := ucf.db.Leveldb.Get(key, nil)
//	if err != nil {
//		return nil, err
//	}
//	var ucfm = &ledger.UnconfirmedMeta{}
//	if err := ucfm.DbDeserialize(data); err != nil {
//		return nil, err
//	}
//	return ucfm, nil
//}
//
//func (ucf *UnconfirmedDB) GetAccHashListByTkId(addr *types.Address, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
//	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, addr.Bytes(), tokenId.Bytes())
//	if err != nil {
//		return nil, err
//	}
//	data, err := ucf.db.Leveldb.Get(key, nil)
//	if err != nil {
//		return nil, err
//	}
//	hList, err := ledger.HashListDbDeserialize(data)
//	if err != nil {
//		return nil, err
//	}
//	return hList, nil
//}
//
//func (ucf *UnconfirmedDB) GetAccTotalHashList(addr *types.Address) ([]*types.Hash, error) {
//	var hashList []*types.Hash
//
//	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, addr.Bytes(), nil)
//	if err != nil {
//		return nil, err
//	}
//
//	iter := ucf.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
//	defer iter.Release()
//
//	for iter.Next() {
//		hList, err := ledger.HashListDbDeserialize(iter.Value())
//		if err != nil {
//			return nil, err
//		}
//		hashList = append(hashList, hList...)
//	}
//	if err := iter.Error(); err != nil {
//		return nil, err
//	}
//	return hashList, nil
//}
//
//func (ucf *UnconfirmedDB) WriteMeta(batch *leveldb.Batch, addr *types.Address, meta *ledger.UnconfirmedMeta) error {
//	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes())
//	if err != nil {
//		return err
//	}
//	data, err := meta.DbSerialize()
//	if err != nil {
//		return err
//	}
//	batch.Put(key, data)
//	return nil
//}
//
//func (ucf *UnconfirmedDB) WriteHashList(batch *leveldb.Batch, addr *types.Address, tokenId *types.TokenTypeId, hList []*types.Hash) error {
//	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, addr.Bytes(), tokenId.Bytes())
//	if err != nil {
//		return err
//	}
//	data, err := ledger.HashListDbSerialize(hList)
//	if err != nil {
//		return err
//	}
//	batch.Put(key, data)
//	return nil
//}
//
//func (ucf *UnconfirmedDB) DeleteMeta(batch *leveldb.Batch, addr *types.Address) error {
//	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes())
//	if err != nil {
//		return err
//	}
//	batch.Delete(key)
//	return nil
//}
//
//func (ucf *UnconfirmedDB) DeleteHashList(batch *leveldb.Batch, addr *types.Address, tokenId *types.TokenTypeId) error {
//	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, addr.Bytes(), tokenId.Bytes())
//	if err != nil {
//		return err
//	}
//	batch.Delete(key)
//	return nil
//}
//
//func (ucf *UnconfirmedDB) DeleteAllHashList(batch *leveldb.Batch, addr *types.Address) error {
//	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, addr.Bytes(), nil)
//	if err != nil {
//		return err
//	}
//
//	iter := ucf.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
//	defer iter.Release()
//
//	for iter.Next() {
//		batch.Delete(iter.Key())
//	}
//	if err := iter.Error(); err != nil {
//		return err
//	}
//	return nil
//}
