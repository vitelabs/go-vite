package core

import (
	"testing"
	"time"

	"github.com/magiconair/properties/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

var genesis = time.Unix(1553849738, 0)

func TestNewGroupInfo(t *testing.T) {
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
