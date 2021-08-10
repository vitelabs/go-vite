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
	box.pointMap[version] = &UpgradePoint{Version: version, Height: height}
	box.heightMap[height] = true
	box.sortedPoints = sortPoints(box.pointMap)
	return box
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
		pointMap: make(map[uint32]*UpgradePoint),
	}
	result.initFromArray(points)
	return result
}

func sortPoints(pointMap map[uint32]*UpgradePoint) []*UpgradePoint {
	result := []*UpgradePoint{}
	for _, value := range pointMap {
		result = append(result, value)
	}
	sort.Sort(byHeight(result))
	return result
}

type byHeight []*UpgradePoint

func (a byHeight) Len() int           { return len(a) }
func (a byHeight) Less(i, j int) bool { return a[i].Height < a[j].Height }
func (a byHeight) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
