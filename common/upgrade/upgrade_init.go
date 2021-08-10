package upgrade

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
			Height:  1000000000,
			Version: 10,
		},
	})
}

func NewCustomUpgradeBox(points map[string]*UpgradePoint) *upgradeBox {
	result := []*UpgradePoint{}
	for _, value := range points {
		result = append(result, value)
	}
	return newUpgradeBox(result)
}

func NewEmptyUpgradeBox() *upgradeBox {
	return newUpgradeBox([]*UpgradePoint{})
}
