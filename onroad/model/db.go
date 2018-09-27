// Data structures stored: key[DBKP_ONROADMETA.address.hash]=value[markType]
// markType: []byte("1"),represents true;[]byte("0"),represents false

package model

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
)

type OnroadSet struct {
	db *leveldb.DB
}

func NewOnroadSet(db *leveldb.DB) *OnroadSet {
	return &OnroadSet{
		db: db,
	}
}

func (ucf *OnroadSet) GetCountByAddress(addr *types.Address) (count uint64, err error) {
	count = 0
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), "KEY_MAX")

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

func (ucf *OnroadSet) GetHashsByCount(count uint64, addr *types.Address) (hashs []*types.Hash, err error) {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), "KEY_MAX")
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

func (ucf *OnroadSet) GetHashList(addr *types.Address) (hashs []*types.Hash, err error) {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), "KEY_MAX")
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

func (ucf *OnroadSet) WriteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash, count uint8) error {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	if batch == nil {
		if err := ucf.db.Put(key, []byte{count}, nil); err != nil {
			return err
		}
	} else {
		batch.Put(key, []byte{count})
	}
	return nil
}

func (ucf *OnroadSet) DeleteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	if batch == nil {
		if err := ucf.db.Delete(key, nil); err != nil {
			return err
		}
	} else {
		batch.Delete(key)
	}
	return nil
}

func (ucf *OnroadSet) GetMeta(addr *types.Address, hash *types.Hash) ([]byte, error) {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}
	value, err := ucf.db.Get(key, nil)
	return value, nil
}

func (ucf *OnroadSet) WriteGidAddrList(batch *leveldb.Batch, gid *types.Gid, addrList []*types.Address) error {
	key, err := database.EncodeKey(database.DBKP_GID_ADDR, gid.Bytes())
	if err != nil {
		return err
	}
	data, err := AddrListDbSerialize(addrList)
	if err != nil {
		return err
	}

	if batch == nil {
		if err := ucf.db.Put(key, data, nil); err != nil {
			return err
		}
	} else {
		batch.Put(key, data)
	}

	return nil
}

func (ucf *OnroadSet) GetContractAddrList(gid *types.Gid) ([]*types.Address, error) {
	key, err := database.EncodeKey(database.DBKP_GID_ADDR, gid.Bytes())
	if err != nil {
		return nil, err
	}

	data, err := ucf.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return nil, err
		}
		return nil, nil
	}
	return AddrListDbDeserialize(data)
}

func (ucf *OnroadSet) IncreaseReceiveErrCount(batch *leveldb.Batch, hash *types.Hash, addr *types.Address) error {
	key, err := database.EncodeKey(database.DBKP_ONROADRECEIVEERR, hash.Bytes(), addr.Bytes())
	if err != nil {
		return err
	}
	count, err := ucf.GetReceiveErrCount(hash, addr)
	if err != nil {
		return err
	}
	count++
	if batch != nil {
		batch.Put(key, []byte{count})
		return nil
	} else {
		return ucf.db.Put(key, []byte{count}, nil)
	}
}

func (ucf *OnroadSet) DecreaseReceiveErrCount(batch *leveldb.Batch, hash *types.Hash, addr *types.Address) error {
	key, err := database.EncodeKey(database.DBKP_ONROADRECEIVEERR, hash.Bytes(), addr.Bytes())
	if err != nil {
		return err
	}
	count, err := ucf.GetReceiveErrCount(hash, addr)
	if err != nil {
		return err
	}
	count--
	if batch != nil {
		if count > 0 {
			batch.Put(key, []byte{count})
		} else {
			batch.Delete(key)
		}
		return nil
	} else {
		if count > 0 {
			return ucf.db.Put(key, []byte{count}, nil)
		} else {
			return ucf.db.Delete(key, nil)
		}
	}
}
func (ucf *OnroadSet) DeleteReceiveErrCount(batch *leveldb.Batch, hash *types.Hash, addr *types.Address) error {
	key, err := database.EncodeKey(database.DBKP_ONROADRECEIVEERR, hash.Bytes(), addr.Bytes())
	if err != nil {
		return err
	}
	if _, err := ucf.db.Get(key, nil); err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
		return nil
	}
	if batch != nil {
		batch.Delete(key)
	} else {
		return ucf.db.Delete(key, nil)
	}
	return nil
}

func (ucf *OnroadSet) GetReceiveErrCount(hash *types.Hash, addr *types.Address) (uint8, error) {
	key, err := database.EncodeKey(database.DBKP_ONROADRECEIVEERR, hash.Bytes(), addr.Bytes())
	if err != nil {
		return 0, err
	}

	data, err := ucf.db.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		return 0, nil
	}
	return uint8(data[0]), nil
}
