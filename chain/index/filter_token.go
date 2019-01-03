package index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type FilterTokenIndex struct {
	chainInstance    chain.Chain
	BlockNumPerBuild uint64
	Meta             map[types.Address]map[types.TokenTypeId]types.Hash
}

func NewFilterTokenIndex() {
	// register
}

func (fti *FilterTokenIndex) build() error {
	// scan
	lastAccountId, err := fti.chainInstance.ChainDb().Account.GetLastAccountId()
	if err != nil {
		return err
	}

	for i := uint64(1); i <= lastAccountId; i++ {
		startHeight := uint64(1)
		for {
			endHeight := startHeight + fti.BlockNumPerBuild

			blocks, err := fti.chainInstance.ChainDb().Ac.GetBlockListByAccountId(i, startHeight, endHeight, true)
			if err != nil {
				return err
			}

			if len(blocks) < 0 {
				break
			}

			if err := fti.AddBlocks(i, blocks); err != nil {
				return err
			}

			// do
			if blocks[len(blocks)-1].Height < endHeight {
				break
			}

			startHeight = endHeight + 1
		}
	}
	return nil

}

func (fti *FilterTokenIndex) getHeadHash(accountId uint64, tokenTypeId types.TokenTypeId) (*types.Hash, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNT_TOKEN_META, accountId, tokenTypeId)
	db := fti.chainInstance.ChainDb().Db()
	value, err := db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	hash, err := types.BytesToHash(value)
	return &hash, err
}

func (fti *FilterTokenIndex) saveHeadHash(batch *leveldb.Batch, accountId uint64, tokenTypeId types.TokenTypeId, hash types.Hash) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNT_TOKEN_META, accountId, tokenTypeId)
	value := hash.Bytes()

	batch.Put(key, value)
}

func (fti *FilterTokenIndex) AddBlocks(accountId uint64, blocks []*ledger.AccountBlock) error {
	// add
	batch := new(leveldb.Batch)

	unsavedHeadHash := make(map[types.TokenTypeId]types.Hash)

	for _, block := range blocks {
		key, _ := database.EncodeKey(database.DBKP_BLOCK_LIST_BY_TOKEN, accountId, block.Hash)

		var prevHashInToken *types.Hash
		if hash, ok := unsavedHeadHash[block.TokenId]; ok {
			prevHashInToken = &hash
		} else {
			var err error
			prevHashInToken, err = fti.getHeadHash(accountId, block.TokenId)

			if err != nil {
				return err
			}
		}

		var value []byte
		if prevHashInToken != nil {
			value = prevHashInToken.Bytes()
		}

		// batch write
		batch.Put(key, value)

	}
	for tokenTypeId, prevHash := range unsavedHeadHash {
		fti.saveHeadHash(batch, accountId, tokenTypeId, prevHash)
	}

	return fti.chainInstance.ChainDb().Commit(batch)
}

func (fti *FilterTokenIndex) deleteBlocks(subLedger map[types.Address][]*ledger.AccountBlock) error {
	// delete
	batch := new(leveldb.Batch)
	db := fti.chainInstance.ChainDb().Db()

	for addr, blocks := range subLedger {
		account, err := fti.chainInstance.ChainDb().Account.GetAccountByAddress(&addr)

		if err != nil {
			return err
		}

		unsavedHeadHash := make(map[types.TokenTypeId]types.Hash)
		for i := len(blocks) - 1; i >= 0; i-- {
			block := blocks[i]

			key, _ := database.EncodeKey(database.DBKP_BLOCK_LIST_BY_TOKEN, account.AccountId, block.Hash)
			var headHash *types.Hash
			if hash, ok := unsavedHeadHash[block.TokenId]; ok {
				headHash = &hash
			} else {
				var err error
				headHash, err = fti.getHeadHash(account.AccountId, block.TokenId)
				if err != nil {
					return err
				}
			}

			if headHash == nil {
				// delete
			} else if *headHash == block.Hash {
				value, err := db.Get(key, nil)
				if err != nil {
					return err
				}

				prevHashInToken, err := types.BytesToHash(value)
				if err != nil {
					return err
				}

				unsavedHeadHash[block.TokenId] = prevHashInToken
			}

			batch.Delete(key)
		}

		for tokenTypeId, prevHash := range unsavedHeadHash {
			// save
			fti.saveHeadHash(batch, account.AccountId, tokenTypeId, prevHash)
		}

	}
	return fti.chainInstance.ChainDb().Commit(batch)
}

func (fti *FilterTokenIndex) GetBlockHashList(account *ledger.Account, originBlockHash *types.Hash, count uint64) []types.Hash {
	return nil
}
