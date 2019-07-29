package api

import (
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

func (p *PledgeApi) GetPledgeData(beneficialAddr types.Address) ([]byte, error) {
	return abi.ABIPledge.PackMethod(abi.MethodNamePledge, beneficialAddr)
}

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

func (p *PledgeApi) GetAgentPledgeData(param AgentPledgeParam) ([]byte, error) {
	stakeHeight, err := StringToUint64(param.StakeHeight)
	if err != nil {
		return nil, err
	}
	return abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, param.PledgeAddr, param.BeneficialAddr, param.Bid, stakeHeight)
}

func (p *PledgeApi) GetAgentCancelPledgeData(param AgentPledgeParam) ([]byte, error) {
	if bAmount, err := stringToBigInt(&param.Amount); err == nil {
		return abi.ABIPledge.PackMethod(abi.MethodNameAgentCancelPledge, param.PledgeAddr, param.BeneficialAddr, bAmount, param.Bid)
	} else {
		return nil, err
	}
}

type QuotaAndTxNum struct {
	QuotaPerSnapshotBlock string `json:"quotaPerSnapshotBlock"`
	CurrentQuota          string `json:"current"`
	CurrentTxNumPerSec    string `json:"utps"`
	CurrentTxNum          string `json:"currentUt"`
	TxNum                 string `json:"utpe"`
	PledgeAmount          string `json:"pledgeAmount"`
}

func (p *PledgeApi) GetPledgeQuota(addr types.Address) (*QuotaAndTxNum, error) {
	q, err := p.chain.GetPledgeQuota(addr)
	if err != nil {
		return nil, err
	}
	amount, err := p.chain.GetPledgeBeneficialAmount(addr)
	if err != nil {
		return nil, err
	}
	return &QuotaAndTxNum{Uint64ToString(q.PledgeQuotaPerSnapshotBlock()), Uint64ToString(q.Current()), Uint64ToString(q.Current() / quota.QuotaForUtps), Float64ToString(float64(q.Current())/float64(quota.QuotaForUtps), 4), Float64ToString(float64(q.PledgeQuotaPerSnapshotBlock()*util.OneRound)/float64(quota.QuotaForUtps), 4), *bigIntToString(amount)}, nil
}

type PledgeInfoList struct {
	TotalPledgeAmount string        `json:"totalPledgeAmount"`
	Count             int           `json:"totalCount"`
	List              []*PledgeInfo `json:"pledgeInfoList"`
}
type PledgeInfo struct {
	Amount         string        `json:"amount"`
	WithdrawHeight string        `json:"withdrawHeight"`
	BeneficialAddr types.Address `json:"beneficialAddr"`
	WithdrawTime   int64         `json:"withdrawTime"`
	Agent          bool          `json:"agent"`
	AgentAddress   types.Address `json:"agentAddress"`
	Bid            uint8         `json:"bid"`
}

func NewPledgeInfo(info *abi.PledgeInfo, snapshotBlock *ledger.SnapshotBlock) *PledgeInfo {
	return &PledgeInfo{
		*bigIntToString(info.Amount),
		Uint64ToString(info.WithdrawHeight),
		info.BeneficialAddr,
		getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
		info.Agent,
		info.AgentAddress,
		info.Bid}
}

type byWithdrawHeight []*abi.PledgeInfo

func (a byWithdrawHeight) Len() int      { return len(a) }
func (a byWithdrawHeight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byWithdrawHeight) Less(i, j int) bool {
	if a[i].WithdrawHeight == a[j].WithdrawHeight {
		return a[i].BeneficialAddr.String() < a[j].BeneficialAddr.String()
	}
	return a[i].WithdrawHeight < a[j].WithdrawHeight
}

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
		return &PledgeInfoList{*bigIntToString(amount), len(list), []*PledgeInfo{}}, nil
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

func (p *PledgeApi) GetPledgeBeneficialAmount(addr types.Address) (string, error) {
	amount, err := p.chain.GetPledgeBeneficialAmount(addr)
	if err != nil {
		return "", err
	}
	return *bigIntToString(amount), nil
}

func (p *PledgeApi) GetQuotaUsedList(addr types.Address) ([]types.QuotaInfo, error) {
	db, err := getVmDb(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	return db.GetQuotaUsedList(addr), nil
}

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

type PledgeQueryParams struct {
	PledgeAddr     types.Address `json:"pledgeAddr"`
	AgentAddr      types.Address `json:"agentAddr"`
	BeneficialAddr types.Address `json:"beneficialAddr"`
	Bid            uint8         `json:"bid"`
}

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

type QuotaCoefficientInfo struct {
	Qc           *string `json:"qc"`
	GlobalQuota  string  `json:"globalQuota"`
	IsCongestion bool    `json:"isCongestion"`
}

func (p *PledgeApi) GetQuotaCoefficient() (*QuotaCoefficientInfo, error) {
	qc, globalQuota, isCongestion := quota.CalcQc(p.chain, p.chain.GetLatestSnapshotBlock().Height)
	return &QuotaCoefficientInfo{bigIntToString(qc), Uint64ToString(globalQuota), isCongestion}, nil
}
