package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/helper"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
)

type AccountChain struct {
	db  *leveldb.DB
	log log15.Logger
}

func NewAccountChain(db *leveldb.DB) *AccountChain {
	return &AccountChain{
		db:  db,
		log: log15.New("module", "ledger/access/account"),
	}
}

func (ac *AccountChain) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	return nil, nil
}

func (ac *AccountChain) GetLatestBlockHeightByAccountId(accountId *big.Int) (*big.Int, error) {
	key, err := helper.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, "KEY_MAX")
	if err != nil {
		return nil, err
	}

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		ac.log.Info("GetLatestBlockHeightByAccountId failed, because account " + accountId.String() + " doesn't exist.")
		return nil, nil
	}

	dbKey, _ := helper.DecodeKey(iter.Key())

	latestBlockHeight := &big.Int{}
	latestBlockHeight.SetBytes(dbKey.KeyPartionList[1][1:])
	return latestBlockHeight, nil
}

func (ac *AccountChain) GetBlockListByAccountId(accountId *big.Int, index int, num int, count int) ([]*ledger.AccountBlock, error) {
	latestBlockHeight, err := ac.GetLatestBlockHeightByAccountId(accountId)
	if err != nil {
		return nil, err
	}

	limitIndex := latestBlockHeight
	limitIndex.Add(limitIndex, big.NewInt(1))

	limitKey, err := helper.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, limitIndex)
	if err != nil {
		return nil, err
	}

	startKey, err := helper.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, big.NewInt(1))
	if err != nil {
		return nil, err
	}

	iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, nil
	}

	for i := 0; i < index*count; i++ {
		if !iter.Prev() {
			return nil, nil
		}
	}

	var blockList []*ledger.AccountBlock

	for i := 0; i < num*count; i++ {
		block := &ledger.AccountBlock{}
		err := block.DbDeSerialize(iter.Value())

		if err != nil {
			return nil, err
		}

		blockList = append(blockList, block)

		if !iter.Prev() {
			break
		}
	}

	return blockList, nil
}

func (ac *AccountChain) GetBlockMeta(blockHash *types.Hash) (*ledger.AccountBlockMeta, error) {
	key, err := helper.EncodeKey(database.DBKP_ACCOUNTBLOCKMETA, blockHash.Bytes())
	if err != nil {
		return nil, err
	}
	blockMetaBytes, err := ac.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	blockMeta := &ledger.AccountBlockMeta{}
	if err := blockMeta.DbDeSerialize(blockMetaBytes); err != nil {
		return nil, err
	}

	return blockMeta, nil
}
