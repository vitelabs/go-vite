package chain

import (
	"fmt"

	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
)

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
