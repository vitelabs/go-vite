package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
	"github.com/vitelabs/go-vite/vm/contracts"
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
	MinRewardTime  int64         `json:"minRewardTime"`
}

type RewardInfo struct {
	Reward    string `json:"reward"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
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
		genesisTime := vmContext.GetGenesisSnapshotBlock().Timestamp
		groupInfo := abi.GetConsensusGroup(vmContext, gid)
		reader := core.NewReader(*genesisTime, groupInfo)
		periodTime, err := reader.PeriodTime()
		if err != nil {
			return nil, err
		}
		genesisTimeInt := genesisTime.Unix()

		for i, info := range list {
			targetList[i] = &RegistrationInfo{
				Name:           info.Name,
				NodeAddr:       info.NodeAddr,
				PledgeAddr:     info.PledgeAddr,
				PledgeAmount:   *bigIntToString(info.Amount),
				WithdrawHeight: uint64ToString(info.WithdrawHeight),
				WithdrawTime:   getWithdrawTime(snapshotBlock.Timestamp, snapshotBlock.Height, info.WithdrawHeight),
				CancelHeight:   uint64ToString(info.CancelHeight),
				MinRewardTime:  contracts.CalcMinRewardTime(info, genesisTimeInt, periodTime),
			}
		}
	}
	return targetList, nil
}

func (r *RegisterApi) GetReward(gid types.Gid, name string) (*RewardInfo, error) {
	snapshotBlock := r.chain.GetLatestSnapshotBlock()
	vmContext, err := vm_context.NewVmContext(r.chain, &snapshotBlock.Hash, nil, nil)
	if err != nil {
		return nil, err
	}
	info := abi.GetRegistration(vmContext, gid, name)
	if info == nil {
		return nil, nil
	}
	startIndex, endIndex, reward, periodTime, err := contracts.CalcReward(vmContext, info, gid)
	if err != nil {
		return nil, err
	}
	genesisTime := vmContext.GetGenesisSnapshotBlock().Timestamp.Unix()
	return &RewardInfo{*bigIntToString(reward), contracts.IndexToTime(startIndex, genesisTime, periodTime), contracts.IndexToTime(endIndex+1, genesisTime, periodTime)}, nil
}

type CandidateInfo struct {
	Name     string        `json:"name"`
	NodeAddr types.Address `json:"nodeAddr"`
	VoteNum  string        `json:"voteNum"`
}

func (r *RegisterApi) GetCandidateList(gid types.Gid) ([]*CandidateInfo, error) {
	head := r.chain.GetLatestSnapshotBlock()
	index, err := r.cs.VoteTimeToIndex(gid, *head.Timestamp)
	if err != nil {
		return nil, err
	}

	details, _, err := r.cs.ReadVoteMapByTime(gid, index)
	if err != nil {
		return nil, err
	}
	var result []*CandidateInfo
	for _, v := range details {
		result = append(result, &CandidateInfo{v.Name, v.CurrentAddr, *bigIntToString(v.Balance)})
	}
	return result, nil
}
