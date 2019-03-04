package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm/util"
	"github.com/vitelabs/go-vite/vm_context"
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

type QuotaAndTxNum struct {
	Quota string `json:"quota"`
	TxNum string `json:"txNum"`
}

func (p *PledgeApi) GetPledgeQuota(addr types.Address) (*QuotaAndTxNum, error) {
	hash, err := p.ledgerApi.GetFittestSnapshotHash(&addr, nil)
	if err != nil {
		return nil, err
	}
	q, err := p.chain.GetPledgeQuota(*hash, addr)
	if err != nil {
		return nil, err
	}
	return &QuotaAndTxNum{uint64ToString(q), uint64ToString(q / util.TxGas)}, nil
}

type PledgeInfoList struct {
	TotalPledgeAmount string        `json:"totalPledgeAmount"`
	Count             int           `json:"totalCount"`
	List              []*PledgeInfo `json:"pledgeInfoList"`
}
type PledgeInfo struct {
	Amount         string         `json:"amount"`
	WithdrawHeight *string        `json:"withdrawHeight"`
	BeneficialAddr *types.Address `json:"beneficialAddr"`
	WithdrawTime   *int64         `json:"withdrawTime"`
	PledgeAddr     *types.Address `json:"pledgeAddr"`
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
	vmContext, err := vm_context.NewVmContext(p.chain, &snapshotBlock.Hash, nil, nil)
	if err != nil {
		return nil, err
	}
	list, amount := abi.GetPledgeInfoList(vmContext, addr)
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
		withdrawHeight := uint64ToString(info.WithdrawHeight)
		withdrawTime := getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight)
		targetList[i] = &PledgeInfo{
			Amount:         *bigIntToString(info.Amount),
			WithdrawHeight: &withdrawHeight,
			BeneficialAddr: &info.BeneficialAddr,
			WithdrawTime:   &withdrawTime}
	}
	return &PledgeInfoList{*bigIntToString(amount), len(list), targetList}, nil
}

func (p *PledgeApi) GetPledgeInfoListByBeneficial(beneficial types.Address, snapshotHash *types.Hash) ([]*PledgeInfo, error) {
	vmContext, err := vm_context.NewVmContext(p.chain, snapshotHash, nil, nil)
	if err != nil {
		return nil, err
	}
	list := abi.GetPledgeInfoListByBeneficial(vmContext, beneficial, snapshotHash)
	pledgeInfoList := make([]*PledgeInfo, len(list))
	for i, info := range list {
		pledgeInfoList[i] = &PledgeInfo{Amount: *bigIntToString(info.Amount), PledgeAddr: info.PledgeAddr}
	}
	return pledgeInfoList, nil
}
