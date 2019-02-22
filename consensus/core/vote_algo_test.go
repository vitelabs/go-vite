package core

import (
	"math/big"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/common/types"
)

func TestAlgo_FilterVotes(t *testing.T) {

	now := time.Unix(1541640427, 0)
	info := NewGroupInfo(now, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	})
	ag := NewAlgo(info)
	printResult(ag, 1000, 25)
	printResult(ag, 100000, 26)
	printResult(ag, 100000, 27)
	printResult(ag, 100000, 28)
	printResult(ag, 100000, 28)
	printResult(ag, 100000, 99)
	printResult(ag, 100000, 100)
	printResult(ag, 100000, 101)

}

func TestAlgo_FilterVotes2(t *testing.T) {

	now := time.Unix(1541640427, 0)
	info := NewGroupInfo(now, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	})
	ag := NewAlgo(info)
	var votes []*Vote
	for i := 0; i < 100; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(i))})
	}
	//result := make(map[string]uint64)
	hashH := &ledger.HashHeight{Height: 1}
	context := NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(nil))
	actual := ag.FilterVotes(context)
	for _, v := range actual {
		print("\""+v.Name+"\"", ",")
	}
	expected := []string{"wj_16", "wj_2", "wj_10", "wj_8", "wj_7", "wj_3", "wj_12", "wj_22", "wj_24", "wj_11", "wj_13", "wj_20", "wj_18", "wj_19", "wj_17", "wj_4", "wj_1", "wj_15", "wj_0", "wj_9", "wj_5", "wj_14", "wj_21", "wj_50", "wj_72"}
	for _, v := range expected {
		print("\""+v+"\"", ",")
	}
	for i, v := range actual {
		if v.Name != expected[i] {
			t.Error("fail.")
		}
	}

}

func printResult(ag *algo, total uint64, cnt int) {
	println("-----------------------", strconv.FormatUint(total, 10), strconv.Itoa(cnt), "--------------------------------------------------")
	var votes []*Vote
	for i := 0; i < cnt; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(i))})
	}
	result := make(map[string]uint64)
	for j := uint64(0); j < total; j++ {
		hashH := &ledger.HashHeight{Height: j}
		tmp := ag.FilterVotes(NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(nil)))
		for _, v := range tmp {
			result[v.Name] = result[v.Name] + 1
		}
	}
	sort.Sort(ByBalance(votes))
	for _, v := range votes {
		vv := result[v.Name]
		println(v.Name, (vv*10000)/total)
	}

	println("-------------------------------------------------------------------------")
}

func TestAlgo_FilterVotes3(t *testing.T) {

	now := time.Unix(1541640427, 0)
	info := NewGroupInfo(now, types.ConsensusGroupInfo{
		Gid:                    types.SNAPSHOT_GID,
		NodeCount:              25,
		Interval:               1,
		PerCount:               3,
		RandCount:              2,
		RandRank:               100,
		CountingTokenId:        ledger.ViteTokenId,
		RegisterConditionId:    0,
		RegisterConditionParam: nil,
		VoteConditionId:        0,
		VoteConditionParam:     nil,
		Owner:                  types.Address{},
		PledgeAmount:           nil,
		WithdrawHeight:         0,
	})
	ag := NewAlgo(info)
	var votes []*Vote
	for i := 0; i < 100; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(100 - i))})
	}
	//result := make(map[string]uint64)
	hashH := &ledger.HashHeight{Height: 1}
	actual := ag.FilterVotes(NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(nil)))
	sort.Sort(ByBalance(actual))
	for _, v := range actual {
		println("\""+v.Name+"\"", v.Balance.String(), ",")
	}

}
