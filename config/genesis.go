package config

import (
	"encoding/json"
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

type Genesis struct {
	GenesisAccountAddress *types.Address
	ForkPoints            *ForkPoints
	ConsensusGroupInfo    *GovernanceContractInfo // Deprecated
	GovernanceInfo        *GovernanceContractInfo
	MintageInfo           *AssetContractInfo // Deprecated
	AssetInfo             *AssetContractInfo
	PledgeInfo            *QuotaContractInfo // Deprecated
	QuotaInfo             *QuotaContractInfo
	AccountBalanceMap     map[string]map[string]*big.Int // address - tokenId - balanceAmount
}

func (g *Genesis) UnmarshalJSON(data []byte) error {
	type Alias Genesis
	aux := &struct{ *Alias }{Alias: (*Alias)(g)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if g.ConsensusGroupInfo != nil {
		g.GovernanceInfo = g.ConsensusGroupInfo
		for _, m := range g.GovernanceInfo.RegistrationInfoMap {
			for _, v := range m {
				v.BlockProducingAddress = v.NodeAddr
				v.StakeAddress = v.PledgeAddr
				v.ExpirationHeight = v.WithdrawHeight
				v.RevokeTime = v.CancelTime
				v.HistoryAddressList = v.HisAddrList
			}
		}
		for _, v := range g.GovernanceInfo.ConsensusGroupInfoMap {
			v.StakeAmount = v.PledgeAmount
			v.ExpirationHeight = v.WithdrawHeight
			v.RegisterConditionParam.StakeAmount = v.RegisterConditionParam.PledgeAmount
			v.RegisterConditionParam.StakeHeight = v.RegisterConditionParam.PledgeHeight
			v.RegisterConditionParam.StakeToken = v.RegisterConditionParam.PledgeToken
		}
	}
	if g.MintageInfo != nil {
		g.AssetInfo = g.MintageInfo
		for _, v := range g.AssetInfo.TokenInfoMap {
			v.IsOwnerBurnOnly = v.OwnerBurnOnly
		}
	}
	if g.PledgeInfo != nil {
		g.QuotaInfo = g.PledgeInfo
		g.QuotaInfo.StakeInfoMap = g.QuotaInfo.PledgeInfoMap
		g.QuotaInfo.StakeBeneficialMap = g.QuotaInfo.PledgeBeneficialMap
		for _, list := range g.QuotaInfo.StakeInfoMap {
			for _, v := range list {
				v.ExpirationHeight = v.WithdrawHeight
				v.Beneficiary = v.BeneficialAddr
			}
		}
	}
	return nil
}

func IsCompleteGenesisConfig(genesisConfig *Genesis) bool {
	if genesisConfig == nil || genesisConfig.GenesisAccountAddress == nil ||
		genesisConfig.GovernanceInfo == nil || len(genesisConfig.GovernanceInfo.ConsensusGroupInfoMap) == 0 ||
		len(genesisConfig.GovernanceInfo.RegistrationInfoMap) == 0 ||
		genesisConfig.AssetInfo == nil || len(genesisConfig.AssetInfo.TokenInfoMap) == 0 ||
		len(genesisConfig.AccountBalanceMap) == 0 {
		return false
	}
	return true
}

type ForkPoint struct {
	Height  uint64
	Version uint32
}

type ForkPoints struct {
	SeedFork      *ForkPoint
	DexFork       *ForkPoint
	DexFeeFork    *ForkPoint
	StemFork      *ForkPoint
	LeafFork      *ForkPoint
	EarthFork     *ForkPoint
	DexMiningFork *ForkPoint
}

type GenesisVmLog struct {
	Data   string
	Topics []types.Hash
}

type GovernanceContractInfo struct {
	ConsensusGroupInfoMap map[string]*ConsensusGroupInfo          // consensus group info, gid - info
	RegistrationInfoMap   map[string]map[string]*RegistrationInfo // registration info, gid - sbpName - info
	HisNameMap            map[string]map[string]string            // used node name for node addr, gid - blockProducingAddress - sbpName
	VoteStatusMap         map[string]map[string]string            // vote info, gid - voteAddr - sbpName
}

type AssetContractInfo struct {
	TokenInfoMap map[string]*TokenInfo // tokenId - info
	LogList      []*GenesisVmLog       // issue events
}

type QuotaContractInfo struct {
	PledgeInfoMap       map[string][]*StakeInfo // Deprecated
	StakeInfoMap        map[string][]*StakeInfo
	PledgeBeneficialMap map[string]*big.Int // Deprecated
	StakeBeneficialMap  map[string]*big.Int
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
	PledgeAmount           *big.Int // Deprecated
	StakeAmount            *big.Int
	WithdrawHeight         uint64 // Deprecated
	ExpirationHeight       uint64
}
type RegisterConditionParam struct {
	PledgeAmount *big.Int // Deprecated
	StakeAmount  *big.Int
	PledgeToken  types.TokenTypeId // Deprecated
	StakeToken   types.TokenTypeId
	PledgeHeight uint64 // Deprecated
	StakeHeight  uint64
}
type VoteConditionParam struct {
}
type RegistrationInfo struct {
	NodeAddr              *types.Address // Deprecated
	BlockProducingAddress *types.Address
	PledgeAddr            *types.Address // Deprecated
	StakeAddress          *types.Address
	Amount                *big.Int
	WithdrawHeight        uint64 // Deprecated
	ExpirationHeight      uint64
	RewardTime            int64
	CancelTime            int64 // Deprecated
	RevokeTime            int64
	HisAddrList           []types.Address // Deprecated
	HistoryAddressList    []types.Address
}
type TokenInfo struct {
	TokenName       string
	TokenSymbol     string
	TotalSupply     *big.Int
	Decimals        uint8
	Owner           types.Address
	MaxSupply       *big.Int
	OwnerBurnOnly   bool // Deprecated
	IsOwnerBurnOnly bool
	IsReIssuable    bool
}
type StakeInfo struct {
	Amount           *big.Int
	WithdrawHeight   uint64 // Deprecated
	ExpirationHeight uint64
	BeneficialAddr   *types.Address // Deprecated
	Beneficiary      *types.Address
}
