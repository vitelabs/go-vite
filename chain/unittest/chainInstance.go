package chain_unittest

import (
	"encoding/json"
	"fmt"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/node"
	"math/big"
	"os"
	"path/filepath"
)

func makeForkPointsConfig(genesisConfig *config.Genesis) *config.ForkPoints {
	forkPoints := &config.ForkPoints{}

	if genesisConfig != nil && genesisConfig.ForkPoints != nil {
		forkPoints = genesisConfig.ForkPoints
	}

	if forkPoints.Smart == nil {
		forkPoints.Smart = &config.ForkPoint{}
	}
	if forkPoints.Smart.Height == 0 {
		forkPoints.Smart.Height = 5788912
	}

	return forkPoints
}

func makeChainConfig(genesisFile string) *config.Genesis {
	defaultGenesisAccountAddress, _ := types.HexToAddress("vite_60e292f0ac471c73d914aeff10bb25925e13b2a9fddb6e6122")
	var defaultBlockProducers []types.Address
	addrStrList := []string{
		"vite_0acbb1335822c8df4488f3eea6e9000eabb0f19d8802f57c87",
		"vite_14edbc9214bd1e5f6082438f707d10bf43463a6d599a4f2d08",
		"vite_1630f8c0cf5eda3ce64bd49a0523b826f67b19a33bc2a5dcfb",
		"vite_1b1dfa00323aea69465366d839703547fec5359d6c795c8cef",
		"vite_27a258dd1ed0ce0de3f4abd019adacd1b4b163b879389d3eca",
		"vite_31a02e4f4b536e2d6d9bde23910cdffe72d3369ef6fe9b9239",
		"vite_383fedcbd5e3f52196a4e8a1392ed3ddc4d4360e4da9b8494e",
		"vite_41ba695ff63caafd5460dcf914387e95ca3a900f708ac91f06",
		"vite_545c8e4c74e7bb6911165e34cbfb83bc513bde3623b342d988",
		"vite_5a1b5ece654138d035bdd9873c1892fb5817548aac2072992e",
		"vite_70cfd586185e552635d11f398232344f97fc524fa15952006d",
		"vite_76df2a0560694933d764497e1b9b11f9ffa1524b170f55dda0",
		"vite_7b76ca2433c7ddb5a5fa315ca861e861d432b8b05232526767",
		"vite_7caaee1d51abad4047a58f629f3e8e591247250dad8525998a",
		"vite_826a1ab4c85062b239879544dc6b67e3b5ce32d0a1eba21461",
		"vite_89007189ad81c6ee5cdcdc2600a0f0b6846e0a1aa9a58e5410",
		"vite_9abcb7324b8d9029e4f9effe76f7336bfd28ed33cb5b877c8d",
		"vite_af60cf485b6cc2280a12faac6beccfef149597ea518696dcf3",
		"vite_c1090802f735dfc279a6c24aacff0e3e4c727934e547c24e5e",
		"vite_c10ae7a14649800b85a7eaaa8bd98c99388712412b41908cc0",
		"vite_d45ac37f6fcdb1c362a33abae4a7d324a028aa49aeea7e01cb",
		"vite_d8974670af8e1f3c4378d01d457be640c58644bc0fa87e3c30",
		"vite_e289d98f33c3ef5f1b41048c2cb8b389142f033d1df9383818",
		"vite_f53dcf7d40b582cd4b806d2579c6dd7b0b131b96c2b2ab5218",
		"vite_fac06662d84a7bea269265e78ea2d9151921ba2fae97595608",
	}

	for _, addrStr := range addrStrList {
		addr, _ := types.HexToAddress(addrStr)
		defaultBlockProducers = append(defaultBlockProducers, addr)
	}

	defaultSnapshotConsensusGroup := config.ConsensusGroupInfo{
		NodeCount:           25,
		Interval:            1,
		PerCount:            3,
		RandCount:           2,
		RandRank:            100,
		CountingTokenId:     ledger.ViteTokenId,
		RegisterConditionId: 1,
		RegisterConditionParam: config.ConditionRegisterData{
			PledgeAmount: new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
			PledgeHeight: uint64(3600 * 24 * 90),
			PledgeToken:  ledger.ViteTokenId,
		},
		VoteConditionId: 1,
		Owner:           defaultGenesisAccountAddress,
		PledgeAmount:    big.NewInt(0),
		WithdrawHeight:  1,
	}
	defaultCommonConsensusGroup := config.ConsensusGroupInfo{
		NodeCount:           25,
		Interval:            3,
		PerCount:            1,
		RandCount:           2,
		RandRank:            100,
		CountingTokenId:     ledger.ViteTokenId,
		RegisterConditionId: 1,
		RegisterConditionParam: config.ConditionRegisterData{
			PledgeAmount: new(big.Int).Mul(big.NewInt(5e5), big.NewInt(1e18)),
			PledgeHeight: uint64(3600 * 24 * 90),
			PledgeToken:  ledger.ViteTokenId,
		},
		VoteConditionId: 1,
		Owner:           defaultGenesisAccountAddress,
		PledgeAmount:    big.NewInt(0),
		WithdrawHeight:  1,
	}

	genesisConfig := &config.Genesis{
		GenesisAccountAddress: defaultGenesisAccountAddress,
		BlockProducers:        defaultBlockProducers,
	}

	if len(genesisFile) > 0 {
		file, err := os.Open(genesisFile)
		if err != nil {

			log15.Crit(fmt.Sprintf("Failed to read genesis file: %v", err), "method", "readGenesis")
		}
		defer file.Close()

		genesisConfig = new(config.Genesis)
		if err := json.NewDecoder(file).Decode(genesisConfig); err != nil {
			log15.Crit(fmt.Sprintf("invalid genesis file: %v", err), "method", "readGenesis")
		}
	}

	if genesisConfig.SnapshotConsensusGroup == nil {
		genesisConfig.SnapshotConsensusGroup = &defaultSnapshotConsensusGroup
	}

	if genesisConfig.CommonConsensusGroup == nil {
		genesisConfig.CommonConsensusGroup = &defaultCommonConsensusGroup
	}

	// set fork points
	genesisConfig.ForkPoints = makeForkPointsConfig(genesisConfig)

	return genesisConfig
}

func NewChainInstance(dirName string, clearDataDir bool) chain.Chain {
	dataDir := filepath.Join(node.DefaultDataDir(), dirName)

	if clearDataDir {
		os.RemoveAll(dataDir)
	}

	chainInstance := chain.NewChain(&config.Config{
		DataDir: dataDir,

		Genesis: makeChainConfig(""),
	})

	chainInstance.Init()

	return chainInstance
}
