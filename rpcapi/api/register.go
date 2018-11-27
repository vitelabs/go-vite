package api

import (
	"sort"
	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"github.com/vitelabs/go-vite/vm_context"
)

type RegisterApi struct {
	chain chain.Chain
	cs    consensus.Consensus
	log   log15.Logger
}

func NewRegisterApi(vite *vite.Vite) *RegisterApi {
	return &RegisterApi{
		chain: vite.Chain(),
		cs:    vite.Consensus(),
		log:   log15.New("module", "rpc_api/register_api"),
	}
}

func (r RegisterApi) String() string {
	return "RegisterApi"
}

func (r *RegisterApi) GetRegisterData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return abi.ABIRegister.PackMethod(abi.MethodNameRegister, gid, name, nodeAddr)
}
func (r *RegisterApi) GetCancelRegisterData(gid types.Gid, name string) ([]byte, error) {
	return abi.ABIRegister.PackMethod(abi.MethodNameCancelRegister, gid, name)
}
func (r *RegisterApi) GetRewardData(gid types.Gid, name string, beneficialAddr types.Address) ([]byte, error) {
	return abi.ABIRegister.PackMethod(abi.MethodNameReward, gid, name, beneficialAddr)
}
func (r *RegisterApi) GetUpdateRegistrationData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return abi.ABIRegister.PackMethod(abi.MethodNameUpdateRegistration, gid, name, nodeAddr)
}

type RegistrationInfo struct {
	Name           string        `json:"name"`
	NodeAddr       types.Address `json:"nodeAddr"`
	PledgeAddr     types.Address `json:"pledgeAddr"`
	PledgeAmount   string        `json:"pledgeAmount"`
	WithdrawHeight string        `json:"withdrawHeight"`
	WithdrawTime   int64         `json:"withdrawTime"`
	CancelHeight   string        `json:"cancelHeight"`
}

type byRegistrationWithdrawHeight []*types.Registration

func (a byRegistrationWithdrawHeight) Len() int      { return len(a) }
func (a byRegistrationWithdrawHeight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byRegistrationWithdrawHeight) Less(i, j int) bool {
	if a[i].WithdrawHeight == a[j].WithdrawHeight {
		return a[i].CancelHeight > a[j].CancelHeight
	}
	return a[i].WithdrawHeight > a[j].WithdrawHeight
}

func (r *RegisterApi) GetRegistrationList(gid types.Gid, pledgeAddr types.Address) ([]*RegistrationInfo, error) {
	snapshotBlock := r.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(r.chain, &snapshotBlock.Hash, nil, nil)
	if err != nil {
		return nil, err
	}
	list := abi.GetRegistrationList(vmContext, gid, pledgeAddr)
	targetList := make([]*RegistrationInfo, len(list))
	if len(list) > 0 {
		sort.Sort(byRegistrationWithdrawHeight(list))
		for i, info := range list {
			targetList[i] = &RegistrationInfo{
				Name:           info.Name,
				NodeAddr:       info.NodeAddr,
				PledgeAddr:     info.PledgeAddr,
				PledgeAmount:   *bigIntToString(info.Amount),
				WithdrawHeight: uint64ToString(info.WithdrawHeight),
				WithdrawTime:   getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
				CancelHeight:   uint64ToString(info.CancelHeight),
			}
		}
	}
	return targetList, nil
}

func (r *RegisterApi) GetRegistration(name string, gid types.Gid) (*types.Registration, error) {
	vmContext, err := vm_context.NewVmContext(r.chain, nil, nil, &types.AddressRegister)
	if err != nil {
		return nil, err
	}
	return abi.GetRegistration(vmContext, gid, name), nil
}

// Deprecated: Use GetRegistration instead
func (r *RegisterApi) GetRegisterPledgeAddr(name string, gid *types.Gid) (*types.Address, error) {
	var g types.Gid
	if gid == nil || *gid == types.DELEGATE_GID {
		g = types.SNAPSHOT_GID
	} else {
		g = *gid
	}
	registration, err := r.GetRegistration(name, g)
	if err != nil {
		return nil, err
	}
	if registration != nil {
		return &registration.PledgeAddr, nil
	}
	return nil, nil
}

type RegistParam struct {
	Name string     `json:"name"`
	Gid  *types.Gid `json:"gid"`
}

func (r *RegisterApi) GetRegisterPledgeAddrList(paramList []*RegistParam) ([]*types.Address, error) {
	if len(paramList) == 0 {
		return nil, nil
	}
	addrList := make([]*types.Address, len(paramList))
	vmContext, err := vm_context.NewVmContext(r.chain, nil, nil, &types.AddressRegister)
	if err != nil {
		return nil, err
	}
	for k, v := range paramList {
		var r *types.Registration
		if v.Gid == nil || *v.Gid == types.DELEGATE_GID {
			r = abi.GetRegistration(vmContext, types.SNAPSHOT_GID, v.Name)
		} else {
			r = abi.GetRegistration(vmContext, *v.Gid, v.Name)
		}
		if r != nil {
			addrList[k] = &r.PledgeAddr
		}
	}
	return addrList, nil
}

type CandidateInfo struct {
	Name     string        `json:"name"`
	NodeAddr types.Address `json:"nodeAddr"`
	VoteNum  string        `json:"voteNum"`
}

func (r *RegisterApi) GetCandidateList(gid types.Gid) ([]*CandidateInfo, error) {
	head := r.chain.GetLatestSnapshotBlock()
	details, _, err := r.cs.ReadVoteMapForAPI(gid, (*head.Timestamp).Add(time.Second))
	if err != nil {
		return nil, err
	}
	var result []*CandidateInfo
	for _, v := range details {
		result = append(result, &CandidateInfo{v.Name, v.CurrentAddr, *bigIntToString(v.Balance)})
	}
	return result, nil
}
