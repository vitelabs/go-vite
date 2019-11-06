package abi

import (
	"bytes"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"strings"
)

const (
	jsonAsset = `
	[
		{"type":"function","name":"Mint","inputs":[{"name":"isReIssuable","type":"bool"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"maxSupply","type":"uint256"},{"name":"isOwnerBurnOnly","type":"bool"}]},
		{"type":"function","name":"IssueToken","inputs":[{"name":"isReIssuable","type":"bool"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"maxSupply","type":"uint256"},{"name":"isOwnerBurnOnly","type":"bool"}]},

		{"type":"function","name":"Issue","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"amount","type":"uint256"},{"name":"receiveAddress","type":"address"}]},
		{"type":"function","name":"ReIssue","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"amount","type":"uint256"},{"name":"receiveAddress","type":"address"}]},

		{"type":"function","name":"Burn","inputs":[]},

		{"type":"function","name":"TransferOwner","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"newOwner","type":"address"}]},
		{"type":"function","name":"TransferOwnership","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"newOwner","type":"address"}]},

		{"type":"function","name":"ChangeTokenType","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"function","name":"DisableReIssue","inputs":[{"name":"tokenId","type":"tokenId"}]},

		{"type":"function","name":"GetTokenInfo","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"bid","type":"uint8"}]},
		{"type":"function","name":"GetTokenInformation","inputs":[{"name":"tokenId","type":"tokenId"}]},
		{"type":"callback","name":"GetTokenInfo","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"bid","type":"uint8"},{"name":"exist","type":"bool"},{"name":"decimals","type":"uint8"},{"name":"tokenSymbol","type":"string"},{"name":"index","type":"uint16"},{"name":"ownerAddress","type":"address"}]},
		{"type":"callback","name":"GetTokenInformation","inputs":[{"name":"id","type":"bytes32"},{"name":"tokenId","type":"tokenId"},{"name":"exist","type":"bool"},{"name":"isReIssuable","type":"bool"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"maxSupply","type":"uint256"},{"name":"isOwnerBurnOnly","type":"bool"},{"name":"index","type":"uint16"},{"name":"ownerAddress","type":"address"}]},		

		{"type":"variable","name":"tokenInfo","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"isReIssuable","type":"bool"},{"name":"maxSupply","type":"uint256"},{"name":"ownerBurnOnly","type":"bool"},{"name":"index","type":"uint16"}]},
		{"type":"variable","name":"tokenIndex","inputs":[{"name":"nextIndex","type":"uint16"}]},
		
		{"type":"event","name":"mint","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"issueToken","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		
		{"type":"event","name":"issue","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"reIssue","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		
		{"type":"event","name":"burn","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"address","type":"address"},{"name":"amount","type":"uint256"}]},

		{"type":"event","name":"transferOwner","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"owner","type":"address"}]},
		{"type":"event","name":"transferOwnership","inputs":[{"name":"tokenId","type":"tokenId","indexed":true},{"name":"owner","type":"address"}]},
		
		{"type":"event","name":"changeTokenType","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]},
		{"type":"event","name":"disableReIssue","inputs":[{"name":"tokenId","type":"tokenId","indexed":true}]}
	]`

	MethodNameIssue               = "Mint"
	MethodNameIssueV2             = "IssueToken"
	MethodNameReIssue             = "Issue"
	MethodNameReIssueV2           = "ReIssue"
	MethodNameBurn                = "Burn"
	MethodNameTransferOwnership   = "TransferOwner"
	MethodNameTransferOwnershipV2 = "TransferOwnership"
	MethodNameDisableReIssue      = "ChangeTokenType"
	MethodNameDisableReIssueV2    = "DisableReIssue"
	MethodNameGetTokenInfo        = "GetTokenInfo"
	MethodNameGetTokenInfoV3      = "GetTokenInformation"
	VariableNameTokenInfo         = "tokenInfo"
	VariableNameTokenIndex        = "tokenIndex"
)

var (
	// ABIAsset is abi definition of asset contract
	ABIAsset, _ = abi.JSONToABIContract(strings.NewReader(jsonAsset))
)

type ParamIssue struct {
	TokenName       string
	TokenSymbol     string
	TotalSupply     *big.Int
	Decimals        uint8
	MaxSupply       *big.Int
	IsOwnerBurnOnly bool
	IsReIssuable    bool
}

type ParamReIssue struct {
	TokenId        types.TokenTypeId
	Amount         *big.Int
	ReceiveAddress types.Address
}

type ParamTransferOwnership struct {
	TokenId  types.TokenTypeId
	NewOwner types.Address
}

type ParamGetTokenInfo struct {
	TokenId types.TokenTypeId
	Bid     uint8
}

