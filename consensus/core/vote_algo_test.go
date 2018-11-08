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

func printResult(ag *algo, total uint64, cnt int) {
	println("-----------------------", strconv.FormatUint(total, 10), strconv.Itoa(cnt), "--------------------------------------------------")
	var votes []*Vote
	for i := 0; i < cnt; i++ {
		votes = append(votes, &Vote{Name: "wj_" + strconv.Itoa(i), Balance: big.NewInt(int64(i))})
	}
	result := make(map[string]uint64)
	for j := uint64(0); j < total; j++ {
		hashH := &ledger.HashHeight{Height: j}
		tmp := ag.FilterVotes(votes, hashH)
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
