package upgrade

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testUpgradePoint(t *testing.T, isUpgrade func(sHeight uint64) bool, sHeight uint64) {
	assert.False(t, isUpgrade(sHeight-1), sHeight)
	assert.True(t, isUpgrade(sHeight), sHeight)
	assert.True(t, isUpgrade(sHeight+1), sHeight)
}

func testEmptyUpgradePoint(t *testing.T, isUpgrade func(sHeight uint64) bool, sHeight uint64) {
	assert.False(t, isUpgrade(sHeight-1), sHeight)
	assert.False(t, isUpgrade(sHeight), sHeight)
	assert.False(t, isUpgrade(sHeight+1), sHeight)
}

func TestMainnetUpgradeBox(t *testing.T) {
	cleanupUpgradeBox()
	InitUpgradeBox(NewMainnetUpgradeBox())

	type uPoint struct {
		fc      func(sHeight uint64) bool
		sHeight uint64
	}
	cases := []*uPoint{
		{
			IsSeedUpgrade,
			3488471,
		},
		{
			IsDexUpgrade,
			5442723,
		},
		{
			IsDexFeeUpgrade,
			8013367,
		},
		{
			IsStemUpgrade,
			8403110,
		},
		{
			IsLeafUpgrade,
			9413600,
		},
		{
			IsEarthUpgrade,
			16634530,
		},
		{
			IsDexMiningUpgrade,
			17142720,
		},
		{
			IsDexRobotUpgrade,
			31305900,
		},
		{
			IsDexStableMarketUpgrade,
			39694000,
		},
		{
			IsVersion10Upgrade,
			77106666,
		},
		{
			IsVersion11Upgrade,
			101320000,
		},
		{
			IsVersion12Upgrade,
			116480000,
		},
		{
			IsVersionXUpgrade,
			EndlessHeight,
		},
	}
	for _, ele := range cases {
		testUpgradePoint(t, ele.fc, ele.sHeight)
	}
	assert.Equal(t, (int)(GetLatestPoint().Version), 12)
}

func TestLatestUpgradeBox(t *testing.T) {
	cleanupUpgradeBox()
	InitUpgradeBox(NewLatestUpgradeBox())

	type uPoint struct {
		fc      func(sHeight uint64) bool
		sHeight uint64
	}
	cases := []*uPoint{
		{
			IsSeedUpgrade,
			1,
		},
		{
			IsDexUpgrade,
			1,
		},
		{
			IsDexFeeUpgrade,
			1,
		},
		{
			IsStemUpgrade,
			1,
		},
		{
			IsLeafUpgrade,
			1,
		},
		{
			IsEarthUpgrade,
			1,
		},
		{
			IsDexMiningUpgrade,
			1,
		},
		{
			IsDexRobotUpgrade,
			1,
		},
		{
			IsDexStableMarketUpgrade,
			1,
		},
		{
			IsVersion10Upgrade,
			1,
		},
		{
			IsVersion11Upgrade,
			1,
		},
		{
			IsVersion12Upgrade,
			1,
		},
		{
			IsVersionXUpgrade,
			1,
		},
	}
	for _, ele := range cases {
		testUpgradePoint(t, ele.fc, ele.sHeight)
	}
}

func TestCustomUpgradeBox(t *testing.T) {
	cleanupUpgradeBox()

	raw := `{
		"SeedFork":{
		"Height":1,
		"Version":1
		},
		"DexFork":{
		"Height":2,
		"Version":2
		},
		"DexFeeFork":{
		"Height":3,
		"Version":3
		},
		"StemFork":{
		"Height":4,
		"Version":4
		},
		"LeafFork":{
		"Height":5,
		"Version":5
		},
		"EarthFork":{
		"Height":6,
		"Version":6
		},
		"DexMiningFork":{
		"Height":7,
		"Version":7
		},
		"DexRobotFork": {
		"Height":8,
		"Version":8
		},
		"DexStableMarketFork": {
		"Height":9,
		"Version":9
		}
	}`

	var m map[string]*UpgradePoint
	json.Unmarshal([]byte(raw), &m)

	InitUpgradeBox(NewCustomUpgradeBox(m))

	type uPoint struct {
		fc      func(sHeight uint64) bool
		sHeight uint64
	}
	cases := []*uPoint{
		{
			IsSeedUpgrade,
			1,
		},
		{
			IsDexUpgrade,
			2,
		},
		{
			IsDexFeeUpgrade,
			3,
		},
		{
			IsStemUpgrade,
			4,
		},
		{
			IsLeafUpgrade,
			5,
		},
		{
			IsEarthUpgrade,
			6,
		},
		{
			IsDexMiningUpgrade,
			7,
		},
		{
			IsDexRobotUpgrade,
			8,
		},
		{
			IsDexStableMarketUpgrade,
			9,
		},
	}
	for _, ele := range cases {
		testUpgradePoint(t, ele.fc, ele.sHeight)
	}
}

func TestEmptyUpgradeBox(t *testing.T) {
	cleanupUpgradeBox()
	// panic when add discontinuous version
	assert.PanicsWithValue(t, "last upgrade version is missing", func() {
		NewEmptyUpgradeBox().AddPoint(1, 20).AddPoint(3, 90)
	})

	InitUpgradeBox(NewEmptyUpgradeBox().AddPoint(1, 20).AddPoint(2, 90))

	type uPoint struct {
		fc      func(sHeight uint64) bool
		sHeight uint64
	}
	cases := []*uPoint{
		{
			IsSeedUpgrade,
			20,
		},
		{
			IsDexUpgrade,
			90,
		},
	}
	for _, ele := range cases {
		testUpgradePoint(t, ele.fc, ele.sHeight)
	}

	cases = []*uPoint{
		{
			IsStemUpgrade,
			20,
		},
		{
			IsEarthUpgrade,
			90,
		},
	}
	for _, ele := range cases {
		testEmptyUpgradePoint(t, ele.fc, ele.sHeight)
	}
}

/**
 * 1. GetCurPoint
 * 2. GetActivePoints
 */
func TestUpgrade(t *testing.T) {
	cleanupUpgradeBox()

	InitUpgradeBox(NewEmptyUpgradeBox().AddPoint(1, 10).AddPoint(2, 20).AddPoint(3, 30))

	cur := GetCurPoint(15)
	assert.True(t, cur.Height == 10)
	assert.True(t, cur.Version == 1)

	cur = GetCurPoint(2)
	assert.Nil(t, cur)

	activePoints := GetActivePoints(25)

	assert.True(t, len(activePoints) == 2)
	assert.True(t, activePoints[0].Version == 1)
	assert.True(t, activePoints[1].Version == 2)

}
