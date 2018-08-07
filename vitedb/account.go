package vitedb

import (
	"github.com/pkg/errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
)

type Account struct {
	db  *DataBase
	log log15.Logger
}

var _account *Account

func GetAccount() *Account {
	if _account == nil {
		db, err := GetLDBDataBase(DB_LEDGER)
		if err != nil {
			log15.Root().Crit(err.Error())
		}

		_account = &Account{
			db:  db,
			log: log15.New("module", "vitedb/account"),
		}
	}
	return _account
}

func (account *Account) WriteMeta(batch *leveldb.Batch, accountAddress *types.Address, meta *ledger.AccountMeta) error {
	key, err := createKey(DBKP_ACCOUNTMETA, accountAddress.Bytes())
	if err != nil {
		return err
	}
	data, err := meta.DbSerialize()
	if err != nil {
		return errors.Wrap(err, "vitedb.Account.WriteMeta")
	}

	batch.Put(key, data)
	return nil
}

func (account *Account) GetAccountMetaByAddress(hexAddress *types.Address) (*ledger.AccountMeta, error) {
	keyAccountMeta, ckErr := createKey(DBKP_ACCOUNTMETA, hexAddress.Bytes())
	if ckErr != nil {
		return nil, ckErr
	}
	data, dgErr := account.db.Leveldb.Get(keyAccountMeta, nil)
	if dgErr != nil {
		account.log.Error("GetAccountMetaByAddress func db.Get()", "dgErr", dgErr)
		return nil, dgErr
	}
	accountMeter := &ledger.AccountMeta{}
	dsErr := accountMeter.DbDeserialize(data)
	if dsErr != nil {
		account.log.Error(dsErr.Error())
		return nil, dsErr
	}

	return accountMeter, nil
}

func (account *Account) GetLastAccountId() (*big.Int, error) {
	key, err := createKey(DBKP_ACCOUNTID_INDEX, nil)
	if err != nil {
		return nil, err
	}

	iter := account.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)

	if !iter.Last() {
		return nil, nil
	}

	lastKey := iter.Key()
	partionList := deserializeKey(lastKey)

	if partionList == nil {
		return big.NewInt(0), nil
	}

	accountId := &big.Int{}
	accountId.SetBytes(partionList[0])

	return accountId, nil
}

func (account *Account) WriteAccountIdIndex(batch *leveldb.Batch, accountId *big.Int, accountAddress *types.Address) error {
	key, err := createKey(DBKP_ACCOUNTID_INDEX, accountId)
	if err != nil {
		return err
	}

	batch.Put(key, accountAddress.Bytes())
	return nil
}

func (account *Account) GetAddressById(accountId *big.Int) (*types.Address, error) {
	keyAccountAddress, ckErr := createKey(DBKP_ACCOUNTID_INDEX, accountId)
	if ckErr != nil {
		return nil, ckErr
	}
	data, dgErr := account.db.Leveldb.Get(keyAccountAddress, nil)
	if dgErr != nil {
		account.log.Error("GetAddressById func db.Get()", "dgErr", dgErr)
		return nil, dgErr
	}
	b2Address, b2Err := types.BytesToAddress(data)
	if b2Err != nil {
		return nil, b2Err
	}
	return &b2Address, nil
}

// To get the latest existing account addresses of the accountChain
func (account *Account) GetAccountList() ([]*types.Address, error) {
	key, ckErr := createKey(DBKP_ACCOUNTID_INDEX, nil)
	if ckErr != nil {
		return nil, ckErr
	}
	iter := account.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()
	if itErr := iter.Error(); itErr != nil {
		return nil, itErr
	}
	var accountList []*types.Address
	for iter.Next() {
		address, err := types.BytesToAddress(iter.Value())
		if err != nil {
			return nil, err
		}
		accountList = append(accountList, &address)
	}
	return accountList, nil
}
