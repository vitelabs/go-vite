package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_db"
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

type NewTokenIdParams struct {
	SelfAddr types.Address
	Height   string
	PrevHash types.Hash
}

func (m *MintageApi) NewTokenId(param NewTokenIdParams) (*types.TokenTypeId, error) {
	h, err := StringToUint64(param.Height)
	if err != nil {
		return nil, err
	}
	tid := abi.NewTokenId(param.SelfAddr, h, param.PrevHash)
	return &tid, nil
}

type MintageParams struct {
	SelfAddr      types.Address
	Height        string
	PrevHash      types.Hash
	TokenName     string
	TokenSymbol   string
	TotalSupply   string
	Decimals      uint8
	IsReIssuable  bool
	MaxSupply     string
	OwnerBurnOnly bool
}

func (m *MintageApi) GetMintageCancelPledgeData(tokenId types.TokenTypeId) ([]byte, error) {
	return abi.ABIMintage.PackMethod(abi.MethodNameCancelMintPledge, tokenId)
}

func (m *MintageApi) GetMintData(param MintageParams) ([]byte, error) {
	h, err := StringToUint64(param.Height)
	if err != nil {
		return nil, err
	}
	tokenId := abi.NewTokenId(param.SelfAddr, h, param.PrevHash)
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
	// TODO tmpchain
	var tmpChain vm_db.Chain
	prevHash, err := getPrevBlockHash(m.chain, &types.AddressMintage)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(tmpChain, &types.AddressMintage, &snapshotBlock.Hash, prevHash)
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

func (m *MintageApi) GetTokenInfoById(tokenId types.TokenTypeId) (*RpcTokenInfo, error) {
	snapshotBlock := m.chain.GetLatestSnapshotBlock()
	// TODO tmpchain
	var tmpChain vm_db.Chain
	prevHash, err := getPrevBlockHash(m.chain, &types.AddressMintage)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(tmpChain, &types.AddressMintage, &snapshotBlock.Hash, prevHash)
	if err != nil {
		return nil, err
	}
	tokenInfo, err := abi.GetTokenById(db, tokenId)
	if err != nil {
		return nil, err
	}
	if tokenInfo != nil {
		return RawTokenInfoToRpc(tokenInfo, tokenId), nil
	}
	return nil, nil
}

func (m *MintageApi) GetTokenInfoListByOwner(owner types.Address) ([]*RpcTokenInfo, error) {
	snapshotBlock := m.chain.GetLatestSnapshotBlock()
	// TODO tmpchain
	var tmpChain vm_db.Chain
	prevHash, err := getPrevBlockHash(m.chain, &types.AddressMintage)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(tmpChain, &types.AddressMintage, &snapshotBlock.Hash, prevHash)
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
	return tokenList, nil
}
