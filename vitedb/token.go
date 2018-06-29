package vitedb

import (
	"errors"
	"log"
	"github.com/syndtr/goleveldb/leveldb/util"
	"math/big"
	"github.com/syndtr/goleveldb/leveldb"
	"bytes"
)

type Token struct {
	db *DataBase
}

var _token *Token

func (Token) GetInstance () *Token {
	db, err := GetLDBDataBase(DB_BLOCK)
	if err != nil {
		log.Fatal(err)
	}

	if _token == nil {
		_token = &Token{
			db: db,
		}
	}
	return _token
}

func (token *Token) GetMintageBlockHashByTokenId(tokenId []byte) ([]byte, error){
	reader := token.db.Leveldb
	// Get mintage block hash
	key := createKey(DBKP_TOKENID_INDEX, tokenId, big.NewInt(0))
	mintageBlockHash, err := reader.Get(key, nil)
	if err != nil {
		return nil, errors.New("Fail to query mintage block hash, Error is " + err.Error())
	}

	return mintageBlockHash, nil
}

func (token *Token) getTokenIdList (key []byte) ([][]byte, error) {
	reader := token.db.Leveldb

	iter := reader.NewIterator(util.BytesPrefix(key), nil)

	defer iter.Release()

	var tokenIdList [][]byte
	for iter.Next() {
		tokenIdList = append(tokenIdList, iter.Value())
	}


	if err := iter.Error() ; err != nil {
		return nil, err
	}

	return tokenIdList, nil
}

func (token *Token) GetTokenIdListByTokenName(tokenName string) ([][]byte, error) {
	return token.getTokenIdList(createKey(DBKP_TOKENNAME_INDEX, []byte(tokenName), nil))
}

func (token *Token) GetTokenIdListByTokenSymbol (tokenSymbol string) ([][]byte, error) {
	return token.getTokenIdList(createKey(DBKP_TOKENSYMBOL_INDEX, []byte(tokenSymbol), nil))
}

// 等vite-explorer-server从自己的数据库查数据时，这个方法就要删掉了，所以当前是hack实现
func (token *Token) GetTokenIdList (index int, num int, count int) ([][]byte, error) {
	reader := token.db.Leveldb

	iter := reader.NewIterator(util.BytesPrefix(DBKP_TOKENID_INDEX), nil)
	defer iter.Release()

	for i:=0; i < index * count; i++ {
		if iter.Next() {
			return nil, nil
		}
	}

	var tokenIdList [][]byte
	for i:=0; i < count * num ; i++ {

		tokenIdList = append(tokenIdList, iter.Value())

		if iter.Next() {
			break
		}
	}

	return tokenIdList, nil

}

func (token *Token) getTopId (key []byte) *big.Int {
	iter := token.db.Leveldb.NewIterator(util.BytesPrefix(key), nil)
	defer iter.Release()

	if !iter.Last() {
		return big.NewInt(-1)
	}

	lastKey := iter.Key()
	partionList := bytes.Split(lastKey, []byte(".."))

	if len(partionList) == 1 {
		return big.NewInt(0)
	}

	count := &big.Int{}
	count.SetBytes(partionList[1])

	return count
}


func (token *Token) getTokenNameCurrentTopId (tokenName string) *big.Int {
	key := createKey(DBKP_TOKENNAME_INDEX, []byte(tokenName), nil)

	return token.getTopId(key)
}

func (token *Token) getTokenSymbolCurrentTopId (tokenSymbol string) *big.Int {
	key := createKey(DBKP_TOKENSYMBOL_INDEX, []byte(tokenSymbol), nil)

	return token.getTopId(key)

}

func (token *Token) WriteTokenIdIndex (batch *leveldb.Batch, tokenId []byte, blockHeightInToken *big.Int, accountBlockHash []byte) error {
	return batchWrite(batch, token.db.Leveldb, func(ctx *batchContext) error {
		key := createKey(DBKP_TOKENID_INDEX, tokenId, blockHeightInToken)
		ctx.Batch.Put(key, accountBlockHash)
		return nil
	})
}

func (token *Token) writeIndex (batch *leveldb.Batch, keyPrefix []byte, indexName string, currentTopId *big.Int, tokenId []byte) error {
	return batchWrite(batch, token.db.Leveldb, func(ctx *batchContext) error {
		currentTopIdStr := currentTopId.String()
		var key []byte
		if currentTopIdStr != "-1" {
			topId := &big.Int{}
			topId.Add(currentTopId, big.NewInt(1))
			key = createKey(keyPrefix, []byte(indexName), topId)
		} else {
			key = createKey(keyPrefix, []byte(indexName), nil)
		}

		ctx.Batch.Put(key, tokenId)
		return nil
	})
}

func (token *Token) WriteTokenNameIndex (batchWriter *leveldb.Batch, tokenName string, tokenId []byte) error {
	currentTopId := token.getTokenNameCurrentTopId(tokenName)
	return token.writeIndex(batchWriter, DBKP_TOKENNAME_INDEX, tokenName, currentTopId, tokenId)
}

func (token *Token) WriteTokenSymbolIndex (batchWriter *leveldb.Batch, tokenSymbol string, tokenId []byte) error {
	currentTopId := token.getTokenSymbolCurrentTopId(tokenSymbol)
	return token.writeIndex(batchWriter, DBKP_TOKENSYMBOL_INDEX, tokenSymbol, currentTopId, tokenId)
}