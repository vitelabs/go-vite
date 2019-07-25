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
	forkName string
}

type ForkPointList []*ForkPointItem
type ForkPointMap map[string]*ForkPointItem

var isInitForkPoint = false
var forkPointList ForkPointList
var forkPointMap = make(ForkPointMap)

func (a ForkPointList) Len() int           { return len(a) }
func (a ForkPointList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ForkPointList) Less(i, j int) bool { return a[i].Height < a[j].Height }

func IsInit() bool {
	return isInitForkPoint
}

func SetForkPoints(points *config.ForkPoints) {
	if points != nil {
		forkPoints = *points

		t := reflect.TypeOf(forkPoints)
		v := reflect.ValueOf(forkPoints)

		for k := 0; k < t.NumField(); k++ {
			forkPoint := v.Field(k).Interface().(*config.ForkPoint)

			forkName := t.Field(k).Name
			forkPointItem := &ForkPointItem{
				ForkPoint: *forkPoint,
				forkName:  forkName,
			}

			// set fork point list
			forkPointList = append(forkPointList, forkPointItem)
			// set fork point map
			forkPointMap[forkName] = forkPointItem
		}

		sort.Sort(forkPointList)
	}

	isInitForkPoint = true
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
	return snapshotHeight >= seedForkPoint.Height
}

func IsDexFork(snapshotHeight uint64) bool {
	dexForkPoint, ok := forkPointMap["DexFork"]
	if !ok {
		panic("check dex fork failed. DexFork is not existed.")
	}
	return snapshotHeight >= dexForkPoint.Height
}

func IsForkPoint(snapshotHeight uint64) bool {
	// assume that fork point list is sorted by height asc
	for i := len(forkPointList) - 1; i >= 0; i-- {
		forkPoint := forkPointList[i]
		if forkPoint.Height == snapshotHeight {
			return true
		}

		if forkPoint.Height < snapshotHeight {
			break
		}
	}

	return false
}

func GetForkPoints() config.ForkPoints {
	return forkPoints
}

func GetForkPointList() ForkPointList {
	return forkPointList
}

func GetRecentForkName(blockHeight uint64) string {
	for i := len(forkPointList) - 1; i >= 0; i-- {
		item := forkPointList[i]
		if item.Height <= blockHeight {
			return item.forkName
		}
	}
	return ""
}
