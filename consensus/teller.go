package consensus

import (
	"math/big"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/contracts"
	"github.com/vitelabs/go-vite/ledger"
)

// Ensure that all nodes get same result
type teller struct {
	info *membersInfo
	//electionHis map[int32]*electionResult
	electionHis sync.Map
	rw          *chainRw
	algo        *algo
}

func newTeller(info *membersInfo, rw *chainRw) *teller {
	t := &teller{rw: rw}
	//t.info = &membersInfo{genesisTime: genesisTime, memberCnt: memberCnt, interval: interval, perCnt: perCnt, randCnt: 2, LowestLimit: big.NewInt(1000)}
	t.info = info
	t.algo = &algo{info: t.info}
	//t.electionHis = make(map[int32]*electionResult)
	return t
}

func (self *teller) voteResults(t time.Time) ([]types.Address, *ledger.HashHeight, error) {
	// record vote
	// todo gid ??
	votes, hashH, err := self.rw.CalVotes(types.SNAPSHOT_GID, t)
	if err != nil {
		return nil, nil, err
	}
	// filter size of members
	finalVotes := self.algo.filterVotes(votes)
	// shuffle the members
	finalVotes = self.algo.shuffleVotes(finalVotes)
	return self.convertToAddress(finalVotes), hashH, nil
}

func toMap(infos []*contracts.VoteInfo) map[string]bool {
	m := make(map[string]bool)
	for _, v := range infos {
		m[v.NodeName] = true
	}
	return m
}

func (self *teller) electionIndex(index int32) (*electionResult, error) {
	r, ok := self.electionHis.Load(index)
	if ok {
		return r.(*electionResult), nil
	}
	sTime := self.info.genSTime(index - 1)
	voteResults, hashH, err := self.voteResults(sTime)
	if err != nil {
		return nil, err
	}

	plans := self.info.genPlan(index, voteResults)
	plans.Hash = hashH.Hash
	plans.Height = hashH.Height
	self.electionHis.Store(index, plans)
	return plans, nil
}

func (self *teller) electionTime(t time.Time) (*electionResult, error) {
	index := self.info.time2Index(t)
	return self.electionIndex(index)
}
func (self *teller) time2Index(t time.Time) int32 {
	index := self.info.time2Index(t)
	return index
}

func (self *teller) removePrevious(rtime time.Time) int32 {
	var i int32 = 0
	self.electionHis.Range(func(k, v interface{}) bool {
		if v.(*electionResult).ETime.Before(rtime) {
			self.electionHis.Delete(k)
			i = i + 1
		}
		return true
	})
	return i
}

func (self *teller) findSeed(votes []*Vote) int64 {
	result := big.NewInt(0)
	for _, v := range votes {
		result.Add(result, v.balance)
	}
	return result.Int64()
}
func (self *teller) convertToAddress(votes []*Vote) []types.Address {
	var result []types.Address
	for _, v := range votes {
		result = append(result, v.addr)
	}
	return result
}

type byBalance []*Vote

func (a byBalance) Len() int      { return len(a) }
func (a byBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byBalance) Less(i, j int) bool {
	r := a[i].balance.Cmp(a[j].balance)
	if r == 0 {
		return a[i].name < a[j].name
	} else {
		return r < 0
	}
}
