package consensus

import (
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/types"
)

type membersInfo struct {
	genesisTime     time.Time
	memberCnt       int32 // Number of producer
	interval        int32 // unit: second, time interval at which the block is generated
	perCnt          int32 // Number of blocks generated per node
	randCnt         int32
	randRange       int32
	LowestLimit     *big.Int
	seed            *big.Int
	countingTokenId types.TokenTypeId
}

type memberPlan struct {
	STime  time.Time
	ETime  time.Time
	Member types.Address
}

type electionResult struct {
	Plans  []*memberPlan
	STime  time.Time
	ETime  time.Time
	Index  int32
	Hash   types.Hash
	Height uint64
}

func (self *membersInfo) genPlan(index int32, members []types.Address) *electionResult {
	result := electionResult{}
	sTime := self.genSTime(index)
	result.STime = sTime

	var plans []*memberPlan
	for _, member := range members {
		for i := int32(0); i < self.perCnt; i++ {
			etime := sTime.Add(time.Duration(self.interval) * time.Second)
			plan := memberPlan{STime: sTime, ETime: etime, Member: member}
			plans = append(plans, &plan)
			sTime = etime
		}
	}
	result.ETime = self.genETime(index)
	result.Plans = plans
	result.Index = index
	return &result
}

func (self *membersInfo) time2Index(t time.Time) int32 {
	subSec := int64(t.Sub(self.genesisTime).Seconds())
	i := subSec / int64((self.interval * self.memberCnt * self.perCnt))
	return int32(i)
}
func (self *membersInfo) genSTime(index int32) time.Time {
	planInterval := self.interval * self.memberCnt * self.perCnt
	return self.genesisTime.Add(time.Duration(planInterval*index) * time.Second)
}
func (self *membersInfo) genETime(index int32) time.Time {
	planInterval := self.interval * self.memberCnt * self.perCnt
	return self.genesisTime.Add(time.Duration(planInterval*index+1) * time.Second)
}
