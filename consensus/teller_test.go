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

	"github.com/stretchr/testify/assert"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

func TestFilterVotes(t *testing.T) {

	info := core.NewGroupInfo(time.Now(), types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, PerCount: 3, RandCount: 0})

	seeds := make(map[types.Address]uint64)
	al := core.NewAlgo(info)
	votes := genVotes(10)
	assert.Equal(t, testFilterVotes(t, al, votes, seeds), "9,6,4,3,5,0,2,8,7,1,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(21), seeds), "20,10,6,5,7,13,19,11,12,4,16,3,14,9,1,8,15,2,17,18,0,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(25), seeds), "1,20,21,0,23,9,17,7,12,22,3,2,13,18,14,19,6,24,15,4,10,16,5,8,11,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(26), seeds), "7,16,20,24,8,25,12,4,23,21,9,5,18,6,14,13,2,19,22,11,10,1,15,3,17,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(30), seeds), "19,23,11,6,25,16,29,27,10,21,17,18,24,13,26,7,9,28,14,8,15,12,20,5,22,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(40), seeds), "21,30,36,35,39,32,38,17,22,24,26,25,15,28,37,27,16,34,31,20,23,18,29,33,19,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(100), seeds), "76,94,86,82,90,92,97,81,83,89,77,80,96,75,95,93,88,78,84,87,85,98,99,91,79,")

	seeds[mockAddress(0)] = uint64(100)

	assert.NotEqual(t, testFilterVotes(t, al, votes, seeds), "9,6,4,3,5,0,2,8,7,1,")
	assert.Equal(t, testFilterVotes(t, al, votes, seeds), "8,2,6,4,9,3,5,0,7,1,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(21), seeds), "19,13,0,15,20,10,16,1,18,12,2,4,9,7,8,6,14,5,3,11,17,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(25), seeds), "23,3,4,19,24,14,20,5,1,16,6,0,13,11,12,10,18,9,7,15,21,2,17,22,8,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(26), seeds), "24,4,5,20,25,15,21,6,2,17,7,1,14,12,13,11,19,10,8,16,22,3,18,23,9,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(30), seeds), "28,8,9,24,29,19,25,10,6,21,11,5,18,16,17,15,23,14,12,20,26,7,22,27,13,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(40), seeds), "38,18,19,34,39,29,35,20,16,31,21,15,28,26,27,25,33,24,22,30,36,17,32,37,23,")
	assert.Equal(t, testFilterVotes(t, al, genVotes(100), seeds), "98,78,79,94,99,89,95,80,76,91,81,75,88,86,87,85,93,84,82,90,96,77,92,97,83,")

}

func TestMembersInfo(t *testing.T) {
	info := core.NewGroupInfo(time.Now(), types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, PerCount: 3, RandCount: 0, Gid: types.SNAPSHOT_GID})

	now := time.Now()
	index := info.Time2Index(now)

	sTime := info.GenSTime(index - 1)
	fmt.Println(sTime.Sub(now).String(), now, sTime)
}

func testFilterVotes(t *testing.T, al core.Algo, votes []*core.Vote, seeds map[types.Address]uint64) string {

	hash, _ := types.HexToHash("706b00a2ae1725fb5d90b3b7a76d76c922eb075be485749f987af7aa46a66785")
	testH := &ledger.HashHeight{Hash: hash, Height: 1}
	fVotes1 := al.FilterVotes(votes, testH, core.NewSeedInfo(seeds))
	fVotes2 := al.FilterVotes(votes, testH, core.NewSeedInfo(seeds))

	result := ""
	for k, v := range fVotes1 {
		result += v.Name + ","
		if fVotes2[k].Name != v.Name {
			t.Error("errr for filterVotes")
		}
	}
	println(result)
	return result
}

func TestShuffleVotes(t *testing.T) {

	info := core.NewGroupInfo(time.Now(), types.ConsensusGroupInfo{NodeCount: 25, Interval: 1, PerCount: 3, RandCount: 0, Gid: types.DELEGATE_GID})

	teller := newTeller(info, &chainRw{}, log15.New("module", "unitTest"))

	votes := genVotes(11)
	shuffleVotes1 := teller.algo.ShuffleVotes(votes, nil, core.NewSeedInfo(nil))
	shuffleVotes2 := teller.algo.ShuffleVotes(votes, nil, core.NewSeedInfo(nil))

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
	votes = teller.algo.FilterVotes(votes, nil, core.NewSeedInfo(nil))

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
