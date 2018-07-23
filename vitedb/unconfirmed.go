package vitedb

import (
	"log"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"github.com/syndtr/goleveldb/leveldb"
)

type Unconfirmed struct {
	db *DataBase
}

var _unconfirmed *Unconfirmed

func GetUnconfirmed() *Unconfirmed {
	db, err := GetLDBDataBase(DB_BLOCK)
	if err != nil {
		log.Fatal(err)
	}

	if _unconfirmed == nil {
		_unconfirmed = &Unconfirmed{
			db: db,
		}
	}

	return _unconfirmed
}

func (ucf *Unconfirmed) GetUnconfirmedMeta(addr *types.Address) (*ledger.UnconfirmedMeta, error) {
	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr)
	if err != nil {
		return nil, err
	}
	data, err := ucf.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, err
	}
	var ucfm = &ledger.UnconfirmedMeta{}
	if err := ucfm.DbDeserialize(data); err != nil {
		return nil, err
	}
	return ucfm, nil
}

func (ucf *Unconfirmed) GetUnconfirmedHashList(accountId *big.Int, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, accountId, tokenId.Bytes())
	if err != nil {
		return nil, err
	}
	data, err := ucf.db.Leveldb.Get(key, nil)
	if err != nil {
		return nil, err
	}
	hList, err := ledger.HashListDbDeserialize(data)
	if err != nil {
		return nil, err
	}
	return hList, nil
}

func (ucf *Unconfirmed) WriteMeta(batch *leveldb.Batch, addr *types.Address, meta *ledger.UnconfirmedMeta) error {
	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes())
	if err != nil {
		return err
	}
	data, err := meta.DbSerialize()
	if err != nil {
		return err
	}
	batch.Put(key, data)
	return nil
}

func (ucf *Unconfirmed) WriteHashList(batch *leveldb.Batch, accountId *big.Int, tokenId *types.TokenTypeId, hList []*types.Hash) error {
	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, accountId, tokenId.Bytes())
	if err != nil {
		return err
	}
	data, err := ledger.HashListDbSerialize(hList)
	if err != nil {
		return err
	}
	batch.Put(key, data)
	return nil
}

func (ucf *Unconfirmed) DeleteMeta(batch *leveldb.Batch, addr *types.Address) error {
	key, err := createKey(DBKP_UNCONFIRMEDMETA, addr.Bytes())
	if err != nil {
		return err
	}
	batch.Delete(key)
	return nil
}

func (ucf *Unconfirmed) DeleteHashList(batch *leveldb.Batch, accountId *big.Int, tokenId *types.TokenTypeId) error {
	key, err := createKey(DBKP_UNCONFIRMEDHASHLIST, accountId, tokenId.Bytes())
	if err != nil {
		return err
	}
	batch.Delete(key)
	return nil
}

func (ucf *Unconfirmed) BatchWrite(batch *leveldb.Batch, writeFunc func(batch *leveldb.Batch) error) error {
	return batchWrite(batch, ucf.db.Leveldb, func(context *batchContext) error {
		err := writeFunc(context.Batch)
		return err
	})
}