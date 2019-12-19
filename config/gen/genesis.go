package config_gen

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/log"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/config"
)

func MakeGenesisConfig(genesisFile string) *config.Genesis {
	var genesisConfig *config.Genesis

	if len(genesisFile) > 0 {
		file, err := os.Open(genesisFile)
		if err != nil {
			log.Crit(fmt.Sprintf("Failed to read genesis file: %v", err), "method", "readGenesis")
		}
		defer file.Close()

		genesisConfig = new(config.Genesis)
		if err := json.NewDecoder(file).Decode(genesisConfig); err != nil {
			log.Crit(fmt.Sprintf("invalid genesis file: %v", err), "method", "readGenesis")
		}
		if !config.IsCompleteGenesisConfig(genesisConfig) {
			log.Crit(fmt.Sprintf("invalid genesis file, genesis account info is not complete"), "method", "readGenesis")
		}
	} else {
		genesisConfig = makeGenesisAccountConfig()
	}

	// set fork points
	genesisConfig.ForkPoints = makeForkPointsConfig(genesisConfig)
	return genesisConfig
}

func makeForkPointsConfig(genesisConfig *config.Genesis) *config.ForkPoints {
	// checkForkPoints(genesisConfig.ForkPoints)
	if genesisConfig != nil && genesisConfig.ForkPoints != nil {
		if err := fork.CheckForkPoints(*genesisConfig.ForkPoints); err != nil {
			panic(err)
		}
		return genesisConfig.ForkPoints
	} else {
		return &config.ForkPoints{
			SeedFork: &config.ForkPoint{
				Height:  3488471,
				Version: 1,
			},

			DexFork: &config.ForkPoint{
				Height:  5442723,
				Version: 2,
			},

			DexFeeFork: &config.ForkPoint{
				Height:  8013367,
				Version: 3,
			},

			StemFork: &config.ForkPoint{
				Height:  8403110,
				Version: 4,
			},
			LeafFork: &config.ForkPoint{
				Height:  9413600,
				Version: 5,
			},

			EarthFork: &config.ForkPoint{
				Height:  16634530,
				Version: 6,
			},

			DexMiningFork: &config.ForkPoint{
				Height:  17142720,
				Version: 7,
			},
		}
	}
}

