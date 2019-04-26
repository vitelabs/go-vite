package abi

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strings"
	"time"
)

const (
	jsonMintage = `
	[
		{"type":"function","name":"CancelMintPledge","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"function","name":"Mint","inputs":[{"name":"isReIssuable","type":"bool"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"}]},
		{"type":"function","name":"Issue","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"amount","type":"uint256"},{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"Burn","inputs":[]},
		{"type":"function","name":"TransferOwner","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"newOwner","type":"address"}]},
		{"type":"function","name":"ChangeTokenType","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"function","name":"GetTokenInfo","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"callback","name":"GetTokenInfo","inputs":[{"name":"exist","type":"bool"},{"name":"decimals","type":"uint8"},{"name":"tokenSymbol","type":"string"},{"name":"index","type":"uint16"}]},
		{"type":"variable","name":"tokenInfo","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"pledgeAddr","type":"address"},{"name":"isReIssuable","type":"bool"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"},{"name":"index","type":"uint16"}]},
		{"type":"variable","name":"tokenNameIndex","inputs":[{"name":"nextIndex","type":"uint16"}]},
		{"type":"event","name":"mint","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"issue","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"burn","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"address","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"event","name":"transferOwner","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"owner","type":"address"}]},
		{"type":"event","name":"changeTokenType","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]}
	]`

	MethodNameCancelMintPledge = "CancelMintPledge"
	MethodNameMint             = "Mint"
	MethodNameIssue            = "Issue"
	MethodNameBurn             = "Burn"
	MethodNameTransferOwner    = "TransferOwner"
	MethodNameChangeTokenType  = "ChangeTokenType"
	MethodNameGetTokenInfo     = "GetTokenInfo"
	VariableNameTokenInfo      = "tokenInfo"
	VariableNameTokenNameIndex = "tokenNameIndex"
	EventNameMint              = "mint"
	EventNameIssue             = "issue"
	EventNameBurn              = "burn"
	EventNameTransferOwner     = "transferOwner"
	EventNameChangeTokenType   = "changeTokenType"
)

var (
	ABIMintage, _ = abi.JSONToABIContract(strings.NewReader(jsonMintage))
)

type ParamMintage struct {
	TokenName     string
	TokenSymbol   string
	TotalSupply   *big.Int
	Decimals      uint8
	MaxSupply     *big.Int
	OwnerBurnOnly bool
	IsReIssuable  bool
}

type ParamIssue struct {
	TokenId    types.TokenTypeId
	Amount     *big.Int
	Beneficial types.Address
}

type ParamTransferOwner struct {
	TokenId  types.TokenTypeId
	NewOwner types.Address
}

func GetMintageKey(tokenId types.TokenTypeId) []byte {
	return tokenId.Bytes()
}
func GetNextIndexKey(tokenSymbol string) []byte {
	return types.DataHash([]byte(tokenSymbol)).Bytes()
}
func GetTokenIdFromMintageKey(key []byte) types.TokenTypeId {
	tokenId, _ := types.BytesToTokenTypeId(key)
	return tokenId
}
func IsMintageKey(key []byte) bool {
	return len(key) == types.TokenTypeIdSize
}
func GetOwnerTokenIdListKey(owner types.Address) []byte {
	return owner.Bytes()
}
func AppendTokenId(oldIdList []byte, tokenId types.TokenTypeId) []byte {
	return append(oldIdList, tokenId.Bytes()...)
}
func DeleteTokenId(oldIdList []byte, tokenId types.TokenTypeId) []byte {
	for i := 0; i < len(oldIdList)/types.TokenTypeIdSize; i++ {
		if bytes.Equal(oldIdList[i*types.TokenTypeIdSize:(i+1)*types.TokenTypeIdSize], tokenId.Bytes()) {
			newIdList := make([]byte, len(oldIdList)-types.TokenTypeIdSize)
			copy(newIdList[0:i*types.TokenTypeIdSize], oldIdList[0:i*types.TokenTypeIdSize])
			copy(newIdList[i*types.TokenTypeIdSize:], oldIdList[(i+1)*types.TokenTypeIdSize:])
			return newIdList
		}
	}
	return oldIdList
}

func NewTokenId(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(
		accountAddress.Bytes(),
		helper.LeftPadBytes(new(big.Int).SetUint64(accountBlockHeight).Bytes(), 8),
		prevBlockHash.Bytes())
}

func GetTokenById(db StorageDatabase, tokenId types.TokenTypeId) (*types.TokenInfo, error) {
	if *db.Address() != types.AddressMintage {
		return nil, util.ErrAddressNotMatch
	}
	data, err := db.GetValue(GetMintageKey(tokenId))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		tokenInfo, _ := ParseTokenInfo(data)
		return tokenInfo, nil
	}
	return nil, nil
}

func GetTokenMap(db StorageDatabase) (map[types.TokenTypeId]*types.TokenInfo, error) {
	if *db.Address() != types.AddressMintage {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getTokenMap"}, time.Now())
	iterator, err := db.NewStorageIterator(nil)
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	tokenInfoMap := make(map[types.TokenTypeId]*types.TokenInfo)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, err
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), IsMintageKey) {
			continue
		}
		tokenId := GetTokenIdFromMintageKey(iterator.Key())
		if tokenInfo, err := ParseTokenInfo(iterator.Value()); err == nil {
			tokenInfoMap[tokenId] = tokenInfo
		}
	}
	return tokenInfoMap, nil
}

func GetTokenMapByOwner(db StorageDatabase, owner types.Address) (tokenInfoMap map[types.TokenTypeId]*types.TokenInfo, err error) {
	if *db.Address() != types.AddressMintage {
		return nil, util.ErrAddressNotMatch
	}
	defer monitor.LogTimerConsuming([]string{"vm", "getTokenMapByOwner"}, time.Now())
	tokenIdList, err := db.GetValue(GetOwnerTokenIdListKey(owner))
	if err != nil {
		return nil, err
	}
	tokenInfoMap = make(map[types.TokenTypeId]*types.TokenInfo)
	for i := 0; i < len(tokenIdList)/types.TokenTypeIdSize; i++ {
		tokenId, _ := types.BytesToTokenTypeId(tokenIdList[i*types.TokenTypeIdSize : (i+1)*types.TokenTypeIdSize])
		tokenInfoMap[tokenId], err = GetTokenById(db, tokenId)
		if err != nil {
			return nil, err
		}
	}
	return tokenInfoMap, nil
}

func ParseTokenInfo(data []byte) (*types.TokenInfo, error) {
	if len(data) == 0 {
		return nil, util.ErrDataNotExist
	}
	tokenInfo := new(types.TokenInfo)
	err := ABIMintage.UnpackVariable(tokenInfo, VariableNameTokenInfo, data)
	return tokenInfo, err
}
