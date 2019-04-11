package consensus

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/ledger"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/log15"
)

func TestContractDposCs_ElectionIndexReader(t *testing.T) {
	dir := "/Users/jie/Documents/vite/src/github.com/vitelabs/cluster1/ledger_datas/ledger_1/devdata"
	genesisJson := GenesisJson
	c, err := NewChainInstanceFromDir(dir, false, genesisJson)

	assert.NoError(t, err)

	rw := newChainRw(c, log15.New())
	groupInfo, err := rw.GetMemberInfo(types.DELEGATE_GID)
	assert.NoError(t, err)

	cs := newContractDposCs(groupInfo, rw, log15.New())

	proof := newRollbackProof(rw.rw)
	index := cs.Time2Index(time.Now())
	for i := index; i > 0; i-- {
		_, etime := cs.Index2Time(i)
		hashes, err := proof.ProofHash(etime)
		if err != nil {
			t.Error(err)
			assert.FailNow(t, err.Error())
		}
		if rw.rw.IsGenesisSnapshotBlock(hashes) {
			break
		}

		result, err := cs.ElectionIndex(i)
		assert.NoError(t, err)

		vs := "\n"
		for _, v := range result.Plans {
			vs += fmt.Sprintf("\t\t[%s][%s][%s]\n", v.STime, v.ETime, v.Member)
		}
		t.Log(i, len(result.Plans), hashes, result.STime, result.ETime, vs)

	}

}

func TestContractDposCs_ElectionIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()
	mock_chain := NewMockChain(ctrl)

	mock_chain.EXPECT().GetGenesisSnapshotBlock().Return(&ledger.SnapshotBlock{
		Timestamp: &simpleGenesis,
	})
	db := NewDb(t, UnitTestDir)
	defer ClearDb(t, UnitTestDir)
	mock_chain.EXPECT().NewDb(gomock.Any()).Return(db, nil)

	group := types.ConsensusGroupInfo{
		Gid:                    types.DELEGATE_GID,
		NodeCount:              2,
		Interval:               1,
		PerCount:               3,
		RandCount:              1,
		RandRank:               100,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	}

	info := core.NewGroupInfo(simpleGenesis, group)

	b1 := GenSnapshotBlock(1, "3fc5224e59433bff4f48c83c0eb4edea0e4c42ea697e04cdec717d03e50d5200", types.Hash{}, simpleGenesis)

	rw := newChainRw(mock_chain, log15.New())

	cs := newContractDposCs(info, rw, log15.New())

	voteTime := cs.GenProofTime(0)
	mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&voteTime)).Return(b1, nil)
	registers := []*types.Registration{{
		Name:           "s1",
		NodeAddr:       common.MockAddress(0),
		PledgeAddr:     common.MockAddress(0),
		Amount:         nil,
		WithdrawHeight: 0,
		RewardTime:     0,
		CancelTime:     0,
		HisAddrList:    nil,
	}, {
		Name:           "s2",
		NodeAddr:       common.MockAddress(1),
		PledgeAddr:     common.MockAddress(1),
		Amount:         nil,
		WithdrawHeight: 0,
		RewardTime:     0,
		CancelTime:     0,
		HisAddrList:    nil,
	}}
	votes := []*types.VoteInfo{
		{
			VoterAddr: common.MockAddress(11),
			NodeName:  "s1",
		},
		{
			VoterAddr: common.MockAddress(12),
			NodeName:  "s1",
		}, {
			VoterAddr: common.MockAddress(21),
			NodeName:  "s2",
		}}

	S1balances := make(map[types.Address]*big.Int)
	S1balances[common.MockAddress(11)] = big.NewInt(11)
	S1balances[common.MockAddress(12)] = big.NewInt(12)
	S2balances := make(map[types.Address]*big.Int)
	S2balances[common.MockAddress(21)] = big.NewInt(21)

	mock_chain.EXPECT().GetRegisterList(b1.Hash, types.DELEGATE_GID).Return(registers, nil)
	mock_chain.EXPECT().GetVoteList(b1.Hash, types.DELEGATE_GID).Return(votes, nil)
	mock_chain.EXPECT().GetConfirmedBalanceList([]types.Address{common.MockAddress(11), common.MockAddress(12)}, ledger.ViteTokenId, b1.Hash).Return(S1balances, nil)
	mock_chain.EXPECT().GetConfirmedBalanceList([]types.Address{common.MockAddress(21)}, ledger.ViteTokenId, b1.Hash).Return(S2balances, nil)
	mock_chain.EXPECT().GetRandomSeed(b1.Hash, 25).Return(uint64(105))

	result, err := cs.ElectionIndex(0)
	assert.NoError(t, err)

	assert.NotNil(t, result)

	assert.Equal(t, simpleGenesis, result.STime)
	assert.Equal(t, simpleGenesis.Add(time.Duration(info.PlanInterval)*time.Second), result.ETime)
	assert.Equal(t, uint64(0), result.Index)
	for k, v := range result.Plans {
		assert.Equal(t, simpleGenesis.Add(time.Duration(int64(k)*info.Interval)*time.Second), v.STime)
		assert.Equal(t, v.STime.Add(time.Second), v.ETime)
		assert.Equal(t, common.MockAddress(k/int(info.PerCount)%int(info.NodeCount)), v.Member, fmt.Sprintf("%d", k))
	}
}
