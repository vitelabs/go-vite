package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm_context"
)

type RegisterApi struct {
	chain chain.Chain
	log   log15.Logger
}

func NewRegisterApi(vite *vite.Vite) *RegisterApi {
	return &RegisterApi{
		chain: vite.Chain(),
		log:   log15.New("module", "rpc_api/register_api"),
	}
}

func (r RegisterApi) String() string {
	return "RegisterApi"
}

func (r *RegisterApi) GetRegisterData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameRegister, gid, name, nodeAddr)
}
func (r *RegisterApi) GetCancelRegisterData(gid types.Gid, name string) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameCancelRegister, gid, name)
}
func (r *RegisterApi) GetRewardData(gid types.Gid, name string, beneficialAddr types.Address) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameReward, gid, name, beneficialAddr)
}
func (r *RegisterApi) GetUpdateRegistrationData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return contracts.ABIRegister.PackMethod(contracts.MethodNameUpdateRegistration, gid, name, nodeAddr)
}

type RegistrationInfo struct {
	Name                 string        `json:"name"`
	NodeAddr             types.Address `json:"nodeAddr"`
	PledgeAddr           types.Address `json:"pledgeAddr"`
	PledgeAmount         string        `json:"pledgeAmount"`
	WithdrawHeight       string        `json:"withdrawHeight"`
	AvailableReward      string        `json:"availableReward"`
	AvailableRewardOneTx string        `json:"availableRewardOneTx"`
}

func (r *RegisterApi) GetRegistrationList(gid types.Gid, pledgeAddr types.Address) ([]*RegistrationInfo, error) {
	snapshotBlock := r.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(r.chain, &snapshotBlock.Hash, nil, nil)
	if err != nil {
		return nil, err
	}
	list := contracts.GetRegistrationList(vmContext, gid, pledgeAddr)
	targetList := make([]*RegistrationInfo, len(list))
	if len(list) > 0 {
		for i, info := range list {
			targetList[i] = &RegistrationInfo{
				Name: info.Name, NodeAddr: info.NodeAddr, PledgeAddr: info.PledgeAddr, PledgeAmount: *bigIntToString(info.Amount), WithdrawHeight: uint64ToString(info.WithdrawHeight),
			}
			_, availableRewardOneTx := contracts.CalcReward(vmContext, info, false)
			targetList[i].AvailableRewardOneTx = *bigIntToString(availableRewardOneTx)
			_, availableReward := contracts.CalcReward(vmContext, info, true)
			targetList[i].AvailableReward = *bigIntToString(availableReward)
		}
	}
	return targetList, nil
}

type CandidateInfo struct {
	Name     string        `json:"name"`
	NodeAddr types.Address `json:"nodeAddr"`
}

func (r *RegisterApi) GetCandidateList(gid types.Gid) ([]*CandidateInfo, error) {
	list := r.chain.GetRegisterList(r.chain.GetLatestSnapshotBlock().Hash, gid)
	targetList := make([]*CandidateInfo, len(list))
	for i, info := range list {
		targetList[i] = &CandidateInfo{info.Name, info.NodeAddr}
	}
	return targetList, nil
}
