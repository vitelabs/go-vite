package vitedb

import (
	"errors"
	"github.com/vitelabs/go-vite/ledger"
)

type Token struct {
	db *DataBase
	accountchainStore *AccountChain
}

var _token *Token

func (Token) GetInstance () *Token {
	db := GetDataBase(DB_BLOCK)

	if _token == nil {
		_token = &Token{
			db: db,
			accountchainStore: AccountChain{}.GetInstance(),
		}
	}
	return _token
}

func (token *Token) GetMintageBlockByTokenId(tokenId []byte) (*ledger.AccountBlock, error){
	// Get mintage block hash
	accountBlockHash, err := token.db.Get(createKey(DBKP_TOKENID_INDEX, tokenId, []byte("0")))
	if err != nil {
		return nil, errors.New("Fail to query mintage block hash, Error is " + err.Error())
	}

	mintageBlock, err := token.accountchainStore.GetBlockByBlockHash(accountBlockHash)
	if err != nil {
		return nil, errors.New("Fail to query account block by block hash, Error is " + err.Error())
	}

	return mintageBlock, nil
}

func (token *Token) GetTokenIdByTokenName(tokenName string) ([][]byte, error) {
	// Get tokenId
	//tokenId, err := token.db.Get(createKey(DBKP_TOKENNAME_INDEX, []byte(tokenName)))
	return nil, nil
}

func (token *Token) GetTokenIdByTokenSymbol (tokenSymbol string) ([]byte, error) {
	return nil, nil
}

func (Token *Token) WriteTokenIdIndex (tokenId []byte, accountBlockHash []byte) error {

	return nil
}