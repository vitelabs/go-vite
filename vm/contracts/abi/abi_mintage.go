package abi

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"strings"
	"time"
)

const (
	jsonMintage = `
	[
		{"type":"function","name":"Mintage","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"}]},
		{"type":"function","name":"CancelPledge","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"function","name":"Mint","inputs":[{"name":"isReIssuable","type":"bool"},{"name":"tokenId","type":"tokenId"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"}]},
		{"type":"function","name":"Issue","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"amount","type":"uint256"},{"name":"beneficial","type":"address"}]},
		{"type":"function","name":"Burn","inputs":[]},
		{"type":"function","name":"TransferOwner","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"newOwner","type":"address"}]},
		{"type":"function","name":"ChangeTokenType","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"variable","name":"mintage","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
		{"type":"variable","name":"tokenInfo","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"},{"name":"isReIssuable","type":"bool"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"}]},
		{"type":"event","name":"mint","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"issue","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"burn","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"address","type":"address"},{"name":"amount","type":"uint256"}]},
		{"type":"event","name":"transferOwner","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"owner","type":"address"}]},
		{"type":"event","name":"changeTokenType","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]}
	]`

	MethodNameMintage             = "Mintage"
	MethodNameMintageCancelPledge = "CancelPledge"
	MethodNameMint                = "Mint"
	MethodNameIssue               = "Issue"
	MethodNameBurn                = "Burn"
	MethodNameTransferOwner       = "TransferOwner"
	MethodNameChangeTokenType     = "ChangeTokenType"
	VariableNameMintage           = "mintage"
	VariableNameTokenInfo         = "tokenInfo"
	EventNameMint                 = "mint"
	EventNameIssue                = "issue"
	EventNameBurn                 = "burn"
	EventNameTransferOwner        = "transferOwner"
	EventNameChangeTokenType      = "changeTokenType"
)

var (
	ABIMintage, _ = abi.JSONToABIContract(strings.NewReader(jsonMintage))
)

type ParamMintage struct {
	TokenId       types.TokenTypeId
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

// TODO no left pad
func GetMintageKey(tokenId types.TokenTypeId) []byte {
	return helper.LeftPadBytes(tokenId.Bytes(), types.HashSize)
}
func GetTokenIdFromMintageKey(key []byte) types.TokenTypeId {
	tokenId, _ := types.BytesToTokenTypeId(key[types.HashSize-types.TokenTypeIdSize:])
	return tokenId
}
func IsMintageKey(key []byte) bool {
	return len(key) == types.HashSize
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

func NewTokenId(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}

func GetTokenById(db StorageDatabase, tokenId types.TokenTypeId) *types.TokenInfo {
	data := db.GetStorageBySnapshotHash(&types.AddressMintage, GetMintageKey(tokenId), nil)
	if len(data) > 0 {
		tokenInfo, _ := ParseTokenInfo(data)
		return tokenInfo
	}
	return nil
}

func GetTokenMap(db StorageDatabase) map[types.TokenTypeId]*types.TokenInfo {
	defer monitor.LogTime("vm", "GetTokenMap", time.Now())
	iterator := db.NewStorageIteratorBySnapshotHash(&types.AddressMintage, nil, nil)
	tokenInfoMap := make(map[types.TokenTypeId]*types.TokenInfo)
	if iterator == nil {
		return tokenInfoMap
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		if !IsMintageKey(key) {
			continue
		}
		tokenId := GetTokenIdFromMintageKey(key)
		if tokenInfo, err := ParseTokenInfo(value); err == nil {
			tokenInfoMap[tokenId] = tokenInfo
		}
	}
	return tokenInfoMap
}

func ParseTokenInfo(data []byte) (*types.TokenInfo, error) {
	if len(data) == 0 {
		return nil, errors.New("token info data is nil")
	}
	tokenInfo := new(types.TokenInfo)
	if data[31] == 224 {
		err := ABIMintage.UnpackVariable(tokenInfo, VariableNameMintage, data)
		return tokenInfo, err
	} else {
		err := ABIMintage.UnpackVariable(tokenInfo, VariableNameTokenInfo, data)
		return tokenInfo, err
	}
}
