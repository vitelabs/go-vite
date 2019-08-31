package api

import (
	"encoding/hex"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/quota"
	"github.com/vitelabs/go-vite/vm/util"
	"sort"
)

type PledgeApi struct {
	chain     chain.Chain
	log       log15.Logger
	ledgerApi *LedgerApi
}

func NewPledgeApi(vite *vite.Vite) *PledgeApi {
	return &PledgeApi{
		chain:     vite.Chain(),
		log:       log15.New("module", "rpc_api/pledge_api"),
		ledgerApi: NewLedgerApi(vite),
	}
}

func (p PledgeApi) String() string {
	return "PledgeApi"
}

// Private
func (p *PledgeApi) GetPledgeData(beneficialAddr types.Address) ([]byte, error) {
	return abi.ABIPledge.PackMethod(abi.MethodNamePledge, beneficialAddr)
}

// Private
func (p *PledgeApi) GetCancelPledgeData(beneficialAddr types.Address, amount string) ([]byte, error) {
	if bAmount, err := stringToBigInt(&amount); err == nil {
		return abi.ABIPledge.PackMethod(abi.MethodNameCancelPledge, beneficialAddr, bAmount)
	} else {
		return nil, err
	}
}

type AgentPledgeParam struct {
	PledgeAddr     types.Address `json:"pledgeAddr"`
	BeneficialAddr types.Address `json:"beneficialAddr"`
	Bid            uint8         `json:"bid"`
	StakeHeight    string        `json:"stakeHeight"`
	Amount         string        `json:"amount"`
}

// Private
func (p *PledgeApi) GetAgentPledgeData(param AgentPledgeParam) ([]byte, error) {
	stakeHeight, err := StringToUint64(param.StakeHeight)
	if err != nil {
		return nil, err
	}
	return abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, param.PledgeAddr, param.BeneficialAddr, param.Bid, stakeHeight)
}

// Private
func (p *PledgeApi) GetAgentCancelPledgeData(param AgentPledgeParam) ([]byte, error) {
	if bAmount, err := stringToBigInt(&param.Amount); err == nil {
		return abi.ABIPledge.PackMethod(abi.MethodNameAgentCancelPledge, param.PledgeAddr, param.BeneficialAddr, bAmount, param.Bid)
	} else {
		return nil, err
	}
}

type QuotaAndTxNum struct {
	QuotaPerSnapshotBlock string `json:"quotaPerSnapshotBlock"` // Deprecated
	CurrentQuota          string `json:"current"`               // Deprecated
	CurrentTxNumPerSec    string `json:"utps"`                  // Deprecated: use currentUt field instead
	CurrentUt             string `json:"currentUt"`
	Utpe                  string `json:"utpe"`
	PledgeAmount          string `json:"pledgeAmount"`
}

type QuotaInfo struct {
	CurrentUt     string  `json:"currentUt"`
	Utpe          string  `json:"utpe"`
	StakingAmount *string `json:"stakingAmount"`
}

// Deprecated: use contract_GetQuota instead
func (p *PledgeApi) GetPledgeQuota(addr types.Address) (*QuotaAndTxNum, error) {
	amount, q, err := p.chain.GetPledgeQuota(addr)
	if err != nil {
		return nil, err
	}
	return &QuotaAndTxNum{
		QuotaPerSnapshotBlock: Uint64ToString(q.PledgeQuotaPerSnapshotBlock()),
		CurrentQuota:          Uint64ToString(q.Current()),
		CurrentTxNumPerSec:    Uint64ToString(q.Current() / quota.QuotaForUtps),
		CurrentUt:             Float64ToString(float64(q.Current())/float64(quota.QuotaForUtps), 4),
		Utpe:                  Float64ToString(float64(q.PledgeQuotaPerSnapshotBlock()*util.OneRound)/float64(quota.QuotaForUtps), 4),
		PledgeAmount:          *bigIntToString(amount)}, nil
}

func (p *ContractApi) GetQuota(addr types.Address) (*QuotaInfo, error) {
	amount, q, err := p.chain.GetPledgeQuota(addr)
	if err != nil {
		return nil, err
	}
	return &QuotaInfo{
		CurrentUt:     Float64ToString(float64(q.Current())/float64(quota.QuotaForUtps), 4),
		Utpe:          Float64ToString(float64(q.PledgeQuotaPerSnapshotBlock()*util.OneRound)/float64(quota.QuotaForUtps), 4),
		StakingAmount: bigIntToString(amount)}, nil
}