//var genesisJsonStr = "{\"GenesisAccountAddress\": \"vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122\",\"ForkPoints\": null,\"GovernanceInfo\": {\"ConsensusGroupInfoMap\": {\"00000000000000000001\": {\"NodeCount\": 25,\"Interval\": 1,\"PerCount\": 3,\"RandCount\": 2,\"RandRank\": 100,\"Repeat\": 1,\"CheckLevel\": 0,\"CountingTokenId\": \"tti_5649544520544f4b454e6e40\",\"RegisterConditionId\": 1,\"RegisterConditionParam\": {\"StakeAmount\": 500000000000000000000000,\"StakeToken\": \"tti_5649544520544f4b454e6e40\",\"StakeHeight\": 7776000},\"VoteConditionId\": 1,\"VoteConditionParam\": {},\"Owner\": \"vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122\",\"StakeAmount\": 0,\"ExpirationHeight\": 1},\"00000000000000000002\": {\"NodeCount\": 25,\"Interval\": 3,\"PerCount\": 1,\"RandCount\": 2,\"RandRank\": 100,\"Repeat\": 48,\"CheckLevel\": 1,\"CountingTokenId\": \"tti_5649544520544f4b454e6e40\",\"RegisterConditionId\": 1,\"RegisterConditionParam\": {\"StakeAmount\": 500000000000000000000000,\"StakeToken\": \"tti_5649544520544f4b454e6e40\",\"StakeHeight\": 7776000},\"VoteConditionId\": 1,\"VoteConditionParam\": {},\"Owner\": \"vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122\",\"StakeAmount\": 0,\"ExpirationHeight\": 1}},\"RegistrationInfoMap\": {\"00000000000000000001\": {\"Vite_SBP01\": {\"BlockProducingAddress\": \"vite_9065ff0e14ebf983e090cde47d59fe77d7164b576a6d2d0eda\",\"StakeAddress\": \"vite_79f6168f60d19dbf74c9f180264a14a47675f2840696873ff7\",\"Amount\": 500000000000000000000000,\"ExpirationHeight\": 7776000,\"RewardTime\": 1,\"RevokeTime\": 0,\"HistoryAddressList\": [\"vite_9065ff0e14ebf983e090cde47d59fe77d7164b576a6d2d0eda\"]},\"Vite_SBP02\": {\"BlockProducingAddress\": \"vite_995769283a01ba8d00258dbb5371c915df59c8657335bfb1b2\",\"StakeAddress\": \"vite_79f6168f60d19dbf74c9f180264a14a47675f2840696873ff7\",\"Amount\": 500000000000000000000000,\"ExpirationHeight\": 7776000,\"RewardTime\": 1,\"RevokeTime\": 0,\"HistoryAddressList\": [\"vite_995769283a01ba8d00258dbb5371c915df59c8657335bfb1b2\"]},\"Vite_SBP03\": {\"BlockProducingAddress\": \"vite_9b33ce9fc70f14407db75cfa8453680f364e6674c7cc1fb785\",\"StakeAddress\": \"vite_79f6168f60d19dbf74c9f180264a14a47675f2840696873ff7\",\"Amount\": 500000000000000000000000,\"ExpirationHeight\": 7776000,\"RewardTime\": 1,\"RevokeTime\": 0,\"HistoryAddressList\": [\"vite_9b33ce9fc70f14407db75cfa8453680f364e6674c7cc1fb785\"]},\"Vite_SBP04\": {\"BlockProducingAddress\": \"vite_aadf6275ecbf07181c5c37c7d709aebf06553e470345d3f699\",\"StakeAddress\": \"vite_79f6168f60d19dbf74c9f180264a14a47675f2840696873ff7\",\"Amount\": 500000000000000000000000,\"ExpirationHeight\": 7776000,\"RewardTime\": 1,\"RevokeTime\": 0,\"HistoryAddressList\": [\"vite_aadf6275ecbf07181c5c37c7d709aebf06553e470345d3f699\"]},\"Vite_SBP05\": {\"BlockProducingAddress\": \"vite_c1d11e6eda9a9b80e388a38e0ac541cbc3333736233b4eaaab\",\"StakeAddress\": \"vite_79f6168f60d19dbf74c9f180264a14a47675f2840696873ff7\",\"Amount\": 500000000000000000000000,\"ExpirationHeight\": 7776000,\"RewardTime\": 1,\"RevokeTime\": 0,\"HistoryAddressList\": [\"vite_c1d11e6eda9a9b80e388a38e0ac541cbc3333736233b4eaaab\"]}}},\"VoteStatusMap\": {\"00000000000000000001\": {\"vite_002f9cc5863815d27679b3a5e4f47675e680926a7ae5365d5e\": \"Vite_SBP01\",\"vite_00306f5d2904c8b2e4fbb0ff07635130bd6c4f14f3c2cc9d32\": \"Vite_SBP02\",\"vite_0040b24dacaa808b4fccec4a482b127e40542398e5880d21ad\": \"Vite_SBP03\",\"vite_0047b193c7d7791a94de7e45b7febf6eac8139fd81695cfdb5\": \"Vite_SBP04\",\"vite_0097b268fb1954e8acbd7ff6d90788e2a118c7a8c48207bc83\": \"Vite_SBP05\"}}},\"AssetInfo\": {\"TokenInfoMap\": {\"tti_251a3e67a41b5ea2373936c8\": {\"TokenName\": \"Vite Community Point\",\"TokenSymbol\": \"VCP\",\"TotalSupply\": 10000000000,\"Decimals\": 0,\"Owner\": \"vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122\",\"MaxSupply\": 0,\"IsOwnerBurnOnly\": false,\"IsReIssuable\": false},\"tti_5649544520544f4b454e6e40\": {\"TokenName\": \"Vite Token\",\"TokenSymbol\": \"VITE\",\"TotalSupply\": 1000000000000000000000000000,\"Decimals\": 18,\"Owner\": \"vite_0000000000000000000000000000000000000004d28108e76b\",\"MaxSupply\": 115792089237316195423570985008687907853269984665640564039457584007913129639935,\"IsOwnerBurnOnly\": false,\"IsReIssuable\": true}},\"LogList\": [{\"Data\": \"\",\"Topics\": [\"3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6\",\"000000000000000000000000000000000000000000005649544520544f4b454e\"]},{\"Data\": \"\",\"Topics\": [\"3f9dcc00d5e929040142c3fb2b67a3be1b0e91e98dac18d5bc2b7817a4cfecb6\",\"00000000000000000000000000000000000000000000251a3e67a41b5ea23739\"]}]},\"QuotaInfo\": {\"StakeInfoMap\": {\"vite_002f9cc5863815d27679b3a5e4f47675e680926a7ae5365d5e\": [{\"Amount\": 1100000000000000000000,\"ExpirationHeight\": 1,\"Beneficiary\": \"vite_002f9cc5863815d27679b3a5e4f47675e680926a7ae5365d5e\"}],\"vite_0040b24dacaa808b4fccec4a482b127e40542398e5880d21ad\": [{\"Amount\": 1000000000000000000000,\"ExpirationHeight\": 1,\"Beneficiary\": \"vite_0040b24dacaa808b4fccec4a482b127e40542398e5880d21ad\"}]},\"StakeBeneficialMap\": {\"vite_002f9cc5863815d27679b3a5e4f47675e680926a7ae5365d5e\": 1100000000000000000000,\"vite_0040b24dacaa808b4fccec4a482b127e40542398e5880d21ad\": 1000000000000000000000}},\"AccountBalanceMap\": {\"vite_002f9cc5863815d27679b3a5e4f47675e680926a7ae5365d5e\": {\"tti_251a3e67a41b5ea2373936c8\": 1000000,\"tti_5649544520544f4b454e6e40\": 27068892782446871294625},\"vite_0040b24dacaa808b4fccec4a482b127e40542398e5880d21ad\": {\"tti_251a3e67a41b5ea2373936c8\": 5000,\"tti_5649544520544f4b454e6e40\": 13575090107549484290851},\"vite_0040b24dacaa808b4fccec4a482b127e40542398e5880d21ad\": {\"tti_251a3e67a41b5ea2373936c8\": 9998995000,\"tti_5649544520544f4b454e6e40\": 999959356017110003644414524},\"vite_79f6168f60d19dbf74c9f180264a14a47675f2840696873ff7\": {}}}"

func makeGenesisAccountConfig() *config.Genesis {
	g := new(config.Genesis)
	err := json.Unmarshal([]byte(genesisJsonStr), g)
	if err != nil {
		panic(err)
	}
	return g
}
