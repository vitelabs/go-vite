package access

import (
	"github.com/vitelabs/go-vite/vitedb"
	"github.com/vitelabs/go-vite/ledger"
)

type TokenAccess struct {
	store *vitedb.Token

}

func (TokenAccess) New () *TokenAccess {
	return &TokenAccess {
		store: vitedb.Token{}.GetInstance(),
	}
}


func (ta *TokenAccess) GetByTokenName () (*ledger.Token, error){
	return nil, nil
}

func (ta *TokenAccess) GetByTokenSymbol () (*ledger.Token, error){
	return nil, nil
}


func (ta *TokenAccess) GetByTokenId (tokenId []byte) (*ledger.Token, error)  {
	mintageBlock, err:= ta.store.GetMintageBlockByTokenId(tokenId)
	return &ledger.Token{
		MintageBlock: mintageBlock,
	}, err
}