type PledgeInfoList struct {
	TotalPledgeAmount string        `json:"totalPledgeAmount"`
	Count             int           `json:"totalCount"`
	List              []*PledgeInfo `json:"pledgeInfoList"`
}
type PledgeInfo struct {
	Amount         string        `json:"amount"`
	BeneficialAddr types.Address `json:"beneficialAddr"`
	WithdrawHeight string        `json:"withdrawHeight"`
	WithdrawTime   int64         `json:"withdrawTime"`
	Agent          bool          `json:"agent"`
	AgentAddress   types.Address `json:"agentAddress"`
	Bid            uint8         `json:"bid"`
}

func NewPledgeInfo(info *types.PledgeInfo, snapshotBlock *ledger.SnapshotBlock) *PledgeInfo {
	return &PledgeInfo{
		*bigIntToString(info.Amount),
		info.BeneficialAddr,
		Uint64ToString(info.WithdrawHeight),
		getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
		info.Agent,
		info.AgentAddress,
		info.Bid}
}

type byWithdrawHeight []*types.PledgeInfo

func (a byWithdrawHeight) Len() int      { return len(a) }
func (a byWithdrawHeight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byWithdrawHeight) Less(i, j int) bool {
	if a[i].WithdrawHeight == a[j].WithdrawHeight {
		return a[i].BeneficialAddr.String() < a[j].BeneficialAddr.String()
	}
	return a[i].WithdrawHeight < a[j].WithdrawHeight
}

// Deprecated: use contract_getStakingList instead
func (p *PledgeApi) GetPledgeList(addr types.Address, index int, count int) (*PledgeInfoList, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	list, amount, err := abi.GetPledgeInfoList(db, addr)
	if err != nil {
		return nil, err
	}
	sort.Sort(byWithdrawHeight(list))
	startHeight, endHeight := index*count, (index+1)*count
	if startHeight >= len(list) {
		return &PledgeInfoList{TotalPledgeAmount: *bigIntToString(amount), Count: len(list), List: []*PledgeInfo{}}, nil
	}
	if endHeight > len(list) {
		endHeight = len(list)
	}
	targetList := make([]*PledgeInfo, endHeight-startHeight)
	snapshotBlock, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	for i, info := range list[startHeight:endHeight] {
		targetList[i] = NewPledgeInfo(info, snapshotBlock)
	}
	return &PledgeInfoList{*bigIntToString(amount), len(list), targetList}, nil
}

type StakingInfoList struct {
	StakingAmount string         `json:"stakingAmount"`
	Count         int            `json:"totalCount"`
	StakingList   []*StakingInfo `json:"stakingList"`
}

type StakingInfo struct {
	Amount           string        `json:"amount"`
	Beneficiary      types.Address `json:"beneficiary"`
	ExpirationHeight string        `json:"expirationHeight"`
	ExpirationTime   int64         `json:"expirationTime"`
	IsDelegated      bool          `json:"isDelegated"`
	DelegateAddress  types.Address `json:"delegateAddress"`
	SkatingAddress   types.Address `json:"stakingAddress"`
	Bid              uint8         `json:"bid"`
}

func NewStakingInfo(addr types.Address, info *types.PledgeInfo, snapshotBlock *ledger.SnapshotBlock) *StakingInfo {
	return &StakingInfo{
		*bigIntToString(info.Amount),
		info.BeneficialAddr,
		Uint64ToString(info.WithdrawHeight),
		getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
		info.Agent,
		info.AgentAddress,
		addr,
		info.Bid}
}

func (p *ContractApi) GetStakingList(addr types.Address, index int, count int) (*StakingInfoList, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	list, amount, err := abi.GetPledgeInfoList(db, addr)
	if err != nil {
		return nil, err
	}
	sort.Sort(byWithdrawHeight(list))
	startHeight, endHeight := index*count, (index+1)*count
	if startHeight >= len(list) {
		return &StakingInfoList{*bigIntToString(amount), len(list), []*StakingInfo{}}, nil
	}
	if endHeight > len(list) {
		endHeight = len(list)
	}
	targetList := make([]*StakingInfo, endHeight-startHeight)
	snapshotBlock, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	for i, info := range list[startHeight:endHeight] {
		targetList[i] = NewStakingInfo(addr, info, snapshotBlock)
	}
	return &StakingInfoList{*bigIntToString(amount), len(list), targetList}, nil
}

