package fork

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/config"
	"reflect"
	"sort"
)

var forkPoints config.ForkPoints

type ForkPointItem struct {
	config.ForkPoint
	ForkName string
}

type ForkPointList []*ForkPointItem
type ForkPointMap map[string]*ForkPointItem

var activeChecker ActiveChecker

var forkPointList ForkPointList
var forkPointMap ForkPointMap

func (a ForkPointList) Len() int           { return len(a) }
func (a ForkPointList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ForkPointList) Less(i, j int) bool { return a[i].Height < a[j].Height }

func IsInitForkPoint() bool {
	return forkPointMap != nil
}
func IsInitActiveChecker() bool {
	return activeChecker != nil
}

func SetForkPoints(points *config.ForkPoints) {
	if points != nil {
		forkPoints = *points

		t := reflect.TypeOf(forkPoints)
		v := reflect.ValueOf(forkPoints)
		forkPointMap = make(ForkPointMap)

		for k := 0; k < t.NumField(); k++ {
			forkPoint := v.Field(k).Interface().(*config.ForkPoint)

			forkName := t.Field(k).Name
			forkPointItem := &ForkPointItem{
				ForkPoint: *forkPoint,
				ForkName:  forkName,
			}

			// set fork point list
			forkPointList = append(forkPointList, forkPointItem)
			// set fork point map
			forkPointMap[forkName] = forkPointItem
		}

		sort.Sort(forkPointList)
	}
}

func SetActiveChecker(ac ActiveChecker) {
	activeChecker = ac
}

func CheckForkPoints(points config.ForkPoints) error {
	t := reflect.TypeOf(points)
	v := reflect.ValueOf(points)

	for k := 0; k < t.NumField(); k++ {
		forkPoint := v.Field(k).Interface().(*config.ForkPoint)

		if forkPoint == nil {
			return errors.New(fmt.Sprintf("The fork point %s can't be nil. the `ForkPoints` config in genesis.json is not correct, "+
				"you can remove the `ForkPoints` key in genesis.json then use the default config of `ForkPoints`", t.Field(k).Name))
		}

		if forkPoint.Height <= 0 {
			return errors.New(fmt.Sprintf("The height of fork point %s is 0. "+
				"the `ForkPoints` config in genesis.json is not correct, you can remove the `ForkPoints` key in genesis.json then use the default config of `ForkPoints`", t.Field(k).Name))
		}

		if forkPoint.Version <= 0 {
			return errors.New(fmt.Sprintf("The version of fork point %s is 0. "+
				"the `ForkPoints` config in genesis.json is not correct, you can remove the `ForkPoints` key in genesis.json then use the default config of `ForkPoints`", t.Field(k).Name))
		}

	}

	return nil
}

/*
IsSeedFork checks whether current snapshot block height is over seed hard fork.
Vite pre-mainnet hard forks at snapshot block height 3488471.
Contents:
  1. Vm log list hash add account address and prevHash since seed fork.
  2. Create contract params add seed count since seed fork.
  3. Verifier verifies seed count since seed fork.
  4. Vm interpreters add SEED opcode since seed fork.
*/
func IsSeedFork(snapshotHeight uint64) bool {
	seedForkPoint, ok := forkPointMap["SeedFork"]
	if !ok {
		panic("check seed fork failed. SeedFork is not existed.")
	}
	return snapshotHeight >= seedForkPoint.Height && IsForkActive(*seedForkPoint)
}

/*
IsDexFork checks whether current snapshot block height is over sprout hard fork.
Vite pre-mainnet hard forks at snapshot block height 5442723.
Features:
  1. Dynamic quota acquisition. Quota acquisition from staking will reduce
     when network traffic rate is too high.
  2. Adjustment of quota consumption for some built-in contract transactions
     and VM instructions.
  3. ViteX decentralized exchange support.
*/
func IsDexFork(snapshotHeight uint64) bool {
	dexForkPoint, ok := forkPointMap["DexFork"]
	if !ok {
		panic("check dex fork failed. DexFork is not existed.")
	}
	return snapshotHeight >= dexForkPoint.Height && IsForkActive(*dexForkPoint)
}

