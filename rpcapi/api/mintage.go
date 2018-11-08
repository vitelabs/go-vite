package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
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
	SelfAddr     types.Address
	Height       uint64
	PrevHash     types.Hash
	SnapshotHash types.Hash
	TokenName    string
	TokenSymbol  string
	TotalSupply  *big.Int
	Decimals     uint8
}

func (m *MintageApi) GetMintageData(param MintageParams) ([]byte, error) {
	tokenId := abi.NewTokenId(param.SelfAddr, param.Height, param.PrevHash, param.SnapshotHash)
	return abi.ABIMintage.PackMethod(abi.MethodNameMintage, tokenId, param.TokenName, param.TokenSymbol, param.TotalSupply, param.Decimals)
}
func (m *MintageApi) GetMintageCancelPledgeData(tokenId types.TokenTypeId) ([]byte, error) {
	return abi.ABIMintage.PackMethod(abi.MethodNameMintageCancelPledge, tokenId)
}
