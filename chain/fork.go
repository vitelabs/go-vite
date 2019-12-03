package chain

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/fork"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"math"
	"sort"
)

func (c *chain) IsForkActive(point fork.ForkPointItem) bool {
	if point.Height <= c.forkActiveCheckPoint.Height {
		// For backward compatibility, auto active old fork point
		return true
	}

	if c.forkActiveCache == nil {
		panic("Not initActiveFork, Can't check IsForkActive")
	}

	return c.checkIsActiveInCache(point)
}

func (c *chain) initActiveFork() error {
	c.forkActiveCache = make(fork.ForkPointList, 0)

	forkPointList := fork.GetForkPointList()

	latestSnapshotBlock := c.GetLatestSnapshotBlock()

	for _, forkPoint := range forkPointList {
		if forkPoint.Height <= c.forkActiveCheckPoint.Height {
			continue
		}
		if forkPoint.Height > latestSnapshotBlock.Height {
			break
		}
		if c.checkIsActive(*forkPoint) {
			c.forkActiveCache = append(c.forkActiveCache, forkPoint)
		}
	}

	return nil
}

func (c *chain) addActiveForkPoint(snapshotBlock *ledger.SnapshotBlock) {
	point := fork.GetForkPoint(snapshotBlock.Height + 1)
	if point == nil {
		return
	}

	// not active
	if !c.checkIsActive(*point) {
		c.log.Info(fmt.Sprintf("fork point is not active, height: %d, version: %d", point.Height, point.Version))
		return
	}

	c.forkActiveCache = append(c.forkActiveCache, point)
}

func (c *chain) deleteActiveForkPoint(chunks []*ledger.SnapshotChunk) {
	if c.forkActiveCache.Len() <= 0 {
		return
	}

	var firstSbHeight uint64
	for _, chunk := range chunks {
		if chunk.SnapshotBlock == nil {
			continue
		}
		firstSbHeight = chunk.SnapshotBlock.Height
		break
	}
	if firstSbHeight <= 0 {
		return
	}

	height := firstSbHeight

	deleteTo := -1
	for i := c.forkActiveCache.Len() - 1; i >= 0; i-- {
		point := c.forkActiveCache[i]

		if point.Height-1 >= height {
			deleteTo = i
		} else {
			break
		}
	}

	if deleteTo >= 0 {
		newForkActiveCache := make(fork.ForkPointList, deleteTo)
		copy(newForkActiveCache, c.forkActiveCache[:deleteTo])

		c.forkActiveCache = newForkActiveCache
	}
}

func (c *chain) checkIsActiveInCache(point fork.ForkPointItem) bool {
	forkActiveCache := c.forkActiveCache
	if forkActiveCache.Len() <= 0 {
		return false
	}

	pointIndex := sort.Search(forkActiveCache.Len(), func(i int) bool {
		forkActivePoint := forkActiveCache[i]
		return forkActivePoint.Height >= point.Height
	})

	if pointIndex < 0 || pointIndex >= forkActiveCache.Len() {
		return false
	}
	searchedPoint := forkActiveCache[pointIndex]
	if searchedPoint.Height == point.Height && searchedPoint.Version == point.Version {
		return true
	}
	return false
}

func (c *chain) checkIsActive(point fork.ForkPointItem) bool {
	snapshotHeight := point.Height - 1

	producers := c.getTopProducersMap(snapshotHeight)
	if producers == nil {
		return false
	}

	headers, err := c.GetSnapshotHeadersByHeight(snapshotHeight, false, 600)
	if err != nil {
		panic(fmt.Sprintf("GetSnapshotHeadersByHeight failed. SnapshotHeight is %d. Error is %s.", snapshotHeight, err))
	}

	if len(headers) <= 0 {
		return false
	}

	if headers[0].Height < snapshotHeight {
		return false
	}

	var versionCountMap = make(map[uint32]int, 2)

	var producerCounted = make(map[types.Address]struct{}, len(producers))
	for _, header := range headers {
		if len(producerCounted) >= len(producers) {
			break
		}

		producer := header.Producer()
		if _, ok := producers[producer]; !ok {
			continue
		}

		if _, ok := producerCounted[producer]; ok {
			continue
		}

		versionCountMap[header.Version]++
		producerCounted[producer] = struct{}{}
	}

	versionCount := versionCountMap[point.Version]
	legalCount := int(math.Ceil(float64(len(producers)*2) / 3))
	if versionCount >= legalCount {
		return true
	}
	return false
}

func (c *chain) getTopProducersMap(snapshotHeight uint64) map[types.Address]struct{} {
	snapshotHash, err := c.GetSnapshotHashByHeight(snapshotHeight)
	if err != nil {
		panic(fmt.Sprintf("GetSnapshotHashByHeight failed. snapshotHeight is %d. Error: %s", snapshotHeight, err))
	}
	if snapshotHash == nil {
		return nil
	}

	snapshotConsensusGroupInfo, err := c.GetConsensusGroup(*snapshotHash, types.SNAPSHOT_GID)

	count := int(snapshotConsensusGroupInfo.NodeCount)

	if err != nil {
		panic(fmt.Sprintf("GetConsensusGroup failed. Error: %s", err))
	}
	if snapshotConsensusGroupInfo == nil {
		panic("snapshotConsensusGroupInfo can't be nil")
	}

	voteDetails, err := c.CalVoteDetails(types.SNAPSHOT_GID, core.NewGroupInfo(*c.genesisSnapshotBlock.Timestamp, *snapshotConsensusGroupInfo), ledger.HashHeight{
		Hash:   *snapshotHash,
		Height: snapshotHeight,
	})
	if err != nil {
		panic(fmt.Sprintf("CalVoteDetails failed. snapshotHeight is %d. Error: %s", snapshotHeight, err))
	}

	topProducers := make(map[types.Address]struct{}, count)
	end := len(voteDetails)
	if end > count {
		end = count
	}

	for i := 0; i < end; i++ {
		topProducers[voteDetails[i].CurrentAddr] = struct{}{}
	}
	return topProducers
}
