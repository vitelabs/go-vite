package core

import (
	"math/big"
	"math/rand"
	"sort"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type VoteAlgoContext struct {
	votes       []*Vote
	hashH       *ledger.HashHeight
	successRate map[types.Address]int32
	seeds       *SeedInfo

	// topN sbp
	sbps []*Vote
}

func NewVoteAlgoContext(votes []*Vote, hashH *ledger.HashHeight, successRate map[types.Address]int32, seeds *SeedInfo) *VoteAlgoContext {
	tmp := successRate
	if tmp == nil {
		tmp = make(map[types.Address]int32)
	}
	return &VoteAlgoContext{votes: votes, hashH: hashH, successRate: tmp, seeds: seeds}
}

type Algo interface {
	ShuffleVotes(votes []*Vote, hashH *ledger.HashHeight, info *SeedInfo) []*Vote
	FilterVotes(context *VoteAlgoContext) []*Vote
	FilterSimple(votes []*Vote) ([]*Vote, []*Vote)
}

type SeedInfo struct {
	seeds uint64
}

func NewSeedInfo(seed uint64) *SeedInfo {
	return &SeedInfo{seeds: seed}
}

type algo struct {
	info *GroupInfo
}

func NewAlgo(info *GroupInfo) *algo {
	return &algo{info: info}
}

// balance + snapshotHeight + gid
func (self *algo) findSeed(votes []*Vote, sheight uint64, info *SeedInfo) int64 {
	seeds := info.seeds
	if seeds == 0 {
		result := big.NewInt(0)
		for _, v := range votes {
			result.Add(result, v.Balance)
		}
		result.Add(result, new(big.Int).SetUint64(sheight))
		return result.Add(result, self.info.seed).Int64()
	}

	result := big.NewInt(0)
	result.Add(result, big.NewInt(0).SetUint64(seeds))
	result.Add(result, new(big.Int).SetUint64(sheight))
	return result.Add(result, self.info.seed).Int64()
}

func (self *algo) ShuffleVotes(votes []*Vote, hashH *ledger.HashHeight, info *SeedInfo) (result []*Vote) {
	seed := self.findSeed(votes, hashH.Height, info)
	l := len(votes)
	random := rand.New(rand.NewSource(seed))
	perm := random.Perm(l)

	for _, v := range perm {
		result = append(result, votes[v])
	}
	return result
}

func (self *algo) FilterVotes(context *VoteAlgoContext) []*Vote {
	votes := context.votes
	hashH := context.hashH
	// simple filter for low balance
	groupA, groupB := self.FilterSimple(votes)
	// top N sbps
	context.sbps = mergeGroup(groupA, groupB)

	successRates := context.successRate

	if successRates != nil {
		groupA, groupB = self.filterBySuccessRate(groupA, groupB, hashH, successRates)
	}

	votes = self.filterRandV2(groupA, groupB, hashH, context.seeds)

	return votes
}

func (self *algo) FilterSimple(votes []*Vote) (groupA []*Vote, groupB []*Vote) {
	if len(votes) <= int(self.info.NodeCount) {
		return votes, groupB
	}
	groupA = votes[0:self.info.NodeCount]
	sort.Sort(ByBalance(votes))
	if len(votes) <= int(self.info.RandRank) {
		groupB = votes[self.info.NodeCount:]
		return groupA, groupB
	}
	groupB = votes[self.info.NodeCount:self.info.RandRank]
	return groupA, groupB
}

func (self *algo) calRandCnt(total int, randNum int) int {
	// the random number can't be greater than 1/3
	// todo need detail info
	if total/3 > randNum {
		return randNum
	} else {
		return (total) / 3
	}
}
func (self *algo) filterRand(votes []*Vote, hashH *ledger.HashHeight, seedInfo *SeedInfo) []*Vote {
	total := self.info.NodeCount
	sort.Sort(ByBalance(votes))
	length := len(votes)
	if length < int(total) {
		return votes
	}
	seed := self.findSeed(votes, hashH.Height, seedInfo)

	randCnt := self.calRandCnt(int(total), int(self.info.RandCount))
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

func (self *algo) filterRandV2(groupA, groupB []*Vote, hashH *ledger.HashHeight, seedInfo *SeedInfo) []*Vote {
	var result []*Vote
	total := int(self.info.NodeCount)
	sort.Sort(ByBalance(groupA))
	sort.Sort(ByBalance(groupB))

	seed := self.findSeed(mergeGroup(groupA, groupB), hashH.Height, seedInfo)
	length := len(groupA) + len(groupB)
	if len(groupB) == 0 {
		random1 := rand.New(rand.NewSource(seed))
		arr := random1.Perm(int(length))
		for _, v := range arr {
			result = append(result, groupA[v])
		}
		return result
	}

	randCnt := self.calRandCnt(total, int(self.info.RandCount))
	topTotal := total - randCnt

	//if (total-randCnt)/total) > randCnt/(length-total) {
	if (topTotal)*(length-total) > randCnt*total {
		random1 := rand.New(rand.NewSource(seed))
		random2 := rand.New(rand.NewSource(seed + 1))

		firstArr := random1.Perm(int(total))[0:topTotal]
		secondArr := random2.Perm(length - total)[0:randCnt]

		for _, v := range firstArr {
			result = append(result, groupA[v])
		}

		for _, v := range secondArr {
			promotion := groupB[v]
			promotion.Type = append(promotion.Type, RANDOM_PROMOTION)
			result = append(result, promotion)
		}
	} else {
		random1 := rand.New(rand.NewSource(seed))
		arr := random1.Perm(int(length))[0:total]
		for _, v := range arr {
			if v >= total {
				promotion := groupB[v-total]
				promotion.Type = append(promotion.Type, RANDOM_PROMOTION)
				result = append(result, promotion)
			} else {
				result = append(result, groupA[v])
			}
		}
	}
	return result
}
func (self *algo) filterBySuccessRate(groupA, groupB []*Vote, height *ledger.HashHeight, successRate map[types.Address]int32) ([]*Vote, []*Vote) {
	if len(groupB) == 0 {
		return groupA, groupB
	}

	var groupA1 []*SuccessRateVote
	var deleteGroupA []*SuccessRateVote
	var improvementGroupB []*SuccessRateVote
	var groupB1 []*SuccessRateVote

	obsoletedNum := 2
	{
		// groupA
		var successRateGroupA []*SuccessRateVote
		for _, v := range groupA {
			rate, ok := successRate[v.Addr]
			if !ok {
				rate = int32(-1)
			}
			vote := &SuccessRateVote{Vote: v, Rate: rate}
			successRateGroupA = append(successRateGroupA, vote)
		}
		sort.Sort(BySuccessRate(successRateGroupA))

		tmps := successRateGroupA[len(successRateGroupA)-obsoletedNum:]

		for _, v := range successRateGroupA[0 : len(successRateGroupA)-obsoletedNum] {
			groupA1 = append(groupA1, v)
		}

		for _, v := range tmps {
			if v.Rate >= 0 && v.Rate < Line {
				deleteGroupA = append(deleteGroupA, v)
			} else {
				groupA1 = append(groupA1, v)
			}
		}

		// groupB
		var successRateGroupB []*SuccessRateVote
		for _, v := range groupB {
			rate, ok := successRate[v.Addr]
			if !ok {
				rate = int32(-1)
			}
			vote := &SuccessRateVote{Vote: v, Rate: rate}
			successRateGroupB = append(successRateGroupB, vote)
		}
		sort.Sort(BySuccessRate(successRateGroupB))

		for _, v := range successRateGroupB {
			if len(improvementGroupB) >= obsoletedNum {
				groupB1 = append(groupB1, v)
				continue
			}
			if v.Rate > Line {
				improvementGroupB = append(improvementGroupB, v)
			} else {
				groupB1 = append(groupB1, v)
			}
		}
	}

	// exchange
	for i := 1; i <= obsoletedNum; i++ {
		lenA := len(deleteGroupA)
		if lenA < i || len(improvementGroupB) < i {
			break
		}
		demotion := deleteGroupA[lenA-i]
		promotion := improvementGroupB[i-1]
		promotion.Type = append(promotion.Type, SUCCESS_RATE_PROMOTION)
		demotion.Type = append(demotion.Type, SUCCESS_RATE_DEMOTION)
		deleteGroupA[lenA-i] = promotion
		improvementGroupB[i-i] = demotion
	}

	var resultGroupA []*Vote
	var resultGroupB []*Vote

	for _, v := range groupA1 {
		resultGroupA = append(resultGroupA, v.Vote)
	}

	for _, v := range deleteGroupA {
		resultGroupA = append(resultGroupA, v.Vote)
	}

	for _, v := range groupB1 {
		resultGroupB = append(resultGroupB, v.Vote)
	}

	for _, v := range improvementGroupB {
		resultGroupB = append(resultGroupB, v.Vote)
	}

	return resultGroupA, resultGroupB
}

func (self *algo) checkValid(groupA []*Vote, groupB []*Vote) error {
	lenA := len(groupA)
	lenB := len(groupB)
	if lenA > int(self.info.NodeCount) {
		return errors.Errorf("groupA's size[%d] must <= %d.", lenA, self.info.NodeCount)
	}
	if lenA < int(self.info.NodeCount) && lenB > 0 {
		return errors.Errorf("groupB's size[%d] should be zero.", lenB)
	}
	if lenB+lenA > int(self.info.RandRank) {
		return errors.Errorf("the sum of groupA[%d] and groupB[%d] must <= %d.", lenA, lenB, self.info.RandRank)
	}
	return nil
}

type SuccessRateVote struct {
	*Vote
	Rate int32
}

type BySuccessRate []*SuccessRateVote

func (a BySuccessRate) Len() int      { return len(a) }
func (a BySuccessRate) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BySuccessRate) Less(i, j int) bool {
	r0 := a[j].Rate - a[i].Rate
	if r0 == 0 {
		r := a[j].Balance.Cmp(a[i].Balance)
		if r == 0 {
			return a[i].Name < a[j].Name
		} else {
			return r < 0
		}
	} else {
		return r0 < 0
	}
}

const (
	Line = 800000
)

func mergeGroup(groupA, groupB []*Vote) []*Vote {
	lenA := len(groupA)
	votes := make([]*Vote, lenA+len(groupB))
	for i, v := range groupA {
		votes[i] = v
	}
	for i, v := range groupB {
		votes[i+lenA] = v
	}
	return votes
}
