package ledger

import (
	"math/big"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/vitepb"
)

type TokenInfo struct {
	TokenId     *types.TokenTypeId
	TotalAmount *big.Int
}

type UnconfirmedMeta struct {
	AccountId     *big.Int
	TotalNumber   *big.Int
	TokenInfoList []*TokenInfo
}

func (ucfm *UnconfirmedMeta) DbDeserialize(buf []byte) error {
	unconfirmedMetaPB := &vitepb.UnconfirmedMeta{}
	if err := proto.Unmarshal(buf, unconfirmedMetaPB); err != nil {
		return err
	}
	ucfm.AccountId = &big.Int{}
	ucfm.AccountId.SetBytes(unconfirmedMetaPB.AccountId)

	ucfm.TotalNumber = &big.Int{}
	ucfm.TotalNumber.SetBytes(unconfirmedMetaPB.TotalNumber)

	var tokenInfoList []*TokenInfo
	if unconfirmedMetaPB.TokenInfoList != nil {
		for _, tokenInfoPB := range unconfirmedMetaPB.TokenInfoList {
			//UnconfirmedByToken
			var tokenInfo = &TokenInfo{}
			if err := tokenInfo.GetTokenInfoByDbPB(tokenInfoPB); err != nil {
				return err
			}
			tokenInfoList = append(tokenInfoList, tokenInfo)
		}
		ucfm.TokenInfoList = tokenInfoList
	}
	return nil
}

func (ucfm *UnconfirmedMeta) DbSerialize() ([]byte, error) {
	var unconfirmedList []*vitepb.TokenInfo

	for _, unconfirmedByToken := range ucfm.TokenInfoList {
		unconfirmedByTokenPB, err := unconfirmedByToken.SetTokenInfoIntoPB()
		if err != nil {
			return nil, err
		}
		unconfirmedList = append(unconfirmedList, unconfirmedByTokenPB)
	}

	unconfirmedMeta := &vitepb.UnconfirmedMeta{
		AccountId:     ucfm.AccountId.Bytes(),
		TotalNumber:   ucfm.TotalNumber.Bytes(),
		TokenInfoList: unconfirmedList,
	}

	data, err := proto.Marshal(unconfirmedMeta)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ti *TokenInfo) SetTokenInfoIntoPB() (*vitepb.TokenInfo, error) {
	unconfirmedByTokenPB := &vitepb.TokenInfo{
		TokenId:     ti.TokenId.Bytes(),
		TotalAmount: ti.TotalAmount.Bytes(),
	}
	return unconfirmedByTokenPB, nil
}

func (ti *TokenInfo) GetTokenInfoByDbPB(unconfirmedByTokenPB *vitepb.TokenInfo) error {
	ti.TotalAmount = &big.Int{}
	ti.TotalAmount.SetBytes(unconfirmedByTokenPB.TotalAmount)

	tokenId, err := types.BytesToTokenTypeId(unconfirmedByTokenPB.TokenId)
	if err != nil {
		return err
	}
	ti.TokenId = &tokenId
	return nil
}

//type HashList []*types.Hash
//
//func (hList *HashList) DbSerialize () ([]byte, error) {
//	var hashList [][]byte
//	for _, hash := range *hList {
//		hashList =  append(hashList, hash.Bytes())
//	}
//	var hListPB = &vitepb.HashList{
//		HashList:             hashList,
//	}
//	data, err := proto.Marshal(hListPB)
//	if err != nil {
//		return nil, err
//	}
//	return data, nil
//}
//
//func (hList *HashList) DbDeserialize (buf []byte) error {
//	var hListPB = &vitepb.HashList{}
//	if err := proto.Unmarshal(buf, hListPB); err != nil {
//		return err
//	}
//	var hashList []*types.Hash
//	for _, hashPB := range hListPB.HashList {
//		hash, tpErr := types.BytesToHash(hashPB)
//		if tpErr != nil {
//			return tpErr
//		}
//		hashList = append(hashList, &hash)
//	}
//	*hList = hashList
//	return nil
//}
//
//func TypeHashToHashList(b []*types.Hash) HashList {
//	var h HashList
//	copy(h[:], b)
//	return h
//}

func HashListDbSerialize(hList []*types.Hash) ([]byte, error) {
	var hashList [][]byte
	for _, hash := range hList {
		hashList = append(hashList, hash.Bytes())
	}
	var hListPB = &vitepb.HashList{
		HashList: hashList,
	}
	data, err := proto.Marshal(hListPB)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func HashListDbDeserialize(buf []byte) ([]*types.Hash, error) {
	var hListPB = &vitepb.HashList{}
	if err := proto.Unmarshal(buf, hListPB); err != nil {
		return nil, err
	}
	var hashList []*types.Hash
	for _, hashPB := range hListPB.HashList {
		hash, err := types.BytesToHash(hashPB)
		if err != nil {
			return nil, err
		}
		hashList = append(hashList, &hash)
	}
	return hashList, nil
}
