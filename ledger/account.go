package ledger

import (
	"math/big"
)

type AccountSimpleToken struct {
	TokenId []byte
	LastAccountBlockHeight *big.Int
}

type AccountMeta struct {
	AccountId *big.Int
	TokenList []*AccountSimpleToken
}

type Account struct {
	AccountMeta
	blockHeight *big.Int
}


// modify by sanjin
// has to be query from accountBlockMeta?????
//
func (account *Account) GetBlockHeight () *big.Int {
	//return big.NewInt(456)
	return account.blockHeight
}

// add by sanjin
func (accountmeta *AccountMeta) GetTokenList () []*AccountSimpleToken {
	//tokenlist := []*AccountSimpleToken //make([]byte,0)
	//for _, accountsimpletoken := range accountmeta.TokenList{
	//	tokenlist = append(tokenlist,accountsimpletoken)
	//}
	//return tokenlist
	return accountmeta.TokenList
}