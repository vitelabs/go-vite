package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_db"
	"sort"
)

type MintageApi struct {
	chain chain.Chain
	vite  *vite.Vite
	log   log15.Logger
}

func NewMintageApi(vite *vite.Vite) *MintageApi {
	return &MintageApi{
		chain: vite.Chain(),
		vite:  vite,
		log:   log15.New("module", "rpc_api/mintage_api"),
	}
}

func (m MintageApi) String() string {
	return "MintageApi"
}

type MintageParams struct {
	TokenName     string
	TokenSymbol   string
	TotalSupply   string
	Decimals      uint8
	IsReIssuable  bool
	MaxSupply     string
	OwnerBurnOnly bool
}

// Private
func (m *MintageApi) GetMintData(param MintageParams) ([]byte, error) {
	totalSupply, err := stringToBigInt(&param.TotalSupply)
	if err != nil {
		return nil, err
	}
	maxSupply, err := stringToBigInt(&param.MaxSupply)
	if err != nil {
		return nil, err
	}
	return abi.ABIAssert.PackMethod(abi.MethodNameIssue, param.IsReIssuable, param.TokenName, param.TokenSymbol, totalSupply, param.Decimals, maxSupply, param.OwnerBurnOnly)
}

type IssueParams struct {
	TokenId    types.TokenTypeId
	Amount     string
	Beneficial types.Address
}

// Private
func (m *MintageApi) GetIssueData(param IssueParams) ([]byte, error) {
	amount, err := stringToBigInt(&param.Amount)
	if err != nil {
		return nil, err
	}
	return abi.ABIAssert.PackMethod(abi.MethodNameReIssue, param.TokenId, amount, param.Beneficial)
}

// Private
func (m *MintageApi) GetBurnData() ([]byte, error) {
	return abi.ABIAssert.PackMethod(abi.MethodNameBurn)
}

type TransferOwnerParams struct {
	TokenId  types.TokenTypeId
	NewOwner types.Address
}

// Private
func (m *MintageApi) GetTransferOwnerData(param TransferOwnerParams) ([]byte, error) {
	return abi.ABIAssert.PackMethod(abi.MethodNameTransferOwnership, param.TokenId, param.NewOwner)
}

// Private
func (m *MintageApi) GetChangeTokenTypeData(tokenId types.TokenTypeId) ([]byte, error) {
	return abi.ABIAssert.PackMethod(abi.MethodNameDisableReIssue, tokenId)
}

// Deprecated: use contract_getTokenInfoList instead
func (m *MintageApi) GetTokenInfoList(index int, count int) (*TokenInfoList, error) {
	db, err := getVmDb(m.chain, types.AddressAssert)
	if err != nil {
		return nil, err
	}
	tokenMap, err := abi.GetTokenMap(db)
	if err != nil {
		return nil, err
	}
	listLen := len(tokenMap)
	tokenList := make([]*RpcTokenInfo, 0)
	for tokenId, tokenInfo := range tokenMap {
		tokenList = append(tokenList, RawTokenInfoToRpc(tokenInfo, tokenId))
	}
	sort.Sort(byName(tokenList))
	start, end := getRange(index, count, listLen)
	return &TokenInfoList{listLen, tokenList[start:end]}, nil
}

// Deprecated: use contract_getTokenInfoById instead
func (m *MintageApi) GetTokenInfoById(tokenId types.TokenTypeId) (*RpcTokenInfo, error) {
	db, err := getVmDb(m.chain, types.AddressAssert)
	if err != nil {
		return nil, err
	}
	tokenInfo, err := abi.GetTokenByID(db, tokenId)
	if err != nil {
		return nil, err
	}
	if tokenInfo != nil {
		return RawTokenInfoToRpc(tokenInfo, tokenId), nil
	}
	return nil, nil
}

// Deprecated: use contract_getTokenInfoListByOwner
func (m *MintageApi) GetTokenInfoListByOwner(owner types.Address) ([]*RpcTokenInfo, error) {
	db, err := getVmDb(m.chain, types.AddressAssert)
	if err != nil {
		return nil, err
	}
	tokenMap, err := abi.GetTokenMapByOwner(db, owner)
	if err != nil {
		return nil, err
	}
	tokenList := make([]*RpcTokenInfo, 0)
	for tokenId, tokenInfo := range tokenMap {
		tokenList = append(tokenList, RawTokenInfoToRpc(tokenInfo, tokenId))
	}
	return checkGenesisToken(db, owner, m.vite.Config().MintageInfo.TokenInfoMap, tokenList)
}

func checkGenesisToken(db vm_db.VmDb, owner types.Address, genesisTokenInfoMap map[string]config.TokenInfo, tokenList []*RpcTokenInfo) ([]*RpcTokenInfo, error) {
	for tidStr, _ := range genesisTokenInfoMap {
		tid, _ := types.HexToTokenTypeId(tidStr)
		info, err := abi.GetTokenByID(db, tid)
		if err != nil {
			return nil, err
		}
		if info != nil && info.Owner == owner {
			tokenList = append(tokenList, RawTokenInfoToRpc(info, tid))
		}
	}
	return tokenList, nil
}
