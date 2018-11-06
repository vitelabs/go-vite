package core

import (
	"math/big"
	"math/rand"
	"sort"

	"github.com/vitelabs/go-vite/ledger"
)

type Algo interface {
	ShuffleVotes(votes []*Vote, hashH *ledger.HashHeight) []*Vote
	FilterVotes(votes []*Vote, hashH *ledger.HashHeight) []*Vote
}

type algo struct {
	info *GroupInfo
}

func NewAlgo(info *GroupInfo) *algo {
	return &algo{info: info}
}

// balance + snapshotHeight + gid
func (self *algo) findSeed(votes []*Vote, sheight uint64) int64 {
	result := big.NewInt(0)
	for _, v := range votes {
		result.Add(result, v.Balance)
	}
	result.Add(result, new(big.Int).SetUint64(sheight))
	return result.Add(result, self.info.seed).Int64()
}

func (self *algo) ShuffleVotes(votes []*Vote, hashH *ledger.HashHeight) (result []*Vote) {
	seed := self.findSeed(votes, hashH.Height)
	l := len(votes)
	random := rand.New(rand.NewSource(seed))
	perm := random.Perm(l)

	for _, v := range perm {
		result = append(result, votes[v])
	}
	return result
}

func (self *algo) FilterVotes(votes []*Vote, hashH *ledger.HashHeight) []*Vote {
	// simple filter for low balance
	simpleVotes := self.filterSimple(votes, self.info)

	if int64(len(simpleVotes)) < int64(self.info.NodeCount) {
		simpleVotes = votes
	}

	votes = self.filterRand(simpleVotes, hashH)

	return votes
}

func (self *algo) filterSimple(votes []*Vote, info *GroupInfo) (result []*Vote) {
	if int32(len(votes)) < int32(info.RandRank) {
		return votes
	}
	sort.Sort(ByBalance(votes))
	result = votes[0:info.RandRank]
	return result

}

func (self *algo) calRandCnt(total uint8, randNum uint8) int {
	// the random number can't be greater than 1/3
	// todo need detail info
	if total/3 > randNum {
		return int(randNum)
	} else {
		return int(total) / 3
	}
}
func (self *algo) filterRand(votes []*Vote, hashH *ledger.HashHeight) []*Vote {
	total := self.info.NodeCount
	sort.Sort(ByBalance(votes))
	length := len(votes)
	if length < int(total) {
		return votes
	}
	seed := self.findSeed(votes, hashH.Height)

	randCnt := self.calRandCnt(total, self.info.RandCount)
	topTotal := int(total) - randCnt

	leftTotal := length - topTotal
	randMembers := make([]bool, leftTotal)
	random := rand.New(rand.NewSource(seed))

	// cal rand index
	for i := 0; i < randCnt; {
		r := random.Intn(leftTotal)
		flag := randMembers[r]
		if flag {
			continue
		} else {
			randMembers[r] = true
			i++
		}
	}

	// generate result for top and rand
	// todo unit test
	result := votes[0:topTotal]
	for k, b := range randMembers {
		if b {
			result = append(result, votes[k+topTotal])
		}
	}
	return result
}
