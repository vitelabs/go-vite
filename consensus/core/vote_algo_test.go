package core

import (
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/magiconair/properties/assert"

	"github.com/vitelabs/go-vite/common"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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
		StakeAmount:            nil,
		ExpirationHeight:       0,
	})
	ag := NewAlgo(info)
	printResult(ag, 1000, 25)
	//printResult(ag, 100000, 26)
	//printResult(ag, 100000, 27)
	//printResult(ag, 100000, 28)
	//printResult(ag, 100000, 28)
	//printResult(ag, 100000, 99)
	//printResult(ag, 100000, 100)
	//printResult(ag, 100000, 101)

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
		StakeAmount:            nil,
		ExpirationHeight:       0,
	})
	ag := NewAlgo(info)
	var votes []*Vote
	for i := 0; i < 100; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(i))})
	}
	//result := make(map[string]uint64)
	hashH := &ledger.HashHeight{Height: 1}
	context := NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(0))
	actual := ag.FilterVotes(context)
	for _, v := range actual {
		print("\""+v.Name+"\"", ",")
	}
	println()
	expected := []string{"wj_99", "wj_98", "wj_97", "wj_96", "wj_95", "wj_94", "wj_92", "wj_91", "wj_90", "wj_89", "wj_88", "wj_87", "wj_86", "wj_85", "wj_84", "wj_83", "wj_82", "wj_81", "wj_80", "wj_79", "wj_78", "wj_77", "wj_75", "wj_49", "wj_27"}
	for _, v := range expected {
		print("\""+v+"\"", ",")
	}
	println()
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
		tmp := ag.FilterVotes(NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(0)))
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
		StakeAmount:            nil,
		ExpirationHeight:       0,
	})
	ag := NewAlgo(info)
	var votes []*Vote
	for i := 0; i < 100; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(100 - i))})
	}
	//result := make(map[string]uint64)
	hashH := &ledger.HashHeight{Height: 1}
	actual := ag.FilterVotes(NewVoteAlgoContext(votes, hashH, nil, NewSeedInfo(0)))
	sort.Sort(ByBalance(actual))
	for _, v := range actual {
		println("\""+v.Name+"\"", v.Balance.String(), ",")
	}

}

func TestAlgo_FilterBySuccessRate(t *testing.T) {
	a := &algo{}

	var groupA []*Vote
	for i := 10; i <= 35; i++ {
		vote := &Vote{
			Name:    fmt.Sprintf("s%d", i),
			Addr:    common.MockAddress(i),
			Balance: big.NewInt(10),
			Type:    nil,
		}
		groupA = append(groupA, vote)
	}

	var groupB []*Vote
	for i := 40; i < 50; i++ {
		vote := &Vote{
			Name:    fmt.Sprintf("s%d", i),
			Addr:    common.MockAddress(i),
			Balance: big.NewInt(10),
			Type:    nil,
		}
		groupB = append(groupB, vote)
	}

	sort.Sort(ByBalance(groupA))
	sort.Sort(ByBalance(groupB))

	successRate := make(map[types.Address]int32)

	for _, v := range groupA {
		successRate[v.Addr] = 1000000
	}
	for _, v := range groupB {
		successRate[v.Addr] = 800001
	}

	successRate[common.MockAddress(24)] = 0
	resultA, resultB := a.filterBySuccessRate(groupA, groupB, nil, successRate)

	nameA := []string{"s10", "s11", "s12", "s13", "s14", "s15", "s16", "s17", "s18", "s19", "s20", "s21", "s22", "s23", "s25", "s26", "s27", "s28", "s29", "s30", "s31", "s32", "s33", "s34", "s35", "s40"}
	for k, v := range resultA {
		assert.Equal(t, nameA[k], v.Name)
	}

	nameB := []string{"s42", "s43", "s44", "s45", "s46", "s47", "s48", "s49", "s24", "s41"}

	for k, v := range resultB {
		assert.Equal(t, nameB[k], v.Name)
	}

	successRate[common.MockAddress(23)] = 0
	resultA, resultB = a.filterBySuccessRate(groupA, groupB, nil, successRate)

	nameA = []string{"s10", "s11", "s12", "s13", "s14", "s15", "s16", "s17", "s18", "s19", "s20", "s21", "s22", "s25", "s26", "s27", "s28", "s29", "s30", "s31", "s32", "s33", "s34", "s35", "s41", "s40"}
	nameB = []string{"s42", "s43", "s44", "s45", "s46", "s47", "s48", "s49", "s24", "s23"}
	for k, v := range resultA {
		assert.Equal(t, nameA[k], v.Name)
	}

	for k, v := range resultB {
		assert.Equal(t, nameB[k], v.Name)
	}

	successRate[common.MockAddress(22)] = 0
	successRate[common.MockAddress(40)] = 0
	resultA, resultB = a.filterBySuccessRate(groupA, groupB, nil, successRate)

	nameA = []string{"s10", "s11", "s12", "s13", "s14", "s15", "s16", "s17", "s18", "s19", "s20", "s21", "s25", "s26", "s27", "s28", "s29", "s30", "s31", "s32", "s33", "s34", "s35", "s22", "s42", "s41"}
	nameB = []string{"s43", "s44", "s45", "s46", "s47", "s48", "s49", "s40", "s24", "s23"}
	for k, v := range resultA {
		assert.Equal(t, nameA[k], v.Name)
	}

	for k, v := range resultB {
		assert.Equal(t, nameB[k], v.Name)
	}
	for _, v := range resultA {
		fmt.Printf("%s \t", v.Name)
	}
	fmt.Println()

	for _, v := range resultB {
		fmt.Printf("%s \t", v.Name)
	}
	fmt.Println()
}
