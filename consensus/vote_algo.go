package consensus

import (
	"math/big"
	"math/rand"
	"sort"

	"github.com/vitelabs/go-vite/ledger"
)

type algo struct {
	info *membersInfo
}

// balance + snapshotHeight + gid
func (self *algo) findSeed(votes []*Vote, sheight uint64) int64 {
	result := big.NewInt(0)
	for _, v := range votes {
		result.Add(result, v.balance)
	}
	result.Add(result, new(big.Int).SetUint64(sheight))
	return result.Add(result, self.info.seed).Int64()
}

func (self *algo) shuffleVotes(votes []*Vote, hashH *ledger.HashHeight) (result []*Vote) {
	seed := self.findSeed(votes, hashH.Height)
	l := len(votes)
	random := rand.New(rand.NewSource(seed))
	perm := random.Perm(l)

	for _, v := range perm {
		result = append(result, votes[v])
	}
	return result
}

func (self *algo) filterVotes(votes []*Vote, hashH *ledger.HashHeight) []*Vote {
	// simple filter for low balance
	simpleVotes := self.filterSimple(votes, self.info)

	if int32(len(simpleVotes)) < self.info.memberCnt {
		simpleVotes = votes
	}

	votes = self.filterRand(simpleVotes, hashH)

	return votes
}

func (self *algo) filterSimple(votes []*Vote, info *membersInfo) []*Vote {
	var result []*Vote
	for _, v := range votes {
		if v.balance.Cmp(info.LowestLimit) < 0 {
			continue
		}
		result = append(result, v)
	}
	return result

}

func (self *algo) calRandCnt(total int32, randNum int32) int {
	// the random number can't be greater than 1/3
	// todo need detail info
	if total/3 > randNum {
		return int(randNum)
	} else {
		return int(total) / 3
	}
}
func (self *algo) filterRand(votes []*Vote, hashH *ledger.HashHeight) []*Vote {
	total := self.info.memberCnt
	sort.Sort(byBalance(votes))
	length := len(votes)
	if int32(length) < total {
		return votes
	}
	seed := self.findSeed(votes, hashH.Height)

	randCnt := self.calRandCnt(total, self.info.randCnt)
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
