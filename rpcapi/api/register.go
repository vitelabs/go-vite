package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger/chain"
	"github.com/vitelabs/go-vite/ledger/consensus"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
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

// Private
func (r *RegisterApi) GetRegisterData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameRegister, gid, name, nodeAddr)
}

// Private
func (r *RegisterApi) GetCancelRegisterData(gid types.Gid, name string) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameRevoke, gid, name)
}

// Private
func (r *RegisterApi) GetRewardData(gid types.Gid, name string, beneficialAddr types.Address) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameWithdrawReward, gid, name, beneficialAddr)
}

// Private
func (r *RegisterApi) GetUpdateRegistrationData(gid types.Gid, name string, nodeAddr types.Address) ([]byte, error) {
	return abi.ABIGovernance.PackMethod(abi.MethodNameUpdateBlockProducingAddress, gid, name, nodeAddr)
}

type RegistrationInfo struct {
	Name                  string        `json:"name"`
	NodeAddr              types.Address `json:"nodeAddr"`
	PledgeAddr            types.Address `json:"pledgeAddr"`
	RewardWithdrawAddress types.Address `json:"rewardWithdrawAddress"`
	PledgeAmount          string        `json:"pledgeAmount"`
	WithdrawHeight        string        `json:"withdrawHeight"`
	WithdrawTime          int64         `json:"withdrawTime"`
	CancelTime            int64         `json:"cancelTime"`
}

type byRegistrationExpirationHeight []*types.Registration

func (a byRegistrationExpirationHeight) Len() int      { return len(a) }
func (a byRegistrationExpirationHeight) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byRegistrationExpirationHeight) Less(i, j int) bool {
	if a[i].ExpirationHeight == a[j].ExpirationHeight {
		if a[i].RevokeTime == a[j].RevokeTime {
			return a[i].Name > a[j].Name
		} else {
			return a[i].RevokeTime > a[j].RevokeTime
		}
	}
	return a[i].ExpirationHeight > a[j].ExpirationHeight
}

type Reward struct {
	BlockReward      string `json:"blockReward"`
	VoteReward       string `json:"voteReward"`
	TotalReward      string `json:"totalReward"`
	BlockNum         string `json:"blockNum"`
	ExpectedBlockNum string `json:"expectedBlockNum"`
	Drained          bool   `json:"drained"`
}

type RewardInfo struct {
	RewardMap map[string]*Reward `json:"rewardMap"`
	StartTime int64              `json:"startTime"`
	EndTime   int64              `json:"endTime"`
}
