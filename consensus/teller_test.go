package consensus

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

func TestFilterVotes(t *testing.T) {

	info := core.NewGroupInfo(time.Now(), types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, PerCount: 3, RandCount: 0})

	teller := newTeller(info, &chainRw{}, log15.New("module", "unitTest"))
	votes := genVotes(10)
	testFilterVotes(t, teller, votes)
	testFilterVotes(t, teller, genVotes(21))
	testFilterVotes(t, teller, genVotes(25))
	testFilterVotes(t, teller, genVotes(26))
	testFilterVotes(t, teller, genVotes(30))
	testFilterVotes(t, teller, genVotes(40))
	testFilterVotes(t, teller, genVotes(100))
}

func TestMembersInfo(t *testing.T) {
	info := core.NewGroupInfo(time.Now(), types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, PerCount: 3, RandCount: 0, Gid: types.SNAPSHOT_GID})

	now := time.Now()
	index := info.Time2Index(now)

	sTime := info.GenSTime(index - 1)
	fmt.Println(sTime.Sub(now).String(), now, sTime)
}

func testFilterVotes(t *testing.T, teller *teller, votes []*core.Vote) {
	hash, _ := types.HexToHash("706b00a2ae1725fb5d90b3b7a76d76c922eb075be485749f987af7aa46a66785")
	testH := &ledger.HashHeight{Hash: hash, Height: 1}
	fVotes1 := teller.algo.FilterVotes(votes, testH)
	fVotes2 := teller.algo.FilterVotes(votes, testH)

	result := ""
	for k, v := range fVotes1 {
		result += v.Name + "\t"
		if fVotes2[k].Name != v.Name {
			t.Error("errr for filterVotes")
		}
	}
	println(result)
}

func TestShuffleVotes(t *testing.T) {

	info := core.NewGroupInfo(time.Now(), types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, PerCount: 3, RandCount: 0, Gid: types.DELEGATE_GID})

	teller := newTeller(info, &chainRw{}, log15.New("module", "unitTest"))

	votes := genVotes(11)
	shuffleVotes1 := teller.algo.ShuffleVotes(votes, nil)
	shuffleVotes2 := teller.algo.ShuffleVotes(votes, nil)

	result := ""
	for k, v := range shuffleVotes1 {
		result += v.Name + "\t"
		if shuffleVotes2[k].Name != v.Name {
			t.Error("errr for shuffleVotes")
		}
	}
	println(result)
}

func TestGenPlans(t *testing.T) {
	info := core.NewGroupInfo(time.Now(), types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, PerCount: 3, RandCount: 0, Gid: types.DELEGATE_GID})

	teller := newTeller(info, &chainRw{}, log15.New("module", "unitTest"))

	votes := genVotes(20)
	testGenPlan(t, teller, votes, 60)
	testGenPlan(t, teller, genVotes(25), 75)
	testGenPlan(t, teller, genVotes(26), 75)

}
func testGenPlan(t *testing.T, teller *teller, votes []*core.Vote, expectedCnt int) {
	votes = teller.algo.FilterVotes(votes, nil)

	plans := teller.info.GenPlanByAddress(teller.info.Time2Index(time.Now()), core.ConvertVoteToAddress(votes))
	for _, v := range plans {
		print(v.STime.String() + "\t" + v.Member.String())
		println()
	}
	if len(plans) != expectedCnt {
		t.Error("size of plan is error")
	}
}

func genVotes(n int) []*core.Vote {
	var votes []*core.Vote
	for i := 0; i < n; i++ {
		vote := &core.Vote{Name: strconv.Itoa(i) + "", Balance: big.NewInt(200000 + int64(i)), Addr: mockAddress(i)}
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

func TestSlice(t *testing.T) {
	perm := rand.Perm(10)

	r := perm[0:5]

	for _, v := range r {
		println(v)
	}
}
