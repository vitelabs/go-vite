package contracts

import (
	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm/abi"
	"math/big"
	"strings"
)

const (
	MethodNameRegister           = "Register"
	MethodNameCancelRegister     = "CancelRegister"
	MethodNameReward             = "Reward"
	MethodNameUpdateRegistration = "UpdateRegistration"
	VariableNameRegistration     = "registration"

	MethodNameVote         = "Vote"
	MethodNameCancelVote   = "CancelVote"
	VariableNameVoteStatus = "voteStatus"

	MethodNamePledge             = "Pledge"
	MethodNameCancelPledge       = "CancelPledge"
	VariableNamePledgeInfo       = "pledgeInfo"
	VariableNamePledgeBeneficial = "pledgeBeneficial"

	MethodNameCreateConsensusGroup        = "CreateConsensusGroup"
	MethodNameCancelConsensusGroup        = "CancelConsensusGroup"
	MethodNameReCreateConsensusGroup      = "ReCreateConsensusGroup"
	VariableNameConsensusGroupInfo        = "consensusGroupInfo"
	VariableNameConditionRegisterOfPledge = "registerOfPledge"
	VariableNameConditionVoteOfBalance    = "voteOfKeepToken"

	MethodNameMintage             = "Mintage"
	MethodNameMintageCancelPledge = "CancelPledge"
	VariableNameMintage           = "mintage"
)

const jsonRegister = `
[
	{"type":"function","name":"Register", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"},{"name":"beneficialAddr","type":"address"}]},
	{"type":"function","name":"UpdateRegistration", "inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"},{"name":"beneficialAddr","type":"address"}]},
	{"type":"function","name":"CancelRegister","inputs":[{"name":"gid","type":"gid"}, {"name":"name","type":"string"}]},
	{"type":"function","name":"Reward","inputs":[{"name":"gid","type":"gid"},{"name":"name","type":"string"},{"name":"endHeight","type":"uint64"},{"name":"startHeight","type":"uint64"},{"name":"amount","type":"uint256"}]},
	{"type":"variable","name":"registration","inputs":[{"name":"name","type":"string"},{"name":"NodeAddr","type":"address"},{"name":"pledgeAddr","type":"address"},{"name":"beneficialAddr","type":"address"},{"name":"amount","type":"uint256"},{"name":"pledgeHeight","type":"uint64"},{"name":"rewardHeight","type":"uint64"},{"name":"cancelHeight","type":"uint64"}]}
]`
const jsonVote = `
[
	{"type":"function","name":"Vote", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeName","type":"string"}]},
	{"type":"function","name":"CancelVote","inputs":[{"name":"gid","type":"gid"}]},
	{"type":"variable","name":"voteStatus","inputs":[{"name":"nodeName","type":"string"}]}
]`
const jsonPledge = `
[
	{"type":"function","name":"Pledge", "inputs":[{"name":"beneficial","type":"address"}]},
	{"type":"function","name":"CancelPledge","inputs":[{"name":"beneficial","type":"address"},{"name":"amount","type":"uint256"}]},
	{"type":"variable","name":"pledgeInfo","inputs":[{"name":"amount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
	{"type":"variable","name":"pledgeBeneficial","inputs":[{"name":"amount","type":"uint256"}]}
]`
const jsonConsensusGroup = `
[
	{"type":"function","name":"CreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"},{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"}]},
	{"type":"function","name":"CancelConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
	{"type":"function","name":"ReCreateConsensusGroup", "inputs":[{"name":"gid","type":"gid"}]},
	{"type":"variable","name":"consensusGroupInfo","inputs":[{"name":"nodeCount","type":"uint8"},{"name":"interval","type":"int64"},{"name":"perCount","type":"int64"},{"name":"randCount","type":"uint8"},{"name":"randRank","type":"uint8"},{"name":"countingTokenId","type":"tokenId"},{"name":"registerConditionId","type":"uint8"},{"name":"registerConditionParam","type":"bytes"},{"name":"voteConditionId","type":"uint8"},{"name":"voteConditionParam","type":"bytes"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]},
	{"type":"variable","name":"registerOfPledge","inputs":[{"name":"pledgeAmount","type":"uint256"},{"name":"pledgeToken","type":"tokenId"},{"name":"pledgeHeight","type":"uint64"}]},
	{"type":"variable","name":"voteOfKeepToken","inputs":[{"name":"keepAmount","type":"uint256"},{"name":"keepToken","type":"tokenId"}]}
]`
const jsonMintage = `
[
	{"type":"function","name":"Mintage","inputs":[{"name":"tokenId","type":"tokenId"},{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"}]},
	{"type":"function","name":"CancelPledge","inputs":[{"name":"tokenId","type":"tokenId"}]},
	{"type":"variable","name":"mintage","inputs":[{"name":"tokenName","type":"string"},{"name":"tokenSymbol","type":"string"},{"name":"totalSupply","type":"uint256"},{"name":"decimals","type":"uint8"},{"name":"owner","type":"address"},{"name":"pledgeAmount","type":"uint256"},{"name":"withdrawHeight","type":"uint64"}]}
]
`

