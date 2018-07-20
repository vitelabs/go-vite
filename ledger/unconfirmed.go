package ledger

import (
	"math/big"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb/proto"
	"github.com/vitelabs/go-vite/vitepb"
)

type UnconfirmedByToken struct {
	TokenId *types.TokenTypeId
	TotalBalance *big.Int
	HashList []*types.Hash
}

type UnconfirmedMeta struct{
	AccountId *big.Int
	TotalNumber *big.Int
	UnconfirmedList []*UnconfirmedByToken
}

func (ucfm *UnconfirmedMeta) DbDeserialize (buf []byte) error {
	unconfirmedMetaPB := &vitepb.UnconfirmedMeta{}
	if err := proto.Unmarshal(buf, unconfirmedMetaPB); err != nil {
		return err
	}
	ucfm.AccountId = &big.Int{}
	ucfm.AccountId.SetBytes(unconfirmedMetaPB.AccountId)

	ucfm.TotalNumber = &big.Int{}
	ucfm.TotalNumber.SetBytes(unconfirmedMetaPB.TotalNumber)

	var unconfirmedList []*UnconfirmedByToken
	if unconfirmedMetaPB.UnconfirmedList != nil {
		for _, unconfirmedByToken := range unconfirmedMetaPB.UnconfirmedList {
			//UnconfirmedByToken
			var oneUnconfirmed = &UnconfirmedByToken{}
			if err := oneUnconfirmed.GetUnconfirmedByTokenByDbPB(unconfirmedByToken); err != nil {
				return err
			}
			unconfirmedList = append(unconfirmedList, oneUnconfirmed)
		}
		ucfm.UnconfirmedList = unconfirmedList
	}
	return nil
}

func (ucfm *UnconfirmedMeta) DbSerialize () ([]byte, error) {
	var unconfirmedList []*vitepb.UnconfirmedByToken

	for _, unconfirmedByToken := range ucfm.UnconfirmedList {
		unconfirmedByTokenPB, err := unconfirmedByToken.SetUnconfirmedByTokenIntoPB()
		if err != nil {
			return nil, err
		}
		unconfirmedList = append(unconfirmedList, unconfirmedByTokenPB)
	}

	unconfirmedMeta := &vitepb.UnconfirmedMeta{
		AccountId:ucfm.AccountId.Bytes(),
		TotalNumber:ucfm.TotalNumber.Bytes(),
		UnconfirmedList: unconfirmedList,
	}

	data, err := proto.Marshal(unconfirmedMeta)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ucfbt *UnconfirmedByToken) SetUnconfirmedByTokenIntoPB () (*vitepb.UnconfirmedByToken, error) {
	var hashList [][]byte
	for _, hash := range ucfbt.HashList {
		hashList =  append(hashList, hash.Bytes())
	}
	unconfirmedByTokenPB := &vitepb.UnconfirmedByToken{
		TokenId: ucfbt.TokenId.Bytes(),
		TotalBalance: ucfbt.TotalBalance.Bytes(),
		HashList: hashList,
	}
	return unconfirmedByTokenPB, nil
}

func (ucfbt *UnconfirmedByToken) GetUnconfirmedByTokenByDbPB (unconfirmedByTokenPB *vitepb.UnconfirmedByToken) error {
	ucfbt.TotalBalance = &big.Int{}
	ucfbt.TotalBalance.SetBytes(unconfirmedByTokenPB.TotalBalance)

	tokenId, err := types.BytesToTokenTypeId(unconfirmedByTokenPB.TokenId)
	if err != nil {
		return err
	}
	ucfbt.TokenId = &tokenId

	var hashList []*types.Hash
	for _, hashPB := range unconfirmedByTokenPB.HashList {
		hash, err := types.BytesToHash(hashPB)
		if err != nil {
			return err
		}
		hashList = append(hashList, &hash)
	}
	ucfbt.HashList = hashList

	return nil
}
