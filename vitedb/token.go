package vitedb

import (
	"errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type Token struct {
	db *DataBase
}

var _token *Token

func GetToken() *Token {
	db, err := GetLDBDataBase(DB_LEDGER)
	if err != nil {
		log15.Root().Crit(err.Error())
	}

	if _token == nil {
		_token = &Token{
			db: db,
		}
	}
	return _token
}

func (token *Token) BatchWrite(batch *leveldb.Batch, writeFunc func(batch *leveldb.Batch) error) error {
	return batchWrite(batch, token.db.Leveldb, func(context *batchContext) error {
		return writeFunc(context.Batch)
	})
}

func (token *Token) GetMintageBlockHashByTokenId(tokenId *types.TokenTypeId) (*types.Hash, error) {
	reader := token.db.Leveldb
	// Get mintage block hash
	key, err := createKey(DBKP_TOKENID_INDEX, tokenId.Bytes(), big.NewInt(0))

	if err != nil {
		return nil, err
	}
	mintageBlockHashBytes, err := reader.Get(key, nil)
	if err != nil {
		return nil, errors.New("Fail to query mintage block hash, Error is " + err.Error())
	}

	mintageBlockHash, err := types.BytesToHash(mintageBlockHashBytes)
	if err != nil {
		return nil, errors.New("Fail to query mintage block hash, Error is " + err.Error())
	}
	return &mintageBlockHash, nil
}

func (token *Token) getTokenIdList(key []byte) ([]*types.TokenTypeId, error) {
	reader := token.db.Leveldb

	iter := reader.NewIterator(util.BytesPrefix(key), nil)

	defer iter.Release()

	var tokenIdList []*types.TokenTypeId

	for iter.Next() {
		tokenId, err := types.BytesToTokenTypeId(iter.Value())
		if err != nil {
			return nil, err
		}
		tokenIdList = append(tokenIdList, &tokenId)
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return tokenIdList, nil
}

func (token *Token) GetTokenIdListByTokenName(tokenName string) ([]*types.TokenTypeId, error) {
	key, err := createKey(DBKP_TOKENNAME_INDEX, tokenName, nil)
	if err != nil {
		return nil, err
	}
	return token.getTokenIdList(key)
}

func (token *Token) GetTokenIdListByTokenSymbol(tokenSymbol string) ([]*types.TokenTypeId, error) {
	key, err := createKey(DBKP_TOKENSYMBOL_INDEX, tokenSymbol, nil)
	if err != nil {
		return nil, err
	}

	return token.getTokenIdList(key)
}

// 等vite-explorer-server从自己的数据库查数据时，这个方法就要删掉了，所以当前是hack实现
func (token *Token) GetTokenIdList(index int, num int, count int) ([]*types.TokenTypeId, error) {
	iterKey, err := createKey(DBKP_TOKENNAME_INDEX, nil)
	if err != nil {
		return nil, err
	}

	iter := token.db.Leveldb.NewIterator(util.BytesPrefix(iterKey), nil)

	defer iter.Release()

	for i := 0; i < index*count; i++ {
		if !iter.Next() {
			return nil, nil
		}
	}

	var tokenIdList []*types.TokenTypeId

	for i := 0; i < count*num; i++ {
		if !iter.Next() {
			break
		}

		tokenId, err := types.BytesToTokenTypeId(iter.Value())
		if err != nil {
			return nil, err
		}

		tokenIdList = append(tokenIdList, &tokenId)
	}

	return tokenIdList, nil

}

func (token *Token) GetLatestBlockHeightByTokenId(tokenId *types.TokenTypeId) (*big.Int, error) {
	key, err := createKey(DBKP_TOKENID_INDEX, tokenId.Bytes())
	if err != nil {
		return nil, err
	}

	iter := token.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, errors.New("GetLatestBlockHeightByTokenId failed, because token " + tokenId.String() + " doesn't exist.")
	}

	lastKey := iter.Key()
	partionList := deserializeKey(lastKey)

	latestBlockHeight := &big.Int{}
	latestBlockHeight.SetBytes(partionList[1])

	return latestBlockHeight, nil
}

