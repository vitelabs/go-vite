package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_context"
	"sort"
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
	if err != nil {
		return nil, err
	}
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

type TokenInfoList struct {
	Count int             `json:"totalCount"`
	List  []*RpcTokenInfo `json:"tokenInfoList"`
}

type byName []*RpcTokenInfo

func (a byName) Len() int      { return len(a) }
func (a byName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byName) Less(i, j int) bool {
	if a[i].TokenName == a[j].TokenName {
		return a[i].TokenId.String() < a[j].TokenId.String()
	}
	return a[i].TokenName < a[j].TokenName
}

func (m *MintageApi) GetTokenInfoList(index int, count int) (*TokenInfoList, error) {
	snapshotBlock := m.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(m.chain, &snapshotBlock.Hash, nil, nil)
	if err != nil {
		return nil, err
	}
	tokenMap := abi.GetTokenMap(vmContext)
	listLen := len(tokenMap)
	tokenList := make([]*RpcTokenInfo, 0)
	for tokenId, tokenInfo := range tokenMap {
		tokenList = append(tokenList, RawTokenInfoToRpc(tokenInfo, tokenId))
	}
	sort.Sort(byName(tokenList))
	start, end := getRange(index, count, listLen)
	return &TokenInfoList{listLen, tokenList[start:end]}, nil
}

func (m *MintageApi) GetTokenInfoById(tokenId types.TokenTypeId) (*RpcTokenInfo, error) {
	snapshotBlock := m.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(m.chain, &snapshotBlock.Hash, nil, nil)
	if err != nil {
		return nil, err
	}
	tokenInfo := abi.GetTokenById(vmContext, tokenId)
	if tokenInfo != nil {
		return RawTokenInfoToRpc(tokenInfo, tokenId), nil
	}
	return nil, nil
}
func (m *MintageApi) GetTokenInfoListByOwner(owner types.Address) ([]*RpcTokenInfo, error) {
	snapshotBlock := m.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(m.chain, &snapshotBlock.Hash, nil, nil)
	if err != nil {
		return nil, err
	}
	tokenMap := abi.GetTokenMapByOwner(vmContext, owner)
	tokenList := make([]*RpcTokenInfo, 0)
	for tokenId, tokenInfo := range tokenMap {
		tokenList = append(tokenList, RawTokenInfoToRpc(tokenInfo, tokenId))
	}
	return tokenList, nil
}
