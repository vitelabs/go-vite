package consensus

import (
	"math/big"
	"math/rand"
	"sort"
)

type algo struct {
	info *membersInfo
	r    int32
}

func (self *algo) findSeed(votes []*Vote) int64 {
	result := big.NewInt(0)
	for _, v := range votes {
		result.Add(result, v.balance)
	}
	return result.Int64()
}

func (self *algo) shuffleVotes(votes []*Vote) []*Vote {
	seed := self.findSeed(votes) + 10
	lenght := len(votes)
	randMembers := make([]bool, lenght)
	var result []*Vote
	random := rand.New(rand.NewSource(seed))

	for i := 0; i < lenght; {
		r := random.Intn(lenght)
		flag := randMembers[r]
		if flag {
			continue
		} else {
			randMembers[r] = true
			i++
			result = append(result, votes[r])
		}
	}
	return result
}

func (self *algo) filterVotes(votes []*Vote) []*Vote {
	// simple filter for low balance
	votes = self.filterSimple(votes, self.info)

	votes = self.filterRand(votes)

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
	if total/3 > randNum {
		return int(randNum)
	} else {
		return int(total) / 3
	}
}
func (self *algo) filterRand(votes []*Vote) []*Vote {
	total := self.info.memberCnt
	sort.Sort(byBalance(votes))
	length := len(votes)
	if int32(length) < total {
		return votes
	}
	seed := self.findSeed(votes)

	randCnt := self.calRandCnt(total, self.info.randCnt)
	topTotal := int(total) - randCnt

	randMembers := make([]bool, length-topTotal)
	random := rand.New(rand.NewSource(seed))

	// cal rand index
	for i := 0; i < randCnt; {
		r := random.Intn(length - topTotal)
		flag := randMembers[r]
		if flag {
			continue
		} else {
			randMembers[r] = true
			i++
		}
	}

	// generate result for top and rand
	var result []*Vote
	for k, v := range votes {
		if k < topTotal {
			result = append(result, v)
			continue
		}
		if randMembers[k-topTotal] {
			result = append(result, v)
			continue
		}
	}
	return result
}
