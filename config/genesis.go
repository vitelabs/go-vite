package config

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
)

type Genesis struct {
	GenesisAccountAddress *types.Address
	ForkPoints            *ForkPoints
	ConsensusGroupInfo    *ConsensusGroupContractInfo
	MintageInfo           *MintageContractInfo
	PledgeInfo            *PledgeContractInfo
	AccountBalanceMap     map[string]map[string]*big.Int // address - tokenId - balanceAmount
}

func IsCompleteGenesisConfig(genesisConfig *Genesis) bool {
	if genesisConfig == nil || genesisConfig.GenesisAccountAddress == nil ||
		genesisConfig.ConsensusGroupInfo == nil || len(genesisConfig.ConsensusGroupInfo.ConsensusGroupInfoMap) == 0 ||
		len(genesisConfig.ConsensusGroupInfo.RegistrationInfoMap) == 0 ||
		genesisConfig.MintageInfo == nil || len(genesisConfig.MintageInfo.TokenInfoMap) == 0 ||
		len(genesisConfig.AccountBalanceMap) == 0 {
		return false
	}
	return true
}

type ForkPoint struct {
	Height uint64
	Hash   *types.Hash
}

type ForkPoints struct{}

type GenesisVmLog struct {
	Data   string
	Topics []types.Hash
}

type ConsensusGroupContractInfo struct {
	ConsensusGroupInfoMap map[string]ConsensusGroupInfo          // consensus group info, gid - info
	RegistrationInfoMap   map[string]map[string]RegistrationInfo // registration info, gid - nodeName - info
	HisNameMap            map[string]map[string]string           // used node name for node addr, gid - nodeAddr - nodeName
	VoteStatusMap         map[string]map[string]string           // vote info, gid - voteAddr - nodeName
}

type MintageContractInfo struct {
	TokenInfoMap map[string]TokenInfo // tokenId - info
	LogList      []GenesisVmLog       // mint events
}

type PledgeContractInfo struct {
	PledgeInfoMap       map[string]PledgeInfo
	PledgeBeneficialMap map[string]*big.Int
}

type ConsensusGroupInfo struct {
	NodeCount              uint8
	Interval               int64
	PerCount               int64
	RandCount              uint8
	RandRank               uint8
	Repeat                 uint16
	CheckLevel             uint8
	CountingTokenId        types.TokenTypeId
	RegisterConditionId    uint8
	RegisterConditionParam RegisterConditionParam
	VoteConditionId        uint8
	VoteConditionParam     VoteConditionParam
	Owner                  types.Address
	PledgeAmount           *big.Int
	WithdrawHeight         uint64
}
type RegisterConditionParam struct {
	PledgeAmount *big.Int
	PledgeToken  types.TokenTypeId
	PledgeHeight uint64
}
type VoteConditionParam struct {
}
type RegistrationInfo struct {
	NodeAddr       types.Address
	PledgeAddr     types.Address
	Amount         *big.Int
	WithdrawHeight uint64
	RewardTime     int64
	CancelTime     int64
	HisAddrList    []types.Address
}
type TokenInfo struct {
	TokenName      string
	TokenSymbol    string
	TotalSupply    *big.Int
	Decimals       uint8
	Owner          types.Address
	PledgeAmount   *big.Int
	PledgeAddr     types.Address
	WithdrawHeight uint64
	MaxSupply      *big.Int
	OwnerBurnOnly  bool
	IsReIssuable   bool
}
type PledgeInfo struct {
	Amount         *big.Int
	WithdrawHeight uint64
	BeneficialAddr types.Address
}
