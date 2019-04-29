package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_db"
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
	Amount         string        `json:"amount"`
}

func (p *PledgeApi) GetAgentPledgeData(param AgentPledgeParam) ([]byte, error) {
	return abi.ABIPledge.PackMethod(abi.MethodNameAgentPledge, param.PledgeAddr, param.BeneficialAddr, param.Bid)
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
}

func (p *PledgeApi) GetPledgeQuota(addr types.Address) (*QuotaAndTxNum, error) {
	q, err := p.chain.GetPledgeQuota(addr)
	if err != nil {
		return nil, err
	}
	return &QuotaAndTxNum{uint64ToString(q.Utps()), uint64ToString(q.Current()), uint64ToString(q.Current() / util.TxGas / util.QuotaRange)}, nil
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
	snapshotBlock := p.chain.GetLatestSnapshotBlock()
	prevHash, err := getPrevBlockHash(p.chain, types.AddressPledge)
	if err != nil {
		return nil, err
	}
	db, err := vm_db.NewVmDb(p.chain, &types.AddressPledge, &snapshotBlock.Hash, prevHash)
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
	for i, info := range list[startHeight:endHeight] {
		targetList[i] = &PledgeInfo{
			*bigIntToString(info.Amount),
			uint64ToString(info.WithdrawHeight),
			info.BeneficialAddr,
			getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
			info.Agent,
			info.AgentAddress,
			info.Bid}
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
