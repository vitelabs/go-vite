package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
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
	Name                 string        `json:"name"`
	NodeAddr             types.Address `json:"nodeAddr"`
	PledgeAddr           types.Address `json:"pledgeAddr"`
	PledgeAmount         string        `json:"pledgeAmount"`
	WithdrawHeight       string        `json:"withdrawHeight"`
	WithdrawTime         int64         `json:"withdrawTime"`
	CancelHeight         string        `json:"cancelHeight"`
	AvailableRewardOneTx string        `json:"availableRewardOneTx"`
	StartHeight          string        `json:"rewardStartHeight"`
	EndHeight            string        `json:"rewardEndHeight"`
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
		for i, info := range list {
			startHeight, endHeight, availableRewardOneTx, err := contracts.CalcReward(vmContext, info, gid)
			if err != nil {
				return nil, err
			}
			targetList[i] = &RegistrationInfo{
				Name:                 info.Name,
				NodeAddr:             info.NodeAddr,
				PledgeAddr:           info.PledgeAddr,
				PledgeAmount:         *bigIntToString(info.Amount),
				WithdrawHeight:       uint64ToString(info.WithdrawHeight),
				WithdrawTime:         getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
				CancelHeight:         uint64ToString(info.CancelHeight),
				AvailableRewardOneTx: *bigIntToString(availableRewardOneTx), // TODO get available reward by another method
				StartHeight:          uint64ToString(startHeight),
				EndHeight:            uint64ToString(endHeight),
			}
		}
	}
	return targetList, nil
}

type CandidateInfo struct {
	Name     string        `json:"name"`
	NodeAddr types.Address `json:"nodeAddr"`
}

func (r *RegisterApi) GetCandidateList(gid types.Gid) ([]*CandidateInfo, error) {
	list, err := r.chain.GetRegisterList(r.chain.GetLatestSnapshotBlock().Hash, gid)
	if err != nil {
		return nil, err
	}
	targetList := make([]*CandidateInfo, len(list))
	for i, info := range list {
		targetList[i] = &CandidateInfo{info.Name, info.NodeAddr}
	}
	return targetList, nil
}
