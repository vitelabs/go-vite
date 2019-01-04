package index

import (
	"encoding/binary"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

const (
	DBKP_BLOCK_LIST_BY_TOKEN = byte(1)

	DBKP_ACCOUNT_TOKEN_META = byte(2)

	DBKP_FILTER_TOKEN_CONSUME_ID = byte(3)
)

type FilterTokenIndex struct {
	db               *leveldb.DB
	chainInstance    chain.Chain
	BlockNumPerBuild uint64
	Meta             map[types.Address]map[types.TokenTypeId]types.Hash
}

func NewFilterTokenIndex() *FilterTokenIndex {
	// register

	return nil
}

func (fti *FilterTokenIndex) updateConsumeId(eventId uint64) error {
	key, _ := database.EncodeKey(DBKP_FILTER_TOKEN_CONSUME_ID)

	eventIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(eventIdBytes, eventId)

	return fti.db.Put(key, eventIdBytes, nil)
}

func (fti *FilterTokenIndex) getConsumeId() (uint64, error) {
	key, _ := database.EncodeKey(DBKP_FILTER_TOKEN_CONSUME_ID)
	value, err := fti.db.Get(key, nil)

	if err != nil {
		return 0, err
	}

	if len(value) <= 0 {
		return 0, nil
	}

	return binary.BigEndian.Uint64(value), nil
}

func (fti *FilterTokenIndex) build() error {
	consumeId, err := fti.getConsumeId()
	if err != nil {
		return err
	}

	latestBeId, err := fti.chainInstance.GetLatestBlockEventId()
	if err != nil {
		return err
	}

	for eventId := consumeId; eventId <= latestBeId; eventId++ {

	}

	//latestBlockEventId := fti.chainInstance.GetLatestBlockEventId()
	//// scan
	//lastAccountId, err := fti.chainInstance.ChainDb().Account.GetLastAccountId()
	//if err != nil {
	//	return err
	//}
	//
	//for i := uint64(1); i <= lastAccountId; i++ {
	//	startHeight := uint64(1)
	//	for {
	//		endHeight := startHeight + fti.BlockNumPerBuild
	//
	//		blocks, err := fti.chainInstance.ChainDb().Ac.GetBlockListByAccountId(i, startHeight, endHeight, true)
	//		if err != nil {
	//			return err
	//		}
	//
	//		if len(blocks) < 0 {
	//			break
	//		}
	//
	//		if err := fti.AddBlocks(i, blocks); err != nil {
	//			return err
	//		}
	//
	//		// do
	//		if blocks[len(blocks)-1].Height < endHeight {
	//			break
	//		}
	//
	//		startHeight = endHeight + 1
	//	}
	//}
	//return nil
}
func (fti *FilterTokenIndex) rollback(accountId uint64) (*types.Hash, error) {
	key, _ := database.EncodeKey(DBKP_ACCOUNT_TOKEN_META, accountId)

	iter := fti.db.NewIterator(util.BytesPrefix(key), nil)
	for iter.Next() {

	}

	if err := iter.Error(); err != nil && err != leveldb.ErrNotFound {
		return nil, err
	}

}
func (fti *FilterTokenIndex) getHeadHash(accountId uint64, tokenTypeId types.TokenTypeId) (*types.Hash, error) {
	key, _ := database.EncodeKey(DBKP_ACCOUNT_TOKEN_META, accountId, tokenTypeId)
	value, err := fti.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	hash, err := types.BytesToHash(value)
	return &hash, err
}

func (fti *FilterTokenIndex) saveHeadHash(batch *leveldb.Batch, accountId uint64, tokenTypeId types.TokenTypeId, hash types.Hash) {
	key, _ := database.EncodeKey(DBKP_ACCOUNT_TOKEN_META, accountId, tokenTypeId)
	value := hash.Bytes()

	batch.Put(key, value)
}

func (fti *FilterTokenIndex) deleteHeadHash(batch *leveldb.Batch, accountId uint64, tokenTypeId types.TokenTypeId) {
	key, _ := database.EncodeKey(DBKP_ACCOUNT_TOKEN_META, accountId, tokenTypeId)

	batch.Delete(key)
}

func (fti *FilterTokenIndex) AddBlocks(accountId uint64, blocks []*ledger.AccountBlock) error {
	// add
	batch := new(leveldb.Batch)

	unsavedHeadHash := make(map[types.TokenTypeId]types.Hash)

	for _, block := range blocks {
		key, _ := database.EncodeKey(DBKP_BLOCK_LIST_BY_TOKEN, accountId, block.Hash)

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
		unsavedHeadHash[block.TokenId] = block.Hash
	}

	// save head hash
	for tokenTypeId, prevHash := range unsavedHeadHash {
		fti.saveHeadHash(batch, accountId, tokenTypeId, prevHash)
	}

	return fti.db.Write(batch, nil)
}

func (fti *FilterTokenIndex) deleteBlocks(subLedger map[types.Address][]*ledger.AccountBlock) error {
	// delete
	batch := new(leveldb.Batch)

	for addr, blocks := range subLedger {
		account, err := fti.chainInstance.ChainDb().Account.GetAccountByAddress(&addr)

		if err != nil {
			return err
		}

		unsavedHeadHash := make(map[types.TokenTypeId]*types.Hash)
		for i := len(blocks) - 1; i >= 0; i-- {
			block := blocks[i]

			key, _ := database.EncodeKey(DBKP_BLOCK_LIST_BY_TOKEN, account.AccountId, block.Hash)
			var headHash *types.Hash
			if hash, ok := unsavedHeadHash[block.TokenId]; ok {
				headHash = hash
			} else {
				var err error
				headHash, err = fti.getHeadHash(account.AccountId, block.TokenId)
				if err != nil {
					return err
				}
			}

			if headHash == nil {
				// delete
				err := errors.New("head hash is nil")
				return err
			} else if *headHash == block.Hash {
				value, err := fti.db.Get(key, nil)
				if err != nil {
					return err
				}
				if len(value) > 0 {
					prevHashInToken, err := types.BytesToHash(value)
					if err != nil {
						return err
					}
					unsavedHeadHash[block.TokenId] = &prevHashInToken
				} else {
					unsavedHeadHash[block.TokenId] = nil
				}

			}

			batch.Delete(key)
		}

		for tokenTypeId, prevHash := range unsavedHeadHash {
			// save
			if prevHash != nil {
				fti.saveHeadHash(batch, account.AccountId, tokenTypeId, *prevHash)
			} else {
				fti.deleteHeadHash(batch, account.AccountId, tokenTypeId)
			}
		}

	}
	return fti.db.Write(batch, nil)
}

func (fti *FilterTokenIndex) GetBlockHashList(account *ledger.Account, originBlockHash *types.Hash, count uint64) []types.Hash {
	return nil
}
