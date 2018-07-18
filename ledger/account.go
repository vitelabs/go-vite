package ledger

import (
	"math/big"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb/proto"
	"bytes"
)

var GenesisAccount, _ = types.BytesToAddress([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})

type AccountSimpleToken struct {
	TokenId *types.TokenTypeId
	LastAccountBlockHeight *big.Int
}

type AccountMeta struct {
	AccountId *big.Int
	TokenList []*AccountSimpleToken
}

type Account struct {
	AccountMeta
	BlockHeight *big.Int
}

func (am *AccountMeta) SetTokenInfoByTokenId (tokenId *types.TokenTypeId, tokenInfo *AccountSimpleToken) {

	// Get token info of account
	for index, token := range am.TokenList {
		if bytes.Equal(token.TokenId.Bytes(), tokenId.Bytes()) {
			am.TokenList[index] = tokenInfo
			break
		}
	}
}


func (am *AccountMeta) GetTokenInfoByTokenId (tokenId *types.TokenTypeId) *AccountSimpleToken {
	var tokenInfo *AccountSimpleToken

	// Get token info of account
	for _, token := range am.TokenList {
		if bytes.Equal(token.TokenId.Bytes(), tokenId.Bytes()) {
			tokenInfo = token
			break
		}
	}
	return tokenInfo
}

func (am *AccountMeta) GetTokenList () []*AccountSimpleToken {
	return am.TokenList
}

func (am *AccountMeta) DbSerialize () ([]byte, error) {
	var pbTokenList []*vitepb.AccountSimpleToken
	for _, lAccountSimpleToken := range am.TokenList {
		pbAccountSimpleToken := &vitepb.AccountSimpleToken{
			TokenId: lAccountSimpleToken.TokenId.Bytes(),
			LastAccountBlockHeight: lAccountSimpleToken.LastAccountBlockHeight.Bytes(),
		}
		pbTokenList = append(pbTokenList, pbAccountSimpleToken)
	}
	accountMetaPB := &vitepb.AccountMeta{
		TokenList: pbTokenList,
	}
	if am.AccountId != nil {
		accountMetaPB.AccountId = am.AccountId.Bytes()
	}
	return proto.Marshal(accountMetaPB)
}

func (am *AccountMeta) DbDeserialize (buf []byte) error {
	accountMetaPB := &vitepb.AccountMeta{}
	if err := proto.Unmarshal(buf, accountMetaPB); err != nil {
		return err
	}
	am.AccountId = &big.Int{}
	am.AccountId.SetBytes(accountMetaPB.AccountId)

	var lTokenList []*AccountSimpleToken
	for _, pbAccountSimpleToken := range accountMetaPB.TokenList {
		labHeight := &big.Int{}
		tTokenIdType, ttErr := types.BytesToTokenTypeId(pbAccountSimpleToken.TokenId)
		if ttErr != nil {
			return ttErr
		}
		lAccountSimpleToken := &AccountSimpleToken{
			TokenId: &tTokenIdType,
			LastAccountBlockHeight: labHeight.SetBytes(pbAccountSimpleToken.LastAccountBlockHeight),
		}
		lTokenList = append(lTokenList, lAccountSimpleToken)
	}
	am.TokenList = lTokenList

	return nil
}