func (token *Token) GetAccountBlockHashListByTokenId(index int, num int, count int, tokenId *types.TokenTypeId) ([]*types.Hash, error) {
	latestBlockHeight, err := token.GetLatestBlockHeightByTokenId(tokenId)
	if err != nil {
		return nil, err
	}

	limitKey, err := createKey(DBKP_TOKENID_INDEX, tokenId.Bytes(), latestBlockHeight.Add(latestBlockHeight, big.NewInt(1)))
	startKey, err := createKey(DBKP_TOKENID_INDEX, tokenId.Bytes(), big.NewInt(0))

	if err != nil {
		return nil, err
	}

	iter := token.db.Leveldb.NewIterator(&util.Range{Start: startKey, Limit: limitKey}, nil)
	defer iter.Release()

	if !iter.Last() {
		return nil, errors.New("GetAccountBlockHashList failed, because token " + tokenId.String() + " doesn't exist.")
	}

	var blockHashList []*types.Hash
	for i := 0; i < index*count; i++ {
		if !iter.Prev() {
			return blockHashList, nil
		}
	}

	for i := 0; i < num*count; i++ {
		blockHash := iter.Value()
		typeHash, err := types.BytesToHash(blockHash)
		if err != nil {
			return nil, err
		}
		blockHashList = append(blockHashList, &typeHash)
		if !iter.Prev() {
			break
		}
	}

	return blockHashList, nil
}

func (token *Token) getTopId(key []byte) *big.Int {
	iter := token.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		return big.NewInt(-1)
	}

	lastKey := iter.Key()
	partionList := deserializeKey(lastKey)

	if partionList == nil {
		return big.NewInt(0)
	}

	count := &big.Int{}
	count.SetBytes(partionList[0])

	return count
}

func (token *Token) getTokenNameCurrentTopId(tokenName string) (*big.Int, error) {
	key, err := createKey(DBKP_TOKENNAME_INDEX, tokenName, nil)

	if err != nil {
		return nil, err
	}
	return token.getTopId(key), nil
}

func (token *Token) getTokenSymbolCurrentTopId(tokenSymbol string) (*big.Int, error) {
	key, err := createKey(DBKP_TOKENSYMBOL_INDEX, tokenSymbol, nil)

	if err != nil {
		return nil, err
	}

	return token.getTopId(key), nil
}

func (token *Token) WriteTokenIdIndex(batch *leveldb.Batch, tokenId *types.TokenTypeId, blockHeightInToken *big.Int, accountBlockHash *types.Hash) error {
	key, err := createKey(DBKP_TOKENID_INDEX, tokenId.Bytes(), blockHeightInToken)
	if err != nil {
		return err
	}

	batch.Put(key, accountBlockHash.Bytes())
	return nil
}

func (token *Token) writeIndex(batch *leveldb.Batch, keyPrefix string, indexName string, currentTopId *big.Int, tokenId *types.TokenTypeId) error {

	topId := &big.Int{}
	topId.Add(currentTopId, big.NewInt(1))
	key, err := createKey(keyPrefix, indexName, topId)

	if err != nil {
		return err
	}

	batch.Put(key, tokenId.Bytes())
	return nil
}

func (token *Token) WriteTokenNameIndex(batchWriter *leveldb.Batch, tokenName string, tokenId *types.TokenTypeId) error {
	currentTopId, err := token.getTokenNameCurrentTopId(tokenName)
	if err != nil {
		return err
	}

	return token.writeIndex(batchWriter, DBKP_TOKENNAME_INDEX, tokenName, currentTopId, tokenId)
}

func (token *Token) WriteTokenSymbolIndex(batchWriter *leveldb.Batch, tokenSymbol string, tokenId *types.TokenTypeId) error {
	currentTopId, err := token.getTokenSymbolCurrentTopId(tokenSymbol)
	if err != nil {
		return err
	}
	return token.writeIndex(batchWriter, DBKP_TOKENSYMBOL_INDEX, tokenSymbol, currentTopId, tokenId)
}
