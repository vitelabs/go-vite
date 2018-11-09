// Data structures stored: key[DBKP_ONROADMETA.address.hash]=value[markType]
// markType: []byte("1"),represents true;[]byte("0"),represents false

package model

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
)

var keySize = 1 + types.AddressSize + types.HashSize

type OnroadSet struct {
	chain chain.Chain
}

func (o OnroadSet) db() *leveldb.DB {
	return o.chain.ChainDb().Db()
}
func NewOnroadSet(chain chain.Chain) *OnroadSet {
	return &OnroadSet{
		chain: chain,
	}
}

func (ucf *OnroadSet) GetCountByAddress(addr *types.Address) (count uint64, err error) {
	count = 0
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes())

	if err != nil {
		return 0, err
	}

	iter := ucf.db().NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	for iter.Next() {
		count += 1
	}
	return count, nil
}

func (ucf *OnroadSet) GetHashsByCount(count uint64, addr *types.Address) (hashs []*types.Hash, err error) {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes())
	if err != nil {
		return nil, err
	}

	iter := ucf.db().NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	i := uint64(1)
	for iter.Next() {
		if i > count {
			break
		}
		key := iter.Key()
		hash, err := types.BytesToHash(key[1+types.AddressSize:])
		if err != nil {
			continue
		}
		hashs = append(hashs, &hash)
		i++
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return hashs, nil
}

func (ucf *OnroadSet) GetHashList(addr *types.Address) (hashs []*types.Hash, err error) {
	createKey, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes())
	if err != nil {
		return nil, err
	}

	iter := ucf.db().NewIterator(util.BytesPrefix(createKey), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()
		hash, err := types.BytesToHash(key[1+types.AddressSize:])
		if err != nil {
			continue
		}
		hashs = append(hashs, &hash)
	}
	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}
	return hashs, nil
}

func (ucf *OnroadSet) WriteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	value := []byte{byte(0)}

	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	if batch == nil {
		if err := ucf.db().Put(key, value, nil); err != nil {
			return err
		}
	} else {
		batch.Put(key, value)
	}
	return nil
}

func (ucf *OnroadSet) DeleteMeta(batch *leveldb.Batch, addr *types.Address, hash *types.Hash) error {
	key, err := database.EncodeKey(database.DBKP_ONROADMETA, addr.Bytes(), hash.Bytes())
	if err != nil {
		return err
	}
	if batch == nil {
		if err := ucf.db().Delete(key, nil); err != nil {
			return err
		}
	} else {
		batch.Delete(key)
	}
	return nil
}

func (ucf *OnroadSet) WriteGidAddrList(batch *leveldb.Batch, gid *types.Gid, addrList []types.Address) error {
	key, err := database.EncodeKey(database.DBKP_GID_ADDR, gid.Bytes())
	if err != nil {
		return err
	}
	data, err := AddrListDbSerialize(addrList)
	if err != nil {
		return err
	}

	if batch == nil {
		if err := ucf.db().Put(key, data, nil); err != nil {
			return err
		}
	} else {
		batch.Put(key, data)
	}

	return nil
}

func (ucf *OnroadSet) GetContractAddrList(gid *types.Gid) ([]types.Address, error) {
	key, err := database.EncodeKey(database.DBKP_GID_ADDR, gid.Bytes())
	if err != nil {
		return nil, err
	}

	data, err := ucf.db().Get(key, nil)
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
		return ucf.db().Put(key, []byte{count}, nil)
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
			return ucf.db().Put(key, []byte{count}, nil)
		} else {
			return ucf.db().Delete(key, nil)
		}
	}
}
func (ucf *OnroadSet) DeleteReceiveErrCount(batch *leveldb.Batch, hash *types.Hash, addr *types.Address) error {
	key, err := database.EncodeKey(database.DBKP_ONROADRECEIVEERR, hash.Bytes(), addr.Bytes())
	if err != nil {
		return err
	}
	if _, err := ucf.db().Get(key, nil); err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
		return nil
	}
	if batch != nil {
		batch.Delete(key)
	} else {
		return ucf.db().Delete(key, nil)
	}
	return nil
}

func (ucf *OnroadSet) GetReceiveErrCount(hash *types.Hash, addr *types.Address) (uint8, error) {
	key, err := database.EncodeKey(database.DBKP_ONROADRECEIVEERR, hash.Bytes(), addr.Bytes())
	if err != nil {
		return 0, err
	}

	data, err := ucf.db().Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		return 0, nil
	}
	return uint8(data[0]), nil
}
