package consensus

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	"math/rand"

	"sort"

	"github.com/vitelabs/go-vite/common/helper"
	"github.com/vitelabs/go-vite/common/types"
)

func TestFilterVotes(t *testing.T) {
	info := &membersInfo{genesisTime: time.Now(), memberCnt: 25, interval: 1, perCnt: 3, randCnt: 10, LowestLimit: helper.Big0}

	teller := newTeller(info, &chainRw{})
	votes := genVotes(10)
	testFilterVotes(t, teller, votes)
	testFilterVotes(t, teller, genVotes(21))
	testFilterVotes(t, teller, genVotes(25))
	testFilterVotes(t, teller, genVotes(26))
	testFilterVotes(t, teller, genVotes(30))
	testFilterVotes(t, teller, genVotes(40))
	testFilterVotes(t, teller, genVotes(100))
}

func testFilterVotes(t *testing.T, teller *teller, votes []*Vote) {
	fVotes1 := teller.algo.filterVotes(votes)
	fVotes2 := teller.algo.filterVotes(votes)

	result := ""
	for k, v := range fVotes1 {
		result += v.name + "\t"
		if fVotes2[k].name != v.name {
			t.Error("errr for filterVotes")
		}
	}
	println(result)
}

func TestShuffleVotes(t *testing.T) {

	info := &membersInfo{genesisTime: time.Now(), memberCnt: 25, interval: 1, perCnt: 3, randCnt: 10, LowestLimit: helper.Big0}

	teller := newTeller(info, &chainRw{})

	votes := genVotes(11)
	shuffleVotes1 := teller.algo.shuffleVotes(votes)
	shuffleVotes2 := teller.algo.shuffleVotes(votes)

	result := ""
	for k, v := range shuffleVotes1 {
		result += v.name + "\t"
		if shuffleVotes2[k].name != v.name {
			t.Error("errr for shuffleVotes")
		}
	}
	println(result)
}

func TestGenPlans(t *testing.T) {
	info := &membersInfo{genesisTime: time.Now(), memberCnt: 25, interval: 1, perCnt: 3, randCnt: 10, LowestLimit: helper.Big0}

	teller := newTeller(info, &chainRw{})

	votes := genVotes(20)
	testGenPlan(t, teller, votes, 60)
	testGenPlan(t, teller, genVotes(25), 75)
	testGenPlan(t, teller, genVotes(26), 75)

}
func testGenPlan(t *testing.T, teller *teller, votes []*Vote, expectedCnt int) {
	votes = teller.algo.filterVotes(votes)

	plans := teller.info.genPlan(teller.info.time2Index(time.Now()), teller.convertToAddress(votes))
	for _, v := range plans.Plans {
		print(v.STime.String() + "\t" + v.Member.String())
		println()
	}
	if len(plans.Plans) != expectedCnt {
		t.Error("size of plan is error")
	}
	println(plans.ETime.String())
}

func genVotes(n int) []*Vote {
	var votes []*Vote
	for i := 0; i < n; i++ {
		vote := &Vote{name: strconv.Itoa(i) + "", balance: big.NewInt(200000), addr: mockAddress(i)}
		votes = append(votes, vote)
	}
	return votes
}

func mockAddress(i int) types.Address {
	base := "00000000000000000fff" + fmt.Sprintf("%0*d", 6, i) + "fff00000000000"
	bytes, _ := hex.DecodeString(base)
	addresses, _ := types.BytesToAddress(bytes)
	return addresses
}
func TestMockAddress(t *testing.T) {
	println(mockAddress(21).String())
}

func TestByBalance_Len(t *testing.T) {
	random := rand.New(rand.NewSource(2))
	perm := random.Perm(20)
	sort.Ints(perm)
	for _, v := range perm {
		println(v)
	}
}

func TestTime(t *testing.T) {
	sTime := time.Now()
	etime := sTime.Add(time.Second * 20)
	println(sTime.Unix() == etime.Unix())
}