type GetStakingListByPageResult struct {
	StakingInfoList []*StakingInfo `json:"list"`
	LastKey         string         `json:"lastKey"`
}

func (p *ContractApi) GetStakingListByPage(snapshotHash types.Hash, lastKey string, count uint64) (*GetStakingListByPageResult, error) {
	lastKeyBytes, err := hex.DecodeString(lastKey)
	if err != nil {
		return nil, err
	}
	list, lastKeyBytes, err := p.chain.GetPledgeListByPage(snapshotHash, lastKeyBytes, count)
	if err != nil {
		return nil, err
	}
	targetList := make([]*StakingInfo, len(list))
	snapshotBlock := p.chain.GetLatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	for i, info := range list {
		targetList[i] = NewStakingInfo(info.PledgeAddress, info, snapshotBlock)
	}
	return &GetStakingListByPageResult{targetList, hex.EncodeToString(lastKeyBytes)}, nil
}

// Deprecated: use contract_getQuota instead
func (p *PledgeApi) GetPledgeBeneficialAmount(addr types.Address) (string, error) {
	amount, err := p.chain.GetPledgeBeneficialAmount(addr)
	if err != nil {
		return "", err
	}
	return *bigIntToString(amount), nil
}

// Private
func (p *PledgeApi) GetQuotaUsedList(addr types.Address) ([]types.QuotaInfo, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	return db.GetQuotaUsedList(addr), nil
}

// Deprecated: use contract_getStakingAmountByUtps instead
func (p *PledgeApi) GetPledgeAmountByUtps(utps string) (*string, error) {
	utpfF, err := StringToFloat64(utps)
	if err != nil {
		return nil, err
	}
	amount, err := quota.CalcPledgeAmountByUtps(utpfF)
	if err != nil {
		return nil, err
	}
	return bigIntToString(amount), nil
}

func (p *ContractApi) GetStakingAmountByUtps(utps string) (*string, error) {
	utpfF, err := StringToFloat64(utps)
	if err != nil {
		return nil, err
	}
	amount, err := quota.CalcPledgeAmountByUtps(utpfF)
	if err != nil {
		return nil, err
	}
	return bigIntToString(amount), nil
}

type PledgeQueryParams struct {
	PledgeAddr     types.Address `json:"pledgeAddr"`
	AgentAddr      types.Address `json:"agentAddr"`
	BeneficialAddr types.Address `json:"beneficialAddr"`
	Bid            uint8         `json:"bid"`
}

// Deprecated
func (p *PledgeApi) GetAgentPledgeInfo(params PledgeQueryParams) (*PledgeInfo, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	snapshotBlock, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	info, err := abi.GetPledgeInfo(db, params.PledgeAddr, params.BeneficialAddr, params.AgentAddr, true, params.Bid)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return NewPledgeInfo(info, snapshotBlock), nil
}

type StakeQueryParams struct {
	PledgeAddr     types.Address `json:"stakingAddress"`
	AgentAddr      types.Address `json:"delegateAddress"`
	BeneficialAddr types.Address `json:"beneficiary"`
	Bid            uint8         `json:"bid"`
}

func (p *ContractApi) GetDelegatedStakingInfo(params StakeQueryParams) (*StakingInfo, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	snapshotBlock, err := db.LatestSnapshotBlock()
	if err != nil {
		return nil, err
	}
	info, err := abi.GetPledgeInfo(db, params.PledgeAddr, params.BeneficialAddr, params.AgentAddr, true, params.Bid)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}
	return NewStakingInfo(params.PledgeAddr, info, snapshotBlock), nil
}

type QuotaCoefficientInfo struct {
	Qc           *string `json:"qc"`
	GlobalQuota  string  `json:"globalQuota"`
	GlobalUt     string  `json:"globalUtPerSecond"`
	IsCongestion bool    `json:"isCongestion"`
}

// Private
func (p *PledgeApi) GetQuotaCoefficient() (*QuotaCoefficientInfo, error) {
	qc, globalQuota, isCongestion := quota.CalcQc(p.chain, p.chain.GetLatestSnapshotBlock().Height)
	return &QuotaCoefficientInfo{bigIntToString(qc), Uint64ToString(globalQuota), Float64ToString(float64(globalQuota)/21000/74, 2), isCongestion}, nil
}
