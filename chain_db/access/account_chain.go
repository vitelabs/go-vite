package access

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/helper"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type AccountChain struct {
	db *leveldb.DB
}

func NewAccountChain(db *leveldb.DB) *AccountChain {
	return &AccountChain{
		db: db,
	}
}

func (ac *AccountChain) GetLatestBlock(accountId *big.Int) (*ledger.AccountBlock, error) {
	key, err := helper.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, "KEY_MAX")
	if err != nil {
		return nil, err
	}

	iter := ac.db.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, nil
	}
	block := &ledger.AccountBlock{}
	ddsErr := block.DbDeSerialize(iter.Value())

	return block, ddsErr
}

func (ac *AccountChain) GetBlockListByAccountId(accountId *big.Int, startHeight *big.Int, endHeight *big.Int) ([]*ledger.AccountBlock, error) {
	limitHeight := big.Int{}
	limitHeight.Add(endHeight, big.NewInt(1))

	limitKey, err := helper.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, limitHeight)
	if err != nil {
		return nil, err
	}

	startKey, err := helper.EncodeKey(database.DBKP_ACCOUNTBLOCK, accountId, startHeight)
	if err != nil {
		return nil, err
	}

	iter := ac.db.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	var blockList []*ledger.AccountBlock

	for iter.Next() {
		block := &ledger.AccountBlock{}
		err := block.DbDeSerialize(iter.Value())

		if err != nil {
			return nil, err
		}

		blockList = append(blockList, block)
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
