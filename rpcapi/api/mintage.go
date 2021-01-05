package api

import (
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/common/config"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
)

type MintageAPI struct {
	chain chain.Chain
	vite  *vite.Vite
	log   log15.Logger
}

// NewMintageAPI mintageApi constructor
func NewMintageAPI(vite *vite.Vite) interface{} {
	return &MintageAPI{
		chain: vite.Chain(),
		vite:  vite,
		log:   log15.New("module", "rpc_api/mintage"),
	}
}

func (m MintageAPI) String() string {
	return "MintageApi"
}

// MintageParams params for GetMintData
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
func (m MintageAPI) GetMintData(param MintageParams) ([]byte, error) {
	totalSupply, err := stringToBigInt(&param.TotalSupply)
	if err != nil {
		return nil, err
	}
	maxSupply, err := stringToBigInt(&param.MaxSupply)
	if err != nil {
		return nil, err
	}
	return abi.ABIAsset.PackMethod(abi.MethodNameIssue, param.IsReIssuable, param.TokenName, param.TokenSymbol, totalSupply, param.Decimals, maxSupply, param.OwnerBurnOnly)
}

type IssueParams struct {
	TokenId    types.TokenTypeId
	Amount     string
	Beneficial types.Address
}

// Private
func (m MintageAPI) GetIssueData(param IssueParams) ([]byte, error) {
	amount, err := stringToBigInt(&param.Amount)
	if err != nil {
		return nil, err
	}
	return abi.ABIAsset.PackMethod(abi.MethodNameReIssue, param.TokenId, amount, param.Beneficial)
}

// Private
func (m MintageAPI) GetBurnData() ([]byte, error) {
	return abi.ABIAsset.PackMethod(abi.MethodNameBurn)
}

type TransferOwnerParams struct {
	TokenId  types.TokenTypeId
	NewOwner types.Address
}

// Private
func (m MintageAPI) GetTransferOwnerData(param TransferOwnerParams) ([]byte, error) {
	return abi.ABIAsset.PackMethod(abi.MethodNameTransferOwnership, param.TokenId, param.NewOwner)
}

// Private
func (m MintageAPI) GetChangeTokenTypeData(tokenId types.TokenTypeId) ([]byte, error) {
	return abi.ABIAsset.PackMethod(abi.MethodNameDisableReIssue, tokenId)
}

func checkGenesisToken(db interfaces.VmDb, owner types.Address, genesisTokenInfoMap map[string]*config.TokenInfo, tokenList []*RpcTokenInfo) ([]*RpcTokenInfo, error) {
	for tidStr := range genesisTokenInfoMap {
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
