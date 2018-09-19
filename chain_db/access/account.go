package access

import (
	"encoding/binary"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Account struct {
	db *leveldb.DB
}

func NewAccount(db *leveldb.DB) *Account {
	return &Account{
		db: db,
	}
}

func (accountAccess *Account) WriteAccountIndex(batch *leveldb.Batch, accountId uint64, accountAddress *types.Address) {
	accountIndexKey, _ := database.EncodeKey(database.DBKP_ACCOUNTID_INDEX, accountId)
	batch.Put(accountIndexKey, accountAddress.Bytes())
}

func (accountAccess *Account) WriteAccount(batch *leveldb.Batch, account *ledger.Account) error {
	// TODO key
	accountKey, _ := database.EncodeKey(database.DBKP_ACCOUNT, account.AccountAddress.Bytes())
	data, err := account.DbSerialize()
	if err != nil {
		return err
	}

	batch.Put(accountKey, data)
	return nil
}

func (accountAccess *Account) GetLastAccountId() (uint64, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNTID_INDEX)
	iter := accountAccess.db.NewIterator(util.BytesPrefix(key), nil)

	if !iter.Last() {
		err := iter.Error()
		if err != leveldb.ErrNotFound {
			return 0, err
		}
		return 0, nil
	}

	return binary.BigEndian.Uint64(iter.Key()[1:]), nil
}

func (accountAccess *Account) GetAddressById(accountId uint64) (*types.Address, error) {
	keyAccountAddress, _ := database.EncodeKey(database.DBKP_ACCOUNTID_INDEX, accountId)
	data, dgErr := accountAccess.db.Get(keyAccountAddress, nil)

	if dgErr != nil {
		return nil, dgErr
	}

	b2Address, b2Err := types.BytesToAddress(data)
	if b2Err != nil {
		return nil, b2Err
	}
	return &b2Address, nil
}

func (accountAccess *Account) GetAccountByAddress(address *types.Address) (*ledger.Account, error) {
	keyAccountMeta, _ := database.EncodeKey(database.DBKP_ACCOUNT, address.Bytes())

	data, dgErr := accountAccess.db.Get(keyAccountMeta, nil)
	if dgErr != nil {
		if dgErr != leveldb.ErrNotFound {
			return nil, dgErr
		}

		return nil, nil
	}
	account := &ledger.Account{}
	dsErr := account.DbDeserialize(data)

	if dsErr != nil {
		return nil, dsErr
	}

	return account, nil
}
