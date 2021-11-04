package upgrade

import (
	"fmt"
	"sort"
)

type upgradeBox struct {
	pointMap     map[uint32]*UpgradePoint
	heightMap    map[uint64]bool
	sortedPoints []*UpgradePoint
}

func (box upgradeBox) checkBox() {
	lastHeight := uint64(0)
	for index, ele := range box.sortedPoints {
		if index != int(ele.Version)-1 {
			panic("error version in upgrade box")
		}
		if ele.Height < lastHeight {
			panic("error height in upgrade box")
		}
		lastHeight = ele.Height
	}
}

func (box *upgradeBox) initFromArray(points []*UpgradePoint) {
	for _, value := range points {
		if value.Name == "" {
			value.Name = fmt.Sprintf("%d", value.Version)
		}
		box.pointMap[value.Version] = value
		box.heightMap[value.Height] = true
	}
	box.sortedPoints = sortPoints(box.pointMap)
}

func (box *upgradeBox) AddPoint(version uint32, height uint64) UpgradeBox {
	if len(box.pointMap) > 0 && box.pointMap[version-1] == nil {
		panic("last upgrade version is missing")
	}
	box.pointMap[version] = &UpgradePoint{Version: version, Height: height}
	box.heightMap[height] = true
	box.sortedPoints = sortPoints(box.pointMap)
	return box
}

func (box upgradeBox) latestPoint() *UpgradePoint {
	var result *UpgradePoint
	for _, v := range box.sortedPoints {
		if v.Height < EndlessHeight {
			result = v
		} else {
			break
		}
	}
	return result
}

func (box upgradeBox) currentPoint(height uint64) *UpgradePoint {
	var result *UpgradePoint
	for _, v := range box.sortedPoints {
		if v.Height <= height {
			result = v
		} else {
			break
		}
	}
	return result
}

func (box upgradeBox) activePoints(height uint64) []*UpgradePoint {
	var result []*UpgradePoint
	for _, v := range box.sortedPoints {
		if v.Height <= height {
			result = append(result, v)
		}
	}
	return result
}

func (box upgradeBox) getUpgradePoint(version uint32) *UpgradePoint {
	return box.pointMap[version]
}

func (box upgradeBox) isActive(version uint32, height uint64) bool {
	point := box.pointMap[version]
	if point == nil {
		return false
	}
	return height >= point.Height
}

// hit the upgrade point
func (box upgradeBox) isPoint(height uint64) bool {
	return box.heightMap[height]
}

func (box upgradeBox) UpgradePoints() []*UpgradePoint {
	return box.sortedPoints
}

func newUpgradeBox(points []*UpgradePoint) *upgradeBox {
	result := &upgradeBox{
		pointMap:     make(map[uint32]*UpgradePoint),
		heightMap:    make(map[uint64]bool),
		sortedPoints: []*UpgradePoint{},
	}
	result.initFromArray(points)
	result.checkBox()
	return result
}

func sortPoints(pointMap map[uint32]*UpgradePoint) []*UpgradePoint {
	result := []*UpgradePoint{}
	for _, value := range pointMap {
		result = append(result, value)
	}
	sort.Sort(byVersion(result))
	return result
}

type byVersion []*UpgradePoint

func (a byVersion) Len() int           { return len(a) }
func (a byVersion) Less(i, j int) bool { return a[i].Version < a[j].Version }
func (a byVersion) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
