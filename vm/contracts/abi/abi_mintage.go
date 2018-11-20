package abi

import (
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
		{"type":"variable","name":"mintage","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]}
	]`

	MethodNameMintage             = "Mintage"
	MethodNameMintageCancelPledge = "CancelPledge"
	VariableNameMintage           = "mintage"
)

var (
	ABIMintage, _ = abi.JSONToABIContract(strings.NewReader(jsonMintage))
)

type ParamMintage struct {
	TokenId     types.TokenTypeId
	TokenName   string
	TokenSymbol string
	TotalSupply *big.Int
	Decimals    uint8
}

// TODO no left pad
func GetMintageKey(tokenId types.TokenTypeId) []byte {
	return helper.LeftPadBytes(tokenId.Bytes(), types.HashSize)
}
func GetTokenIdFromMintageKey(key []byte) types.TokenTypeId {
	tokenId, _ := types.BytesToTokenTypeId(key[types.HashSize-types.TokenTypeIdSize:])
	return tokenId
}

func NewTokenId(accountAddress types.Address, accountBlockHeight uint64, prevBlockHash types.Hash, snapshotHash types.Hash) types.TokenTypeId {
	return types.CreateTokenTypeId(
		accountAddress.Bytes(),
		new(big.Int).SetUint64(accountBlockHeight).Bytes(),
		prevBlockHash.Bytes(),
		snapshotHash.Bytes())
}

func GetTokenById(db StorageDatabase, tokenId types.TokenTypeId) *types.TokenInfo {
	data := db.GetStorageBySnapshotHash(&AddressMintage, GetMintageKey(tokenId), nil)
	if len(data) > 0 {
		tokenInfo := new(types.TokenInfo)
		ABIMintage.UnpackVariable(tokenInfo, VariableNameMintage, data)
		return tokenInfo
	}
	return nil
}

func GetTokenMap(db StorageDatabase) map[types.TokenTypeId]*types.TokenInfo {
	defer monitor.LogTime("vm", "GetTokenMap", time.Now())
	iterator := db.NewStorageIteratorBySnapshotHash(&AddressMintage, nil, nil)
	tokenInfoMap := make(map[types.TokenTypeId]*types.TokenInfo)
	if iterator == nil {
		return tokenInfoMap
	}
	for {
		key, value, ok := iterator.Next()
		if !ok {
			break
		}
		tokenId := GetTokenIdFromMintageKey(key)
		tokenInfo := new(types.TokenInfo)
		if err := ABIMintage.UnpackVariable(tokenInfo, VariableNameMintage, value); err == nil {
			tokenInfoMap[tokenId] = tokenInfo
		}
	}
	return tokenInfoMap
}
