package core

import (
	"fmt"
	"math/big"
	"time"

	"github.com/vitelabs/go-vite/common/types"
)

type MemberPlan struct {
	STime  time.Time
	ETime  time.Time
	Member types.Address
	Name   string
}

type GroupInfo struct {
	TimeIndex
	types.ConsensusGroupInfo

	GenesisTime  time.Time
	seed         *big.Int
	PlanInterval uint64
}

func NewGroupInfo(genesisTime time.Time, info types.ConsensusGroupInfo) *GroupInfo {
	groupInfo := &GroupInfo{
		ConsensusGroupInfo: info,
		GenesisTime:        genesisTime,
		seed:               new(big.Int).SetBytes(info.Gid.Bytes()),
	}

	groupInfo.PlanInterval = planInterval(groupInfo)
	groupInfo.TimeIndex = NewTimeIndex(genesisTime, time.Second*time.Duration(groupInfo.PlanInterval))
	return groupInfo
}

func planInterval(info *GroupInfo) uint64 {
	return uint64(info.Interval) * uint64(info.NodeCount) * uint64(info.PerCount) * uint64(info.Repeat)
}

func (self GroupInfo) GenPlan(index uint64, members []*Vote) []*MemberPlan {
	sTime, _ := self.Index2Time(index)
	var plans []*MemberPlan
	for _, member := range members {
		for i := int64(0); i < self.PerCount; i++ {
			etime := sTime.Add(time.Duration(self.Interval) * time.Second)
			plan := MemberPlan{STime: sTime, ETime: etime, Member: member.Addr, Name: member.Name}
			plans = append(plans, &plan)
			sTime = etime
		}
	}
	return plans
}

func (self GroupInfo) GenPlanByAddress(index uint64, members []types.Address) []*MemberPlan {
	sTime, _ := self.Index2Time(index)
	var plans []*MemberPlan

	for j := uint16(0); j < self.Repeat; j++ {
		for _, member := range members {
			for i := int64(0); i < self.PerCount; i++ {
				etime := sTime.Add(time.Duration(self.Interval) * time.Second)
				plan := MemberPlan{STime: sTime, ETime: etime, Member: member}
				plans = append(plans, &plan)
				sTime = etime
			}
		}
		if len(members) < int(self.NodeCount) {
			sTime = sTime.Add(time.Duration(self.Interval*self.PerCount*(int64(self.NodeCount)-int64(len(members)))) * time.Second)
		}
	}
	return plans
}

func (self GroupInfo) String() string {
	return fmt.Sprintf("genesisTime:%s, memberCnt:%d, interval:%d, perCnt:%d, randCnt:%d, randRange:%d, seed:%s, countingTokenId:%s",
		self.GenesisTime.String(), self.NodeCount, self.Interval, self.PerCount, self.RandCount, self.RandRank, self.seed, self.CountingTokenId.String())

}
