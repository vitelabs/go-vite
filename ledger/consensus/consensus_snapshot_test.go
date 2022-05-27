package consensus

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/vitelabs/go-vite/v2/common"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
	"github.com/vitelabs/go-vite/v2/ledger/pool/lock"
	"github.com/vitelabs/go-vite/v2/ledger/test_tools"
	"github.com/vitelabs/go-vite/v2/log15"
)

func TestSnapshotCs_ElectionIndex(t *testing.T) {
	c, tempDir := test_tools.NewTestChainInstance(t.Name(), true, nil)
	defer test_tools.ClearChain(c, tempDir)

	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()
	mock_chain := NewMockChain(ctrl)

	mock_chain.EXPECT().GetGenesisSnapshotBlock().Return(&ledger.SnapshotBlock{
		Timestamp: &simpleGenesis,
	})
	db := NewDb(t, tempDir)
	defer ClearDb(t, tempDir)
	mock_chain.EXPECT().NewDb(gomock.Any()).Return(db, nil)

	group := types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              3,
		Interval:               1,
		PerCount:               3,
		RandCount:              1,
		RandRank:               100,
		Repeat:                 1,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		StakeAmount:            nil,
		ExpirationHeight:       0,
	}

	info := core.NewGroupInfo(simpleGenesis, group)

	b1 := GenSnapshotBlock(1, "3fc5224e59433bff4f48c83c0eb4edea0e4c42ea697e04cdec717d03e50d5200", types.Hash{}, simpleGenesis)

	mock_chain.EXPECT().GetConsensusGroupList(b1.Hash).Return([]*types.ConsensusGroupInfo{&group}, nil)
	mock_chain.EXPECT().GetLatestSnapshotBlock().Return(b1)
	rw := newChainRw(mock_chain, log15.New(), &lock.EasyImpl{})

	cs := newSnapshotCs(rw, log15.New())

	voteTime := cs.GenProofTime(0)
	mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&voteTime)).Return(b1, nil)
	registers := []*types.Registration{{
		Name:                  "s1",
		BlockProducingAddress: common.MockAddress(0),
		StakeAddress:          common.MockAddress(0),
		Amount:                nil,
		ExpirationHeight:      0,
		RewardTime:            0,
		RevokeTime:            0,
		HisAddrList:           nil,
	}, {
		Name:                  "s2",
		BlockProducingAddress: common.MockAddress(1),
		StakeAddress:          common.MockAddress(1),
		Amount:                nil,
		ExpirationHeight:      0,
		RewardTime:            0,
		RevokeTime:            0,
		HisAddrList:           nil,
	}, {
		Name:                  "s3",
		BlockProducingAddress: common.MockAddress(2),
		StakeAddress:          common.MockAddress(2),
		Amount:                nil,
		ExpirationHeight:      0,
		RewardTime:            0,
		RevokeTime:            0,
		HisAddrList:           nil,
	}}
	votes := []*types.VoteInfo{
		{
			VoteAddr: common.MockAddress(11),
			SbpName:  "s1",
		},
		{
			VoteAddr: common.MockAddress(12),
			SbpName:  "s1",
		}, {
			VoteAddr: common.MockAddress(21),
			SbpName:  "s2",
		}, {
			VoteAddr: common.MockAddress(31),
			SbpName:  "s3",
		}, {
			VoteAddr: common.MockAddress(32),
			SbpName:  "s3",
		}}

	S1balances := make(map[types.Address]*big.Int)
	S1balances[common.MockAddress(11)] = big.NewInt(11)
	S1balances[common.MockAddress(12)] = big.NewInt(12)
	S2balances := make(map[types.Address]*big.Int)
	S2balances[common.MockAddress(21)] = big.NewInt(21)

	S3balances := make(map[types.Address]*big.Int)
	S3balances[common.MockAddress(31)] = big.NewInt(31)
	S3balances[common.MockAddress(32)] = big.NewInt(32)

	mock_chain.EXPECT().GetRegisterList(b1.Hash, types.SNAPSHOT_GID).Return(registers, nil)
	mock_chain.EXPECT().GetVoteList(b1.Hash, types.SNAPSHOT_GID).Return(votes, nil)
	mock_chain.EXPECT().GetConfirmedBalanceList([]types.Address{common.MockAddress(11), common.MockAddress(12)}, ledger.ViteTokenId, b1.Hash).Return(S1balances, nil)
	mock_chain.EXPECT().GetConfirmedBalanceList([]types.Address{common.MockAddress(21)}, ledger.ViteTokenId, b1.Hash).Return(S2balances, nil)
	mock_chain.EXPECT().GetConfirmedBalanceList([]types.Address{common.MockAddress(31), common.MockAddress(32)}, ledger.ViteTokenId, b1.Hash).Return(S3balances, nil)
	mock_chain.EXPECT().GetRandomSeed(b1.Hash, 25).Return(uint64(105))

	result, err := cs.ElectionIndex(0)
	assert.NoError(t, err)

	assert.NotNil(t, result)

	assert.Equal(t, simpleGenesis, result.STime)
	assert.Equal(t, simpleGenesis.Add(time.Duration(info.PlanInterval)*time.Second), result.ETime)
	assert.Equal(t, uint64(0), result.Index)
	assert.Equal(t, 9, len(result.Plans))
	for k, v := range result.Plans {
		assert.Equal(t, simpleGenesis.Add(time.Duration(int64(k)*info.Interval)*time.Second), v.STime)
		assert.Equal(t, v.STime.Add(time.Second), v.ETime)
		assert.Equal(t, common.MockAddress(k/int(info.PerCount)%int(info.NodeCount)), v.Member, fmt.Sprintf("%d", k))
	}
}

func TestSnapshotCs_Tools(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect ledger data.")

	dir := "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_1/devdata"
	c, err := test_tools.NewChainInstanceFromDir(dir, false, GenesisJson)
	if err != nil {
		assert.FailNow(t, err.Error())
	}
	rw := newChainRw(c, log15.New(), &lock.EasyImpl{})
	cs := newSnapshotCs(rw, log15.New())

	rw.init(cs)

	result, err := cs.ElectionIndex(251694)
	if err != nil {
		assert.FailNow(t, err.Error())
	}

	t.Log(fmt.Sprintf("stime:%s, etime:%s", result.STime, result.ETime))
	for _, v := range result.Plans {
		t.Log(v.STime, v.ETime, v.Member)
		t.Log(cs.VerifyProducer(v.Member, v.STime))
	}
}

func TestNumber(t *testing.T) {
	i := uint64(25)/3*2 + 1
	assert.Equal(t, uint64(17), i)
}

func TestChainSnapshotAAAA(t *testing.T) {
	t.Skip("Skipped by default. This test can be used to inspect ledger data.")

	dir := "/Users/jie/Library/GVite/maindata"

	c, err := test_tools.NewChainInstanceFromDir(dir, false, "")
	if err != nil {
		t.Fatal(err)
		return
	}

	rw := newChainRw(c, log15.New(), &lock.EasyImpl{})
	cs := newSnapshotCs(rw, log15.New())

	rw.init(cs)

	point, err := rw.dayPoints.GetByIndex(95)
	if err != nil {
		panic(err)
	}
	fmt.Println(point.Hash, point.Votes.Total)

	votes, err := cs.rw.rw.GetVoteList(point.Hash, types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}
	for _, v := range votes {
		if v.SbpName == "N4Q.org" {
			fmt.Println(v.VoteAddr)
		}
	}
}
