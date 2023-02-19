package upgrade

// all upgrade point is active.
func NewLatestUpgradeBox() *upgradeBox {
	return newUpgradeBox([]*UpgradePoint{
		{
			Height:  1,
			Version: 1,
		},
		{
			Height:  1,
			Version: 2,
		},
		{
			Height:  1,
			Version: 3,
		},
		{
			Height:  1,
			Version: 4,
		},
		{
			Height:  1,
			Version: 5,
		},
		{
			Height:  1,
			Version: 6,
		},
		{
			Height:  1,
			Version: 7,
		},
		{
			Height:  1,
			Version: 8,
		},
		{
			Height:  1,
			Version: 9,
		},
		{
			Height:  1,
			Version: 10,
		},
		{
			Height:  1,
			Version: 11,
		},
		{
			Height:  1,
			Version: 12,
		},
		{
			Height:  1,
			Version: 13,
		},
	})
}

func NewMainnetUpgradeBox() *upgradeBox {
	return newUpgradeBox([]*UpgradePoint{
		{
			Name:    "SeedFork",
			Height:  3488471,
			Version: 1,
		},
		{
			Name:    "DexFork",
			Height:  5442723,
			Version: 2,
		},
		{
			Name:    "DexFeeFork",
			Height:  8013367,
			Version: 3,
		},
		{
			Name:    "StemFork",
			Height:  8403110,
			Version: 4,
		},
		{
			Name:    "LeafFork",
			Height:  9413600,
			Version: 5,
		},
		{
			Name:    "EarthFork",
			Height:  16634530,
			Version: 6,
		},
		{
			Name:    "DexMiningFork",
			Height:  17142720,
			Version: 7,
		},
		{
			Name:    "DexRobotFork",
			Height:  31305900,
			Version: 8,
		},
		{
			Name:    "DexStableMarketFork",
			Height:  39694000,
			Version: 9,
		},
		{
			Name:    "Version10",
			Height:  77106666,
			Version: 10,
		},
		{
			Name:    "Version11",
			Height:  101320000,
			Version: 11,
		},
		{
			Name:    "Version12",
			Height:  116480000,
			Version: 12,
		},
		{
			Name:    "VersionX",
			Height:  EndlessHeight,
			Version: 13,
		},
	})
}

func NewCustomUpgradeBox(points map[string]*UpgradePoint) *upgradeBox {
	result := []*UpgradePoint{}
	for key, value := range points {
		if value.Name == "" {
			value.Name = key
		}
		result = append(result, value)
	}
	return newUpgradeBox(result)
}

func NewEmptyUpgradeBox() *upgradeBox {
	return newUpgradeBox([]*UpgradePoint{})
}
