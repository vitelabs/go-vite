package config

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type Genesis struct {
	GenesisAccountAddress *types.Address
	ForkPoints            *fork.ForkPoints
	GovernanceInfo        *GovernanceContractInfo
	AssetInfo             *AssetContractInfo
	QuotaInfo             *QuotaContractInfo
	AccountBalanceMap     map[string]map[string]*big.Int // address - tokenId - balanceAmount
}

func (g *Genesis) UnmarshalJSON(data []byte) error {
	type Alias Genesis
	aux := &struct{ *Alias }{Alias: (*Alias)(g)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
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
	StakeInfoMap       map[string][]*StakeInfo
	StakeBeneficialMap map[string]*big.Int
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
	StakeAmount            *big.Int
	ExpirationHeight       uint64
}
type RegisterConditionParam struct {
	StakeAmount *big.Int
	StakeToken  types.TokenTypeId
	StakeHeight uint64
}
type VoteConditionParam struct {
}
type RegistrationInfo struct {
	BlockProducingAddress *types.Address
	StakeAddress          *types.Address
	Amount                *big.Int
	ExpirationHeight      uint64
	RewardTime            int64
	RevokeTime            int64
	HistoryAddressList    []types.Address
}
type TokenInfo struct {
	TokenName       string
	TokenSymbol     string
	TotalSupply     *big.Int
	Decimals        uint8
	Owner           types.Address
	MaxSupply       *big.Int
	IsOwnerBurnOnly bool
	IsReIssuable    bool
}
type StakeInfo struct {
	Amount           *big.Int
	ExpirationHeight uint64
	Beneficiary      *types.Address
}

func MakeGenesisConfig(genesisFile string) *Genesis {
	var genesisConfig *Genesis

	log := log15.New("module", "gvite/config")
	if len(genesisFile) > 0 {
		file, err := os.Open(genesisFile)
		if err != nil {
			log.Crit(fmt.Sprintf("Failed to read genesis file: %v", err), "method", "readGenesis")
		}
		defer file.Close()

		genesisConfig = new(Genesis)
		if err := json.NewDecoder(file).Decode(genesisConfig); err != nil {
			log.Crit(fmt.Sprintf("invalid genesis file: %v", err), "method", "readGenesis")
		}
		if !IsCompleteGenesisConfig(genesisConfig) {
			log.Crit(fmt.Sprintf("invalid genesis file, genesis account info is not complete"), "method", "readGenesis")
		}
	} else {
		genesisConfig = makeGenesisAccountConfig()
	}

	// set fork points
	genesisConfig.ForkPoints = makeForkPointsConfig(genesisConfig)
	return genesisConfig
}

func makeForkPointsConfig(genesisConfig *Genesis) *fork.ForkPoints {
	// checkForkPoints(genesisConfig.ForkPoints)
	if genesisConfig != nil && genesisConfig.ForkPoints != nil {
		if err := fork.CheckForkPoints(*genesisConfig.ForkPoints); err != nil {
			panic(err)
		}
		return genesisConfig.ForkPoints
	} else {
		return &fork.ForkPoints{
			SeedFork: &fork.ForkPoint{
				Height:  3488471,
				Version: 1,
			},

			DexFork: &fork.ForkPoint{
				Height:  5442723,
				Version: 2,
			},

			DexFeeFork: &fork.ForkPoint{
				Height:  8013367,
				Version: 3,
			},

			StemFork: &fork.ForkPoint{
				Height:  8403110,
				Version: 4,
			},
			LeafFork: &fork.ForkPoint{
				Height:  9413600,
				Version: 5,
			},

			EarthFork: &fork.ForkPoint{
				Height:  16634530,
				Version: 6,
			},

			DexMiningFork: &fork.ForkPoint{
				Height:  17142720,
				Version: 7,
			},

			DexRobotFork: &fork.ForkPoint{
				Height:  31305900,
				Version: 8,
			},

			DexStableMarketFork: &fork.ForkPoint{
				Height:  39694000,
				Version: 9,
			},
		}
	}
}

func makeGenesisAccountConfig() *Genesis {
	g := new(Genesis)
	err := json.Unmarshal([]byte(GenesisJson()), g)
	if err != nil {
		panic(err)
	}
	return g
}

func GenesisJson() string {
	return genesisJson
}
