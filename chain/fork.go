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

	return false
}

func (c *chain) initForkActive() error {
	return nil
}

func (c *chain) checkNewForkPoint() {

}

func (c *chain) rollbackForkPoint() {

}

func (c *chain) checkIsActiveInCache(point fork.ForkPointItem) bool {
	if c.forkActiveCache.Len() <= 0 {
		return false
	}
	pointIndex := sort.Search(c.forkActiveCache.Len(), func(i int) bool {
		forkActivePoint := c.forkActiveCache[i]
		return forkActivePoint.Height >= point.Height
	})

	if pointIndex < 0 || pointIndex >= c.forkActiveCache.Len() {
		return false
	}
	searchedPoint := c.forkActiveCache[pointIndex]
	if searchedPoint.Height == point.Height && searchedPoint.Version == point.Version {
		return true
	}
	return false
}

func (c *chain) checkIsActive(point fork.ForkPointItem) bool {
	snapshotHeight := point.Height

	producers := c.getTopProducersMap(snapshotHeight, 25)
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

	var versionCountMap = make(map[uint32]int, 2)

	var producerCounted = make(map[types.Address]struct{}, len(producers))
	for i := len(headers) - 1; i >= 0; i-- {
		if len(producerCounted) >= len(producers) {
			break
		}

		header := headers[i]
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

func (c *chain) getTopProducersMap(snapshotHeight uint64, count int) map[types.Address]struct{} {
	snapshotHash, err := c.GetSnapshotHashByHeight(snapshotHeight)
	if err != nil {
		panic(fmt.Sprintf("GetSnapshotHashByHeight failed. snapshotHeight is %d. Error: %s", snapshotHeight, err))
	}
	if snapshotHash == nil {
		return nil
	}

	snapshotConsensusGroupInfo, err := c.GetConsensusGroup(*snapshotHash, types.SNAPSHOT_GID)
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
	for i := 0; i < len(voteDetails); i++ {
		topProducers[voteDetails[i].CurrentAddr] = struct{}{}
	}
	return topProducers
}
