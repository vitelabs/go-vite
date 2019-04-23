package consensus

import (
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/config/gen"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
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
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	}

	return []*types.ConsensusGroupInfo{info}, nil
}

func testDataDir() string {
	return "testdata-consensus"
}

func prepareChain() chain.Chain {
	clearChain(nil)
	c := chain.NewChain(testDataDir(), nil, config_gen.MakeGenesisConfig(""))

	err := c.Init()
	if err != nil {
		panic(err)
	}
	err = c.Start()
	if err != nil {
		panic(err)
	}
	return c
}

func clearChain(c chain.Chain) {
	if c != nil {
		c.Stop()
	}
	err := os.RemoveAll(testDataDir())
	if err != nil {
		panic(err)
	}
}

func Test_chainRw(t *testing.T) {
	c := prepareChain()
	defer clearChain(c)

	//log := log15.New("unittest", "chainrw")
	//rw := newChainRw(c, log)
	//rw.initArray(nil)
}

func TestChainRw_GetMemberInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	dir := "testdata-consensus"
	db := NewDb(t, dir)
	defer ClearDb(t, dir)

	mch := NewMockChain(ctrl)
	genesisBlock := &ledger.SnapshotBlock{Height: uint64(1), Timestamp: &simpleGenesis}
	genesisBlock.ComputeHash()
	mch.EXPECT().GetLatestSnapshotBlock().Return(genesisBlock).AnyTimes()
	mch.EXPECT().GetGenesisSnapshotBlock().Return(genesisBlock).AnyTimes()
	mch.EXPECT().NewDb(gomock.Any()).Return(db, nil).MaxTimes(1)
	infos, err := GetConsensusGroupList()
	mch.EXPECT().GetConsensusGroupList(genesisBlock.Hash).Return(infos, err).MaxTimes(1)

	rw := newChainRw(mch, log15.New("unittest", "chainrw"))
	block := rw.GetLatestSnapshotBlock()
	assert.Equal(t, genesisBlock.Timestamp, block.Timestamp)
	groupInfo, err := rw.GetMemberInfo(types.SNAPSHOT_GID)
	assert.Nil(t, err)
	assert.Equal(t, groupInfo.PlanInterval, uint64(30))
	stime, _ := groupInfo.Index2Time(0)
	assert.Equal(t, stime, simpleGenesis)
}

func TestChainRw_GetMemberInfo2(t *testing.T) {
	c := NewChain(t, UnitTestDir, GenesisJson)
	defer ClearChain(UnitTestDir)
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
