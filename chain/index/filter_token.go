package index

import (
	"encoding/binary"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	errors2 "github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/chain_db/database"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	DBKP_BLOCK_LIST_BY_TOKEN = byte(1)

	DBKP_ACCOUNT_TOKEN_META = byte(2)

	DBKP_FILTER_TOKEN_CONSUME_ID = byte(3)

	DBKP_HEAD_HASH = byte(4)
)

const (
	STOP  = 1
	START = 2
)

type FilterTokenIndex struct {
	db *leveldb.DB

	dataDirName      string
	log              log15.Logger
	chainInstance    chain.Chain
	EventNumPerBatch uint64

	status   int
	ticker   *time.Ticker
	terminal chan struct{}
	wg       sync.WaitGroup
}

// TODO register process func
func NewFilterTokenIndex(cfg *config.Config, chainInstance chain.Chain) (*FilterTokenIndex, error) {
	// register
	fti := &FilterTokenIndex{
		log:         log15.New("module", "filter_token"),
		dataDirName: filepath.Join(cfg.DataDir, "ledger_index"),

		chainInstance:    chainInstance,
		EventNumPerBatch: 1000,
	}

	db, err := fti.initDb()
	if err != nil {
		err := errors.New("initDb failed, error is " + err.Error())
		fti.log.Error(err.Error(), "method", "NewFilterTokenIndex")

		return nil, err
	}

	fti.db = db

	return fti, nil
}

func (fti *FilterTokenIndex) Start() {
	fti.ticker = time.NewTicker(time.Second * 2)
	fti.wg.Add(1)
	go func() {
		defer fti.wg.Done()
		for {
			select {
			case <-fti.ticker.C:

			case <-fti.terminal:
				return
			}
		}
	}()
}

func (fti *FilterTokenIndex) initDb() (*leveldb.DB, error) {
	db, err := database.NewLevelDb(fti.dataDirName)
	if err != nil {
		switch err.(type) {
		case *errors2.ErrCorrupted:
			// clear
			return fti.clearAndInitDb()
		default:
			fti.log.Error("NewLevelDb failed, error is "+err.Error(), "method", "initDb")
			return nil, nil
		}
	}

	if db == nil {
		err := errors.New("NewFilterTokenIndex failed, db is nil")
		fti.log.Error(err.Error(), "method", "initDb")
		return nil, err
	}

	if isConsistency, err := fti.checkConsistency(); err != nil {
		fti.log.Error("checkConsistency failed, error is "+err.Error(), "method", "initDb")
		return nil, err
	} else if !isConsistency {
		return fti.clearAndInitDb()
	}

	return db, nil
}

func (fti *FilterTokenIndex) checkConsistency() (bool, error) {
	latestBlockEventId, err := fti.chainInstance.GetLatestBlockEventId()
	if err != nil {
		return false, err
	}

	consumeId, err := fti.getConsumeId()
	if err != nil {
		return false, err
	}

	if consumeId > latestBlockEventId {
		return false, nil
	}
	return true, nil

}

