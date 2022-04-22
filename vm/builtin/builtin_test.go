package builtin

import (
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/common/upgrade"
	"testing"
)

func TestExists(t *testing.T) {
	upgrade.CleanupUpgradeBox(t)
	upgrade.InitUpgradeBox(upgrade.NewCustomUpgradeBox(
		map[string]*upgrade.UpgradePoint{
			"SeedFork":            &upgrade.UpgradePoint{Height: 100, Version: 1},
			"DexFork":             &upgrade.UpgradePoint{Height: 200, Version: 2},
			"DexFeeFork":          &upgrade.UpgradePoint{Height: 250, Version: 3},
			"StemFork":            &upgrade.UpgradePoint{Height: 300, Version: 4},
			"LeafFork":            &upgrade.UpgradePoint{Height: 400, Version: 5},
			"EarthFork":           &upgrade.UpgradePoint{Height: 500, Version: 6},
			"DexMiningFork":       &upgrade.UpgradePoint{Height: 600, Version: 7},
			"DexRobotFork":        &upgrade.UpgradePoint{Height: 600, Version: 8},
			"DexStableMarketFork": &upgrade.UpgradePoint{Height: 600, Version: 9},
			"Version10Fork": 	   &upgrade.UpgradePoint{Height: 1000, Version: 10},
			"VEP19Fork": 		   &upgrade.UpgradePoint{Height: 1100, Version: 11},
		},
	))

	assert.Equal(t, Exists(types.AddressAsset, AbiAsset.Methods["issue"].Id(), 1000), false)
	assert.Equal(t, Exists(types.AddressAsset, AbiAsset.Methods["issue"].Id(), 1100), true)
	assert.Equal(t, Exists(types.AddressAsset, AbiAsset.Methods["name"].Id(), 1100), true)
	assert.Equal(t, Exists(types.AddressAsset, []byte{1, 2, 3, 4}, 1100), false)
}
