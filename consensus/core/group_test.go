package core

import (
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

func TestNewGroupInfo(t *testing.T) {
	genesis := time.Now()
	info := NewGroupInfo(genesis, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               3,
		PerCount:               1,
		RandCount:              2,
		RandRank:               100,
		Repeat:                 1,
		CheckLevel:             1,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	})
	idx := info.Time2Index(genesis)
	assert.Equal(t, uint64(0), uint64(idx))

	stime, etime := info.Index2Time(0)
	assert.Equal(t, genesis, stime)
	assert.Equal(t, genesis.Add(time.Second*75), etime)

	stime, etime = info.Index2Time(1)
	assert.Equal(t, genesis.Add(time.Second*75), stime)
	assert.Equal(t, genesis.Add(time.Second*150), etime)
}

func TestGroupInfo_Time2Index(t *testing.T) {
	now := time.Now()
	info := NewGroupInfo(now, types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, Gid: types.SNAPSHOT_GID, PerCount: 3, Repeat: 1})

	index := info.Time2Index(now)
	assert.Equal(t, uint64(0), index)

	index = info.Time2Index(now.Add(time.Second))
	assert.Equal(t, uint64(0), index)

	index = info.Time2Index(now.Add(6 * time.Second))
	assert.Equal(t, uint64(0), index)

	index = info.Time2Index(now.Add(74 * time.Second))
	assert.Equal(t, uint64(0), index)

	index = info.Time2Index(now.Add(75 * time.Second))
	assert.Equal(t, uint64(1), index)

	index = info.Time2Index(now.Add(77 * time.Second))
	assert.Equal(t, uint64(1), index)

	index = info.Time2Index(now.Add(150 * time.Second))
	assert.Equal(t, uint64(2), index)

	N := uint64(100)
	for i := uint64(0); i < N; i++ {
		stime, etime := info.Index2Time(i)
		assert.Equal(t, now.Add(time.Second*time.Duration(25*3*i)), stime)
		assert.Equal(t, now.Add(time.Second*time.Duration(25*3*(i+1))), etime)
	}
}

func genAddress(n int) []types.Address {
	var result []types.Address
	for i := 0; i < n; i++ {
		result = append(result, common.MockAddress(i))
	}

	return result
}

func TestGroupInfo_GenPlan(t *testing.T) {
	now := time.Now()
	info := NewGroupInfo(now, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              10,
		Interval:               6,
		PerCount:               3,
		RandCount:              0,
		RandRank:               100,
		Repeat:                 1,
		CheckLevel:             0,
		CountingTokenId:        types.TokenTypeId{},
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	})

	var n = uint64(10)
	plans := info.GenPlanByAddress(0, genAddress(int(n)))
	assert.Equal(t, 30, len(plans))

	plans = info.GenPlanByAddress(0, genAddress(11))
	assert.Equal(t, 0, len(plans))
}
