package vm

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"strings"
)

const (
	MethodNameMintage             = "Mintage"
	MethodNameMintageCancelPledge = "CancelPledge"
	VariableNameMintage           = "mintage"

	MethodNameRegister       = "Register"
	MethodNameCancelRegister = "CancelRegister"
	MethodNameReward         = "Reward"
	VariableNameRegistration = "registration"

	MethodNameVote         = "Vote"
	MethodNameCancelVote   = "CancelVote"
	VariableNameVoteStatus = "voteStatus"

	MethodNamePledge             = "Pledge"
	MethodNameCancelPledge       = "CancelPledge"
	VariableNamePledgeInfo       = "pledgeInfo"
	VariableNamePledgeBeneficial = "pledgeBeneficial"

	MethodNameCreateConsensusGroup = "CreateConsensusGroup"
	VariableNameConsensusGroupInfo = "consensusGroupInfo"
	VariableNameConditionCounting1 = "counting1"
	VariableNameConditionRegister1 = "register1"
	VariableNameConditionVote2     = "vote2"
)

const json_mintage = `
[
	{"type":"function","name":"Mintage","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"}]},
	{"type":"function","name":"CancelPledge","inputs":[{"name":"tokenId","type":"tokenId"}]},
	{"type":"variable","name":"mintage","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"timestamp","type":"int64"}]}
]
`
const json_register = `
[
	{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"}]},
	{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}]},
	{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"endHeight","type":"uint64"},{"name":"startHeight","type":"uint64"},{"name":"amount","type":"uint256"}]},
	{"type":"variable","name":"registration","inputs":[{"name":"amount","type":"uint256"},{"name":"timestamp","type":"int64"},{"name":"rewardHeight","type":"uint64"},{"name":"cancelHeight","type":"uint64"}]}
]`
const json_vote = `
[
	{"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"node","type":"address"}]},
	{"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]},
	{"type":"variable","name":"voteStatus","inputs":[{"name":"node","type":"address"}]}
]`
const json_pledge = `
[
	{"type":"function","name":"Pledge", "inputs":[{"name":"beneficial","type":"address"},{"name":"withdrawTime","type":"int64"}]},
	{"type":"function","name":"CancelPledge","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
	{"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawTime","type":"int64"}]},
	{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
]`
const json_consensusGroup = `
[
	{"type":"function","name":"CreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"countingRuleId","type":"uint8"},{"name":"countingRuleParam","type":"bytes"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"}]},
	{"type":"variable","name":"consensusGroupInfo","inputs":[{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"countingRuleId","type":"uint8"},{"name":"countingRuleParam","type":"bytes"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"}]},
	{"type":"variable","name":"counting1","inputs":[{"name":"tokenId","type":"tokenId"}]},
	{"type":"variable","name":"register1","inputs":[{"name":"pledgeAmount","type":"uint256"},{"name":"pledgeToken","type":"tokenId"},{"name":"pledgeTime","type":"int64"}]},
	{"type":"variable","name":"vote2","inputs":[{"name":"keepAmount","type":"uint256"},{"name":"keepToken","type":"tokenId"}]}
]`

var (
	ABI_mintage, _        = abi.JSONToABIContract(strings.NewReader(json_mintage))
	ABI_register, _       = abi.JSONToABIContract(strings.NewReader(json_register))
	ABI_vote, _           = abi.JSONToABIContract(strings.NewReader(json_vote))
	ABI_pledge, _         = abi.JSONToABIContract(strings.NewReader(json_pledge))
	ABI_consensusGroup, _ = abi.JSONToABIContract(strings.NewReader(json_consensusGroup))
)

type ParamMintage struct {
	TokenId     types.TokenTypeId
	TokenName   string
	TokenSymbol string
	TotalSupply *big.Int
	Decimals    uint8
}
type TokenInfo struct {
	TokenName    string
	TokenSymbol  string
	TotalSupply  *big.Int
	Decimals     uint8
	Owner        types.Address
	PledgeAmount *big.Int
	Timestamp    int64
}

type VariableRegistration struct {
	Amount       *big.Int
	Timestamp    int64
	RewardHeight uint64
	CancelHeight uint64
}
type ParamReward struct {
	Gid         types.Gid
	EndHeight   uint64
	StartHeight uint64
	Amount      *big.Int
}
type ParamVote struct {
	Gid  types.Gid
	Node types.Address
}
type VariablePledgeInfo struct {
	Amount       *big.Int
	WithdrawTime int64
}
type VariablePledgeBeneficial struct {
	Amount *big.Int
}
type ParamPledge struct {
	Beneficial   types.Address
	WithdrawTime int64
}
type ParamCancelPledge struct {
	Beneficial types.Address
	Amount     *big.Int
}
type ConsensusGroupInfo struct {
	NodeCount              uint8
	Interval               int64
	CountingRuleId         uint8
	CountingRuleParam      []byte
	RegisterConditionId    uint8
	RegisterConditionParam []byte
	VoteConditionId        uint8
	VoteConditionParam     []byte
}
type VariableConditionRegister1 struct {
	PledgeAmount *big.Int
	PledgeToken  types.TokenTypeId
	PledgeTime   int64
}
type VariableConditionVote2 struct {
	KeepAmount *big.Int
	KeepToken  types.TokenTypeId
}
type ParamCreateConsensusGroup struct {
	Gid types.Gid
	ConsensusGroupInfo
}
