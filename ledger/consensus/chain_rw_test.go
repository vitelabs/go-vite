package consensus

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/vitelabs/go-vite/v2/common/config"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/pool/lock"
	"github.com/vitelabs/go-vite/v2/ledger/test_tools"
	"github.com/vitelabs/go-vite/v2/log15"
)

func GetConsensusGroupList() ([]*types.ConsensusGroupInfo, error) {
	info := &types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              10,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		Repeat:                 1,
		CheckLevel:             0,
		CountingTokenId:        types.TokenTypeId{},
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		StakeAmount:            nil,
		ExpirationHeight:       0,
	}

	return []*types.ConsensusGroupInfo{info}, nil
}

func Test_chainRw(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, nil)
	defer test_tools.ClearChain(c, tempDir)
}

func TestChainRw_GetMemberInfo(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, nil)
	defer test_tools.ClearChain(c, tempDir)

	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	db := NewDb(t, tempDir)
	defer ClearDb(t, tempDir)

	mch := NewMockChain(ctrl)
	genesisBlock := &ledger.SnapshotBlock{Height: uint64(1), Timestamp: &simpleGenesis}
	genesisBlock.ComputeHash()
	mch.EXPECT().GetLatestSnapshotBlock().Return(genesisBlock).AnyTimes()
	mch.EXPECT().GetGenesisSnapshotBlock().Return(genesisBlock).AnyTimes()
	mch.EXPECT().NewDb(gomock.Any()).Return(db, nil).MaxTimes(1)
	infos, err := GetConsensusGroupList()
	mch.EXPECT().GetConsensusGroupList(genesisBlock.Hash).Return(infos, err).MaxTimes(1)

	rw := newChainRw(mch, log15.New("unittest", "chainrw"), &lock.EasyImpl{})
	block := rw.GetLatestSnapshotBlock()
	assert.Equal(t, genesisBlock.Timestamp, block.Timestamp)
	groupInfo, err := rw.GetMemberInfo(types.SNAPSHOT_GID)
	assert.Nil(t, err)
	assert.Equal(t, groupInfo.PlanInterval, uint64(30))
	stime, _ := groupInfo.Index2Time(0)
	assert.Equal(t, stime, simpleGenesis)
}

func TestChainRw_GetMemberInfo2(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, config.MockGenesis())
	defer test_tools.ClearChain(c, tempDir)

	genesis := c.GetGenesisSnapshotBlock()
	infos, err := c.GetConsensusGroupList(genesis.Hash)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	assert.Equal(t, 2, len(infos))
	for _, v := range infos {
		if v.Gid == types.SNAPSHOT_GID {
			assert.Equal(t, v.CheckLevel, uint8(0))
			assert.Equal(t, v.Repeat, uint16(1))
		} else if v.Gid == types.DELEGATE_GID {
			assert.Equal(t, v.CheckLevel, uint8(1))
			assert.Equal(t, v.Repeat, uint16(48))
		} else {
			t.FailNow()
		}
	}
}

func TestChainRw_GenLruKey(t *testing.T) {
	gid := types.SNAPSHOT_GID
	hash := types.Hash{}

	rw := chainRw{}
	key := rw.genLruKey(gid, hash)

	result := make(map[interface{}]string)
	result[key] = "1"

	for k, v := range key {
		println(k, v)
	}
}
