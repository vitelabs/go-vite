package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
)

type TokenAccess struct {
	store *vitedb.Token
	accountChainStore *vitedb.AccountChain
}

var _tokenAccess *TokenAccess

func (TokenAccess) GetInstance () *TokenAccess {
	if _tokenAccess == nil {
		_tokenAccess = &TokenAccess {
			store: vitedb.Token{}.GetInstance(),
			accountChainStore: vitedb.AccountChain{}.GetInstance(),
		}
	}
	return _tokenAccess
}

func (ta *TokenAccess) getListByTokenIdList (tokenIdList [][]byte) ([]*ledger.Token, error) {
	var tokenList []*ledger.Token
	for _, tokenId := range tokenIdList {
		token, err := ta.GetByTokenId(tokenId)
		if err != nil {
			return nil, err
		}
		tokenList = append(tokenList, token)
	}
	return tokenList, nil
}

func (ta *TokenAccess) GetListByTokenName (tokenName string) ([]*ledger.Token, error){
	tokenIdList, err:= ta.store.GetTokenIdListByTokenName(tokenName)
	if err != nil {
		return nil, err
	}

	return ta.getListByTokenIdList(tokenIdList)

}

func (ta *TokenAccess) GetListByTokenSymbol (tokenSymbol string) ([]*ledger.Token, error){
	tokenIdList, err:= ta.store.GetTokenIdListByTokenSymbol(tokenSymbol)
	if err != nil {
		return nil, err
	}

	return ta.getListByTokenIdList(tokenIdList)
}


func (ta *TokenAccess) GetByTokenId (tokenId []byte) (*ledger.Token, error)  {
	mintageBlockHash, err:= ta.store.GetMintageBlockHashByTokenId(tokenId)
	if err != nil {
		return nil, err
	}
	mintageBlock, err:= ta.accountChainStore.GetBlockByBlockHash(mintageBlockHash)
	if err != nil {
		return nil, err
	}

	return &ledger.Token{
		MintageBlock: mintageBlock,
	}, nil
}

func (ta *TokenAccess) GetList (index int, num int, count int) ([]*ledger.Token, error) {
	tokenIdList, err:= ta.store.GetTokenIdList(index, num, count)

	if err != nil {
		return nil, err
	}

	return ta.getListByTokenIdList(tokenIdList)
}