var (
	ABIRegister, _       = abi.JSONToABIContract(strings.NewReader(jsonRegister))
	ABIVote, _           = abi.JSONToABIContract(strings.NewReader(jsonVote))
	ABIPledge, _         = abi.JSONToABIContract(strings.NewReader(jsonPledge))
	ABIConsensusGroup, _ = abi.JSONToABIContract(strings.NewReader(jsonConsensusGroup))
	ABIMintage, _        = abi.JSONToABIContract(strings.NewReader(jsonMintage))
)

type ParamRegister struct {
	Gid            types.Gid
	Name           string
	NodeAddr       types.Address
	BeneficialAddr types.Address
}
type ParamCancelRegister struct {
	Gid  types.Gid
	Name string
}
type ParamReward struct {
	Gid         types.Gid
	Name        string
	EndHeight   uint64
	StartHeight uint64
	Amount      *big.Int
}
type ParamVote struct {
	Gid      types.Gid
	NodeName string
}
type VariablePledgeBeneficial struct {
	Amount *big.Int
}
type ParamCancelPledge struct {
	Beneficial types.Address
	Amount     *big.Int
}
type VariableConditionRegisterOfPledge struct {
	PledgeAmount *big.Int
	PledgeToken  types.TokenTypeId
	PledgeHeight uint64
}
type VariableConditionVoteOfKeepToken struct {
	KeepAmount *big.Int
	KeepToken  types.TokenTypeId
}
type ParamMintage struct {
	TokenId     types.TokenTypeId
	TokenName   string
	TokenSymbol string
	TotalSupply *big.Int
	Decimals    uint8
}

func GetRegisterKey(name string, gid types.Gid) []byte {
	var data = make([]byte, types.HashSize)
	copy(data[:types.GidSize], gid[:])
	copy(data[types.GidSize:], types.DataHash([]byte(name)).Bytes()[types.GidSize:])
	return data
}
func GetVoteKey(addr types.Address, gid types.Gid) []byte {
	var data = make([]byte, types.HashSize)
	copy(data[:types.GidSize], gid[:])
	copy(data[types.GidSize:types.GidSize+types.AddressSize], addr[:])
	return data
}
func GetAddrFromVoteKey(key []byte) types.Address {
	addr, _ := types.BytesToAddress(key[types.GidSize : types.GidSize+types.AddressSize])
	return addr
}
func GetPledgeBeneficialKey(beneficial types.Address) []byte {
	return beneficial.Bytes()
}
func GetPledgeKey(addr types.Address, pledgeBeneficialKey []byte) []byte {
	return append(addr.Bytes(), pledgeBeneficialKey...)
}
func IsPledgeKey(key []byte) bool {
	return len(key) > types.AddressSize
}
func GetBeneficialFromPledgeKey(key []byte) types.Address {
	address, _ := types.BytesToAddress(key[types.AddressSize:])
	return address
}
func GetConsensusGroupKey(gid types.Gid) []byte {
	return helper.LeftPadBytes(gid.Bytes(), types.HashSize)
}
func GetGidFromConsensusGroupKey(key []byte) types.Gid {
	gid, _ := types.BytesToGid(key[types.HashSize-types.GidSize:])
	return gid
}
func GetMintageKey(tokenId types.TokenTypeId) []byte {
	return helper.LeftPadBytes(tokenId.Bytes(), types.HashSize)
}
func GetTokenIdFromMintageKey(key []byte) types.TokenTypeId {
	tokenId, _ := types.BytesToTokenTypeId(key[types.HashSize-types.TokenTypeIdSize:])
	return tokenId
}
