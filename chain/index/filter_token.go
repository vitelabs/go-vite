package index

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
)

type FilterTokenIndex struct {
	chainInstance chain.Chain
	Meta          map[types.Address]map[types.TokenTypeId]types.Hash
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
			endHeight := startHeight + 999

			blocks, err := fti.chainInstance.ChainDb().Ac.GetBlockListByAccountId(i, startHeight, endHeight, true)
			if err != nil {
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

func (fti *FilterTokenIndex) getHeadHash(account *ledger.Account, tokenTypeId types.TokenTypeId) (*types.Hash, error) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNT_TOKEN_META, account.AccountId, tokenTypeId)
	db := fti.chainInstance.ChainDb().Db()
	value, err := db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	hash, err := types.BytesToHash(value)
	return &hash, err
}

func (fti *FilterTokenIndex) saveHeadHash(batch *leveldb.Batch, account *ledger.Account, tokenTypeId types.TokenTypeId, hash types.Hash) {
	key, _ := database.EncodeKey(database.DBKP_ACCOUNT_TOKEN_META, account.AccountId, tokenTypeId)
	value := hash.Bytes()

	batch.Put(key, value)
}

func (fti *FilterTokenIndex) addBlocks(account *ledger.Account, vmBlocks []*vm_context.VmAccountBlock) error {
	// add
	batch := new(leveldb.Batch)
	for _, block := range vmBlocks {
		key, _ := database.EncodeKey(database.DBKP_BLOCK_LIST_BY_TOKEN, account.AccountId, block.AccountBlock.Hash)
		prevHashInToken, err := fti.getHeadHash(account, block.AccountBlock.TokenId)
		if err != nil {
			return err
		}

		value := prevHashInToken.Bytes()

		// batch write
		batch.Put(key, value)

		fti.saveHeadHash(batch, account, block.AccountBlock.TokenId, block.AccountBlock.Hash)
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
				headHash, err = fti.getHeadHash(account, block.TokenId)
				if err != nil {
					return err
				}
			}

			if headHash != nil &&
				*headHash == block.Hash {
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
			fti.saveHeadHash(batch, account, tokenTypeId, prevHash)
		}

	}
	return fti.chainInstance.ChainDb().Commit(batch)
}

func (fti *FilterTokenIndex) GetBlockHashList(account *ledger.Account, originBlockHash *types.Hash, count uint64) []types.Hash {
	return nil
}
