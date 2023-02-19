package upgrade

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/log15"
)

var log = log15.New("module", "upgrade")

type UpgradePoint struct {
	Name    string
	Height  uint64
	Version uint32
}

type UpgradeBox interface {
	UpgradePoints() []*UpgradePoint
	AddPoint(version uint32, height uint64) UpgradeBox
	activePoints(height uint64) []*UpgradePoint
	latestPoint() *UpgradePoint
	currentPoint(height uint64) *UpgradePoint
	isPoint(height uint64) bool
	isActive(version uint32, height uint64) bool
	getUpgradePoint(version uint32) *UpgradePoint
}

var upgrade UpgradeBox

var EndlessHeight = uint64(1000000000)

func assertUpgradeNotNil() {
	if upgrade == nil {
		panic("upgrade is nil")
	}
}

func cleanupUpgradeBox() {
	upgrade = nil
}

func CleanupUpgradeBox() {
	log.Info("clean up upgrade box")
	cleanupUpgradeBox()
}

func InitUpgradeBox(box UpgradeBox) error {
	if upgrade != nil {
		panic("init upgrade twice")
	}
	points := box.UpgradePoints()
	log.Info(fmt.Sprintf("init upgrade: %s\n", common.ToJson(points)))
	upgrade = newUpgradeBox(points)
	return nil
}

func AddUpgradePoint(version uint32, height uint64) error {
	upgrade = upgrade.AddPoint(version, height)
	return nil
}

func IsUpgradePoint(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isPoint(sHeight)
}

func GetCurPoint(sHeight uint64) *UpgradePoint {
	assertUpgradeNotNil()
	return upgrade.currentPoint(sHeight)
}

func GetLatestPoint() *UpgradePoint {
	assertUpgradeNotNil()
	return upgrade.latestPoint()
}

func GetActivePoints(sHeight uint64) []*UpgradePoint {
	assertUpgradeNotNil()
	return upgrade.activePoints(sHeight)
}

func GetAllPoints() []*UpgradePoint {
	return upgrade.UpgradePoints()
}

/*
IsSeedUpgrade checks whether current snapshot block height is over seed hard fork.
Vite pre-mainnet hard forks at snapshot block height 3488471.
Contents:
  1. Vm log list hash add account address and prevHash since seed fork.
  2. Create contract params add seed count since seed fork.
  3. Verifier verifies seed count since seed fork.
  4. Vm interpreters add SEED opcode since seed fork.
*/
func IsSeedUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(1, sHeight)
}

/*
IsDexUpgrade checks whether current snapshot block height is over sprout hard fork.
Vite pre-mainnet hard forks at snapshot block height 5442723.
Features:
  1. Dynamic quota acquisition. Quota acquisition from staking will reduce
     when network traffic rate is too high.
  2. Adjustment of quota consumption for some built-in contract transactions
     and VM instructions.
  3. ViteX decentralized exchange support.
*/
func IsDexUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(2, sHeight)
}

/*
IsDexFeeUpgrade checks whether current snapshot block height is over dex fee hard fork.
Vite pre-mainnet hard forks at snapshot block height 8013367.
Dex fee hard fork is an emergency hard fork to solve one wrongly placed order which
has caused ViteX failed to display user balances.
*/
func IsDexFeeUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(3, sHeight)
}

/*
IsStemUpgrade checks whether current snapshot block height is over stem hard fork.
Vite pre-mainnet hard forks at snapshot block height 8403110.
Features:
  1. Capability of placing/cancelling orders via delegation.
  2. Super VIP membership. Stake and then enjoy zero trading fee!
     (Additional operator fee cannot be exempted)
*/
func IsStemUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(4, sHeight)
}

func IsLeafUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(5, sHeight)
}
func GetLeafUpgradePoint() UpgradePoint {
	assertUpgradeNotNil()
	return *upgrade.getUpgradePoint(5)
}

func IsEarthUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(6, sHeight)
}

func IsDexMiningUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(7, sHeight)
}

func IsDexRobotUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(8, sHeight)
}

func IsDexStableMarketUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(9, sHeight)
}

func IsVersion10Upgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(10, sHeight)
}

func IsVersion11Upgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(11, sHeight)
}

func IsVersion12Upgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(12, sHeight)
}

func IsVersionXUpgrade(sHeight uint64) bool {
	assertUpgradeNotNil()
	return upgrade.isActive(13, sHeight)
}
