package api

import (
	"github.com/vitelabs/go-vite/common/types"
	ledger "github.com/vitelabs/go-vite/interfaces/core"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/quota"
)

type QuotaApi struct {
	chain     chain.Chain
	log       log15.Logger
	ledgerApi *LedgerApi
}

func NewQuotaApi(vite *vite.Vite) *QuotaApi {
	return &QuotaApi{
		chain:     vite.Chain(),
		log:       log15.New("module", "rpc_api/quota_api"),
		ledgerApi: NewLedgerApi(vite),
	}
}

func (p QuotaApi) String() string {
	return "QuotaApi"
}

// Private
func (p *QuotaApi) GetPledgeData(beneficialAddr types.Address) ([]byte, error) {
	return abi.ABIQuota.PackMethod(abi.MethodNameStake, beneficialAddr)
}

// Private
func (p *QuotaApi) GetCancelPledgeData(beneficialAddr types.Address, amount string) ([]byte, error) {
	if bAmount, err := stringToBigInt(&amount); err == nil {
		return abi.ABIQuota.PackMethod(abi.MethodNameCancelStake, beneficialAddr, bAmount)
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
func (p *QuotaApi) GetAgentPledgeData(param AgentPledgeParam) ([]byte, error) {
	stakeHeight, err := StringToUint64(param.StakeHeight)
	if err != nil {
		return nil, err
	}
	return abi.ABIQuota.PackMethod(abi.MethodNameDelegateStake, param.PledgeAddr, param.BeneficialAddr, param.Bid, stakeHeight)
}

// Private
func (p *QuotaApi) GetAgentCancelPledgeData(param AgentPledgeParam) ([]byte, error) {
	if bAmount, err := stringToBigInt(&param.Amount); err == nil {
		return abi.ABIQuota.PackMethod(abi.MethodNameCancelDelegateStake, param.PledgeAddr, param.BeneficialAddr, bAmount, param.Bid)
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
	Id             *types.Hash   `json:"id"`
}

func NewPledgeInfo(info *types.StakeInfo, snapshotBlock *ledger.SnapshotBlock) *PledgeInfo {
	return &PledgeInfo{
		*bigIntToString(info.Amount),
		info.Beneficiary,
		Uint64ToString(info.ExpirationHeight),
		getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.ExpirationHeight),
		info.IsDelegated,
		info.DelegateAddress,
		info.Bid,
		info.Id}
}

type byExpirationHeight []*types.StakeInfo

func (a byExpirationHeight) Len() int      { return len(a) }
func (a byExpirationHeight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byExpirationHeight) Less(i, j int) bool {
	if a[i].ExpirationHeight == a[j].ExpirationHeight {
		return a[i].Beneficiary.String() < a[j].Beneficiary.String()
	}
	return a[i].ExpirationHeight < a[j].ExpirationHeight
}

// Private
func (p *QuotaApi) GetQuotaUsedList(addr types.Address) ([]types.QuotaInfo, error) {
	db, err := getVmDb(p.chain, types.AddressQuota)
	if err != nil {
		return nil, err
	}
	return db.GetQuotaUsedList(addr), nil
}

type PledgeQueryParams struct {
	PledgeAddr     types.Address `json:"pledgeAddr"`
	AgentAddr      types.Address `json:"agentAddr"`
	BeneficialAddr types.Address `json:"beneficialAddr"`
	Bid            uint8         `json:"bid"`
}

type QuotaCoefficientInfo struct {
	Qc           *string `json:"qc"`
	GlobalQuota  string  `json:"globalQuota"`
	GlobalUt     string  `json:"globalUtPerSecond"`
	IsCongestion bool    `json:"isCongestion"`
}

// Private
func (p *QuotaApi) GetQuotaCoefficient() (*QuotaCoefficientInfo, error) {
	qc, globalQuota, isCongestion := quota.CalcQc(p.chain, p.chain.GetLatestSnapshotBlock().Height)
	return &QuotaCoefficientInfo{bigIntToString(qc), Uint64ToString(globalQuota), Float64ToString(float64(globalQuota)/21000/74, 2), isCongestion}, nil
}
