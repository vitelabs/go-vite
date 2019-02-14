package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
)

type MintageApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewMintageApi(vite *vite.Vite) *MintageApi {
	return &MintageApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/mintage_api"),
	}
}

func (m MintageApi) String() string {
	return "MintageApi"
}

type MintageParams struct {
	SelfAddr      types.Address
	Height        uint64
	PrevHash      types.Hash
	SnapshotHash  types.Hash
	TokenName     string
	TokenSymbol   string
	TotalSupply   string
	Decimals      uint8
	IsReIssuable  bool
	MaxSupply     string
	OwnerBurnOnly bool
}

func (m *MintageApi) GetMintageData(param MintageParams) ([]byte, error) {
	tokenId := abi.NewTokenId(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	totalSupply, err := stringToBigInt(&param.TotalSupply)
	if err != nil {
		return nil, err
	}
	return abi.ABIMintage.PackMethod(abi.MethodNameMintage, tokenId, param.TokenName, param.TokenSymbol, totalSupply, param.Decimals)
}
func (m *MintageApi) GetMintageCancelPledgeData(tokenId types.TokenTypeId) ([]byte, error) {
	return abi.ABIMintage.PackMethod(abi.MethodNameMintageCancelPledge, tokenId)
}

func (m *MintageApi) GetMintData(param MintageParams) ([]byte, error) {
	tokenId := abi.NewTokenId(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	totalSupply, err := stringToBigInt(&param.TotalSupply)
	if err != nil {
		return nil, err
	}
	maxSupply, err := stringToBigInt(&param.MaxSupply)
	return abi.ABIMintage.PackMethod(abi.MethodNameMint, param.IsReIssuable, tokenId, param.TokenName, param.TokenSymbol, totalSupply, param.Decimals, maxSupply, param.OwnerBurnOnly)
}

type IssueParams struct {
	TokenId    types.TokenTypeId
	Amount     string
	Beneficial types.Address
}

func (m *MintageApi) GetIssueData(param IssueParams) ([]byte, error) {
	amount, err := stringToBigInt(&param.Amount)
	if err != nil {
		return nil, err
	}
	return abi.ABIMintage.PackMethod(abi.MethodNameIssue, param.TokenId, amount, param.Beneficial)
}
func (m *MintageApi) GetBurnData() ([]byte, error) {
	return abi.ABIMintage.PackMethod(abi.MethodNameBurn)
}

type TransferOwnerParams struct {
	TokenId  types.TokenTypeId
	NewOwner types.Address
}

func (m *MintageApi) GetTransferOwnerData(param TransferOwnerParams) ([]byte, error) {
	return abi.ABIMintage.PackMethod(abi.MethodNameTransferOwner, param.TokenId, param.NewOwner)
}
func (m *MintageApi) GetChangeTokenTypeData(tokenId types.TokenTypeId) ([]byte, error) {
	return abi.ABIMintage.PackMethod(abi.MethodNameChangeTokenType, tokenId)
}
