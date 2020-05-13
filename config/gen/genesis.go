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

			DexRobotFork: &config.ForkPoint{
				Height:  31305900,
				Version: 8,
			},
		}
	}
}

func makeGenesisAccountConfig() *config.Genesis {
	g := new(config.Genesis)
	err := json.Unmarshal([]byte(genesisJsonStr), g)
	if err != nil {
		panic(err)
	}
	return g
}