func (fti *FilterTokenIndex) clearAndInitDb() (*leveldb.DB, error) {
	if fti.db != nil {
		if closeErr := fti.db.Close(); closeErr != nil {
			return nil, errors.New("Close db failed, error is " + closeErr.Error())
		}
	}

	if err := os.RemoveAll(fti.dataDirName); err != nil && err != os.ErrNotExist {
		return nil, errors.New("Remove " + fti.dataDirName + " failed, error is " + err.Error())
	}

	fti.db = nil
	return fti.initDb()
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

// todo save consume id
func (fti *FilterTokenIndex) build() error {
	consumeId, err := fti.getConsumeId()
	if err != nil {
		return err
	}

	latestBeId, err := fti.chainInstance.GetLatestBlockEventId()
	if err != nil {
		return err
	}

	unsavedBlocks := make(map[types.Address][]*ledger.AccountBlock)

	notFoundBlocks := make(map[types.Hash]struct{})

	eventNum := uint64(0)
	for eventId := consumeId; eventId <= latestBeId; eventId++ {
		eventType, blockHashList, err := fti.chainInstance.GetEvent(eventId)
		if err != nil {
			return err
		}
		switch eventType {
		// AddAccountBlocksEvent = byte(1)
		case byte(1):
			for _, blockHash := range blockHashList {
				block, err := fti.chainInstance.GetAccountBlockByHash(&blockHash)
				if err != nil {
					// .log.Error("GetAccountBlockByHash failed, error is "+err.Error(), "method", "send")
					return err
				}

				if block == nil {
					notFoundBlocks[blockHash] = struct{}{}
					continue
				}

				//fti.AddBlocks(block.Meta.AccountId, []*ledger.AccountBlock{block})
				unsavedBlocks[block.AccountAddress] = append(unsavedBlocks[block.AccountAddress], block)
			}
		case byte(2):
			for _, hash := range blockHashList {
				if _, ok := notFoundBlocks[hash]; ok {
					delete(notFoundBlocks, hash)
					continue
				}
				if err := fti.deleteHash(hash); err != nil {
					return err
				}
			}
		}

		eventNum++
		if eventNum >= fti.EventNumPerBatch {
			for addr, blocks := range unsavedBlocks {
				account, err := fti.chainInstance.GetAccount(&addr)
				if err != nil {
					return err
				}

				fti.AddBlocks(account.AccountId, blocks)
			}
			unsavedBlocks = make(map[types.Address][]*ledger.AccountBlock)
			eventNum = 0
		}
		if err := fti.updateConsumeId(eventId); err != nil {
			return err
		}
	}
	return nil
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

	key2, _ := database.EncodeKey(DBKP_HEAD_HASH, hash)

	accountIdBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(accountIdBytes, accountId)
	value2 := append(accountIdBytes, tokenTypeId.Bytes()...)
	batch.Put(key2, value2)
}

func (fti *FilterTokenIndex) deleteHash(headHash types.Hash) error {
	key, _ := database.EncodeKey(DBKP_HEAD_HASH, headHash)
	value, err := fti.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil
		}
		return err
	}
	if len(value) <= 0 {
		return nil
	}

	accountId := binary.BigEndian.Uint64(value[:8])
	tokenTypeId, err := types.BytesToTokenTypeId(value[8:])
	if err != nil {
		return err
	}

	batch := new(leveldb.Batch)
	newHeadHash := &headHash

	for {
		deletedKey, _ := database.EncodeKey(DBKP_HEAD_HASH, newHeadHash)
		batch.Delete(deletedKey)

		prevHash, err := fti.getPrevHash(*newHeadHash)
		if err != nil {
			return err
		}
		newHeadHash = prevHash

		if newHeadHash == nil {
			break
		}

		isExisted, err := fti.chainInstance.IsAccountBlockExisted(*prevHash)
		if err != nil {
			return err
		}

		if isExisted {
			break
		}
	}
	fti.deleteHeadHash(batch, accountId, tokenTypeId)
	fti.deleteHeadHashIndex(batch, headHash)

	if newHeadHash != nil {
		fti.saveHeadHash(batch, accountId, tokenTypeId, *newHeadHash)
	}

	return fti.db.Write(batch, nil)
}

func (fti *FilterTokenIndex) getPrevHash(hash types.Hash) (*types.Hash, error) {
	key, _ := database.EncodeKey(DBKP_BLOCK_LIST_BY_TOKEN, hash)

	value, err := fti.db.Get(key, nil)
	if err != nil {
		return nil, err
	}

	prevHash, err := types.BytesToHash(value)
	if err != nil {
		return nil, err
	}

	return &prevHash, nil
}

func (fti *FilterTokenIndex) deleteHeadHashIndex(batch *leveldb.Batch, hash types.Hash) {
	key, _ := database.EncodeKey(DBKP_HEAD_HASH, hash)

	batch.Delete(key)
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
		key, _ := database.EncodeKey(DBKP_BLOCK_LIST_BY_TOKEN, block.Hash)

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

			key, _ := database.EncodeKey(DBKP_BLOCK_LIST_BY_TOKEN, block.Hash)
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
				fti.deleteHeadHashIndex(batch, block.Hash)
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