/*
IsDexFeeFork checks whether current snapshot block height is over dex fee hard fork.
Vite pre-mainnet hard forks at snapshot block height 8013367.
Dex fee hard fork is an emergency hard fork to solve one wrongly placed order which
has caused ViteX failed to display user balances.
*/
func IsDexFeeFork(snapshotHeight uint64) bool {
	dexFeeForkPoint, ok := forkPointMap["DexFeeFork"]
	if !ok {
		panic("check dex fee fork failed. DexFeeFork is not existed.")
	}
	return snapshotHeight >= dexFeeForkPoint.Height && IsForkActive(*dexFeeForkPoint)
}

/*
IsStemFork checks whether current snapshot block height is over stem hard fork.
Vite pre-mainnet hard forks at snapshot block height 8403110.
Features:
  1. Capability of placing/cancelling orders via delegation.
  2. Super VIP membership. Stake and then enjoy zero trading fee!
     (Additional operator fee cannot be exempted)
*/
func IsStemFork(snapshotHeight uint64) bool {
	stemForkPoint, ok := forkPointMap["StemFork"]
	if !ok {
		panic("check stem fork failed. StemFork is not existed.")
	}
	return snapshotHeight >= stemForkPoint.Height && IsForkActive(*stemForkPoint)
}

func IsLeafFork(snapshotHeight uint64) bool {
	leafForkPoint, ok := forkPointMap["LeafFork"]
	if !ok {
		panic("check leaf fork failed. LeafFork is not existed.")
	}
	return snapshotHeight >= leafForkPoint.Height && IsForkActive(*leafForkPoint)
}

func IsEarthFork(snapshotHeight uint64) bool {
	earthForkPoint, ok := forkPointMap["EarthFork"]
	if !ok {
		panic("check earth fork failed. EarthFork is not existed.")
	}
	return snapshotHeight >= earthForkPoint.Height && IsForkActive(*earthForkPoint)
}

func IsDexMiningFork(snapshotHeight uint64) bool {
	dexMiningForkPoint, ok := forkPointMap["DexMiningFork"]
	if !ok {
		panic("check dex mining fork failed. DexMiningFork is not existed.")
	}
	return snapshotHeight >= dexMiningForkPoint.Height && IsForkActive(*dexMiningForkPoint)
}

func GetLeafForkPoint() *ForkPointItem {
	leafForkPoint, ok := forkPointMap["LeafFork"]
	if !ok {
		panic("check leaf fork failed. LeafFork is not existed.")
	}

	return leafForkPoint
}

func IsActiveForkPoint(snapshotHeight uint64) bool {
	// assume that fork point list is sorted by height asc
	for i := len(forkPointList) - 1; i >= 0; i-- {
		forkPoint := forkPointList[i]
		if forkPoint.Height == snapshotHeight {
			return IsForkActive(*forkPoint)
		}

		if forkPoint.Height < snapshotHeight {
			break
		}
	}

	return false
}

func GetForkPoint(snapshotHeight uint64) *ForkPointItem {
	// assume that fork point list is sorted by height asc
	for i := len(forkPointList) - 1; i >= 0; i-- {
		forkPoint := forkPointList[i]
		if forkPoint.Height == snapshotHeight {
			return forkPoint
		}

		if forkPoint.Height < snapshotHeight {
			break
		}
	}

	return nil
}

func GetForkPoints() config.ForkPoints {
	return forkPoints
}

func GetForkPointList() ForkPointList {
	return forkPointList
}

func GetForkPointMap() ForkPointMap {
	return forkPointMap
}

func GetActiveForkPointList() ForkPointList {
	activeForkPointList := make(ForkPointList, 0, len(forkPointList))
	for _, forkPoint := range forkPointList {
		if IsForkActive(*forkPoint) {
			activeForkPointList = append(activeForkPointList, forkPoint)
		}
	}

	return activeForkPointList
}

func GetRecentActiveFork(blockHeight uint64) *ForkPointItem {
	for i := len(forkPointList) - 1; i >= 0; i-- {
		item := forkPointList[i]
		if item.Height <= blockHeight && IsForkActive(*item) {
			return item
		}
	}
	return nil
}

func GetLastForkPoint() *ForkPointItem {
	return forkPointList[forkPointList.Len()-1]
}

func IsForkActive(point ForkPointItem) bool {
	// TODO suppose all point is active.
	return true
	//return activeChecker.IsForkActive(point)
}
