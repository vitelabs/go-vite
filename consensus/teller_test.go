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

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

func TestFilterVotes(t *testing.T) {
	info := &membersInfo{genesisTime: time.Now(), memberCnt: 25, interval: 1, perCnt: 3, randCnt: 0, seed: big.NewInt(10)}

	teller := newTeller(info, types.DELEGATE_GID, &chainRw{}, log15.New("module", "unitTest"))
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
	info := &membersInfo{genesisTime: time.Now(), memberCnt: 25, interval: 1, perCnt: 3, randCnt: 0, seed: big.NewInt(10)}

	now := time.Now()
	index := info.time2Index(now)

	sTime := info.genSTime(index - 1)
	fmt.Println(sTime.Sub(now).String(), now, sTime)
}

func testFilterVotes(t *testing.T, teller *teller, votes []*Vote) {
	hash, _ := types.HexToHash("706b00a2ae1725fb5d90b3b7a76d76c922eb075be485749f987af7aa46a66785")
	testH := &ledger.HashHeight{Hash: hash, Height: 1}
	fVotes1 := teller.algo.filterVotes(votes, testH)
	fVotes2 := teller.algo.filterVotes(votes, testH)

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

	info := &membersInfo{genesisTime: time.Now(), memberCnt: 25, interval: 1, perCnt: 3, randCnt: 10}

	teller := newTeller(info, types.DELEGATE_GID, &chainRw{}, log15.New("module", "unitTest"))

	votes := genVotes(11)
	shuffleVotes1 := teller.algo.shuffleVotes(votes, nil)
	shuffleVotes2 := teller.algo.shuffleVotes(votes, nil)

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
	info := &membersInfo{genesisTime: time.Now(), memberCnt: 25, interval: 1, perCnt: 3, randCnt: 10}

	teller := newTeller(info, types.DELEGATE_GID, &chainRw{}, log15.New("module", "unitTest"))

	votes := genVotes(20)
	testGenPlan(t, teller, votes, 60)
	testGenPlan(t, teller, genVotes(25), 75)
	testGenPlan(t, teller, genVotes(26), 75)

}
func testGenPlan(t *testing.T, teller *teller, votes []*Vote, expectedCnt int) {
	votes = teller.algo.filterVotes(votes, nil)

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
		vote := &Vote{name: strconv.Itoa(i) + "", balance: big.NewInt(200000 + int64(i)), addr: mockAddress(i)}
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
