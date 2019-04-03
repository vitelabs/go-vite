package consensus

import (
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
)

type electionResult struct {
	Plans []*core.MemberPlan
	STime time.Time
	ETime time.Time
	Index uint64
}

func genElectionResult(info *core.GroupInfo, index uint64, members []types.Address) *electionResult {
	self := &electionResult{}
	self.STime = info.GenSTime(index)
	self.ETime = info.GenETime(index)
	self.Plans = info.GenPlanByAddress(index, members)
	self.Index = index
	return self
}
