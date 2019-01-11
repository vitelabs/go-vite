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

type ForkPointList []*ForkPointItem

func (a ForkPointList) Len() int           { return len(a) }
func (a ForkPointList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ForkPointList) Less(i, j int) bool { return a[i].Height < a[j].Height }

func SetForkPoints(points *config.ForkPoints) {
	forkPoints = *points

	t := reflect.TypeOf(forkPoints)
	v := reflect.ValueOf(forkPoints)

	for k := 0; k < t.NumField(); k++ {
		forkPoint := v.Field(k).Interface().(*config.ForkPoint)
		forkPointList = append(forkPointList, &ForkPointItem{
			ForkPoint: *forkPoint,
			forkName:  t.Field(k).Name,
		})
	}

	sort.Sort(forkPointList)
}

func IsSmartFork(blockHeight uint64) bool {
	return forkPoints.Vite1.Height > 0 && blockHeight >= forkPoints.Vite1.Height
}

func GetForkPoints() config.ForkPoints {
	return forkPoints
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