// GetTokenInfoKey generate db key for token info
func GetTokenInfoKey(tokenID types.TokenTypeId) []byte {
	return tokenID.Bytes()
}

// GetTokenIdFromTokenInfoKey decode token id from token info key
func GetTokenIdFromTokenInfoKey(key []byte) types.TokenTypeId {
	tokenID, _ := types.BytesToTokenTypeId(key)
	return tokenID
}
func isTokenInfoKey(key []byte) bool {
	return len(key) == types.TokenTypeIdSize
}

// GetNextTokenIndexKey generate db key for next token index
func GetNextTokenIndexKey(tokenSymbol string) []byte {
	return types.DataHash([]byte(tokenSymbol)).Bytes()
}

// GetTokenIDListKey generate db key for token id list
func GetTokenIDListKey(owner types.Address) []byte {
	return owner.Bytes()
}

// AppendTokenID add token id into id list
func AppendTokenID(oldIDList []byte, tokenID types.TokenTypeId) []byte {
	return append(oldIDList, tokenID.Bytes()...)
}

// DeleteTokenID remove token id from id list
func DeleteTokenID(oldIDList []byte, tokenID types.TokenTypeId) []byte {
	for i := 0; i < len(oldIDList)/types.TokenTypeIdSize; i++ {
		if bytes.Equal(oldIDList[i*types.TokenTypeIdSize:(i+1)*types.TokenTypeIdSize], tokenID.Bytes()) {
			newIDList := make([]byte, len(oldIDList)-types.TokenTypeIdSize)
			copy(newIDList[0:i*types.TokenTypeIdSize], oldIDList[0:i*types.TokenTypeIdSize])
			copy(newIDList[i*types.TokenTypeIdSize:], oldIDList[(i+1)*types.TokenTypeIdSize:])
			return newIDList
		}
	}
	return oldIDList
}

// GetTokenByID query token info by id
func GetTokenByID(db StorageDatabase, tokenID types.TokenTypeId) (*types.TokenInfo, error) {
	if *db.Address() != types.AddressAsset {
		return nil, util.ErrAddressNotMatch
	}
	data, err := db.GetValue(GetTokenInfoKey(tokenID))
	if err != nil {
		return nil, err
	}
	if len(data) > 0 {
		tokenInfo, _ := parseTokenInfo(data)
		return tokenInfo, nil
	}
	return nil, nil
}

// GetTokenMap query all token info map
func GetTokenMap(db StorageDatabase) (map[types.TokenTypeId]*types.TokenInfo, error) {
	if *db.Address() != types.AddressAsset {
		return nil, util.ErrAddressNotMatch
	}
	iterator, err := db.NewStorageIterator(nil)
	if err != nil {
		return nil, err
	}
	defer iterator.Release()
	tokenInfoMap := make(map[types.TokenTypeId]*types.TokenInfo)
	for {
		if !iterator.Next() {
			if iterator.Error() != nil {
				return nil, iterator.Error()
			}
			break
		}
		if !filterKeyValue(iterator.Key(), iterator.Value(), isTokenInfoKey) {
			continue
		}
		tokenID := GetTokenIdFromTokenInfoKey(iterator.Key())
		if tokenInfo, err := parseTokenInfo(iterator.Value()); err == nil {
			tokenInfoMap[tokenID] = tokenInfo
		}
	}
	return tokenInfoMap, nil
}

// GetTokenMapByOwner query token info map by owner
func GetTokenMapByOwner(db StorageDatabase, owner types.Address) (tokenInfoMap map[types.TokenTypeId]*types.TokenInfo, err error) {
	if *db.Address() != types.AddressAsset {
		return nil, util.ErrAddressNotMatch
	}
	tokenIDList, err := db.GetValue(GetTokenIDListKey(owner))
	if err != nil {
		return nil, err
	}
	tokenInfoMap = make(map[types.TokenTypeId]*types.TokenInfo)
	for i := 0; i < len(tokenIDList)/types.TokenTypeIdSize; i++ {
		tokenID, _ := types.BytesToTokenTypeId(tokenIDList[i*types.TokenTypeIdSize : (i+1)*types.TokenTypeIdSize])
		tokenInfoMap[tokenID], err = GetTokenByID(db, tokenID)
		if err != nil {
			return nil, err
		}
	}
	return tokenInfoMap, nil
}

func parseTokenInfo(data []byte) (*types.TokenInfo, error) {
	if len(data) == 0 {
		return nil, util.ErrDataNotExist
	}
	tokenInfo := new(types.TokenInfo)
	err := ABIAsset.UnpackVariable(tokenInfo, VariableNameTokenInfo, data)
	return tokenInfo, err
}
