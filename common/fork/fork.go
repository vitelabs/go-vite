package fork

import (
	"github.com/vitelabs/go-vite/config"
	"reflect"
	"sort"
)

var forkPoints config.ForkPoints

type ForkPointItem struct {
	config.ForkPoint
	forkName string
}

var forkPointList ForkPointList
var forkPointMap ForkPointMap

type ForkPointList []*ForkPointItem
type ForkPointMap map[string]*ForkPointItem

func (a ForkPointList) Len() int           { return len(a) }
func (a ForkPointList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ForkPointList) Less(i, j int) bool { return a[i].Height < a[j].Height }

func SetForkPoints(points *config.ForkPoints) {
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

/**
@TODO add feature affected by this fork
use by a. xxx eg.
       b. xxx eg.
*/
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
