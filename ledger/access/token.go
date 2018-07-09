package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/syndtr/goleveldb/leveldb"
	"math/big"
)

type TokenAccess struct {
	store *vitedb.Token
	accountChainStore *vitedb.AccountChain
}


var tokenAccess = &TokenAccess {
	store: vitedb.GetToken(),
	accountChainStore: vitedb.GetAccountChain(),
}

func GetTokenAccess () *TokenAccess {
	return tokenAccess
}

func (ta *TokenAccess) WriteMintage (batch *leveldb.Batch, mintage *ledger.Mintage, bloch *ledger.AccountBlock) (error) {
	// Write TokenIdIndex
	if err := ta.store.WriteTokenIdIndex(batch, mintage.Id, big.NewInt(0), bloch.Hash); err != nil{
		return err
	}

	// Write TokenNameIndex
	if err := ta.store.WriteTokenNameIndex(batch, mintage.Name, mintage.Id); err != nil{
		return err
	}

	// Write TokenSymbolIndex
	if err := ta.store.WriteTokenSymbolIndex(batch, mintage.Symbol, mintage.Id); err != nil{
		return err
	}

	return nil
}


func (ta *TokenAccess) getListByTokenIdList (tokenIdList []*types.TokenTypeId) ([]*ledger.Token, error) {
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


func (ta *TokenAccess) GetByTokenId (tokenId *types.TokenTypeId) (*ledger.Token, error)  {
	mintageBlockHash, err:= ta.store.GetMintageBlockHashByTokenId(tokenId)
	if err != nil {
		return nil, err
	}
	mintageBlock, err:= ta.accountChainStore.GetBlockByHash(mintageBlockHash)
	if err != nil {
		return nil, err
	}

	token, err := ledger.NewToken(mintageBlock)
	return token, nil
}

func (ta *TokenAccess) GetList (index int, num int, count int) ([]*ledger.Token, error) {
	tokenIdList, err:= ta.store.GetTokenIdList(index, num, count)

	if err != nil {
		return nil, err
	}

	return ta.getListByTokenIdList(tokenIdList)
}