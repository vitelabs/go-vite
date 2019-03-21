package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
)

type electionResult struct {
	Plans     []*core.MemberPlan
	STime     time.Time
	ETime     time.Time
	Index     uint64
	Hash      types.Hash
	Height    uint64
	Timestamp time.Time
}

func genElectionResult(info *core.GroupInfo, index uint64, members []types.Address, hashH *ledger.SnapshotBlock) *electionResult {
	self := &electionResult{}
	self.STime = info.GenSTime(index)
	self.ETime = info.GenETime(index)
	self.Plans = info.GenPlanByAddress(index, members)
	self.Index = index
	self.Hash = hashH.Hash
	self.Height = hashH.Height
	self.Timestamp = *hashH.Timestamp
	return self
}
