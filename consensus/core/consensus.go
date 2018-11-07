package core

import (
	"math/big"

	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Detail struct {
	PlanNum   uint64 // member plan cnt
	ActualNum uint64 // member actual cnt
	//top100, index: nodeName: balance
	PeriodM map[uint64]*PeriodDetails
}

type PeriodDetails struct {
	ActualNum uint64 // actual block num in period
	VoteMap   map[string]*big.Int
}

type ConsensusReader interface {
	PeriodTime() (uint64, error)
	TimeToIndex(time time.Time) (uint64, error)
	// return
	VoteDetails(startIndex, endIndex uint64, register *types.Registration, r stateCh) (*Detail, error)
}

func NewReader(genesisTime time.Time, info *types.ConsensusGroupInfo) ConsensusReader {
	i := NewGroupInfo(genesisTime, *info)
	return &reader{info: i, ag: NewAlgo(i)}
}

type Vote struct {
	Name    string
	Addr    types.Address
	Balance *big.Int
}

type ByBalance []*Vote

func (a ByBalance) Len() int      { return len(a) }
func (a ByBalance) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByBalance) Less(i, j int) bool {

	r := a[i].Balance.Cmp(a[j].Balance)
	if r == 0 {
		return a[i].Name < a[j].Name
	} else {
		return r < 0
	}
}

type reader struct {
	info *GroupInfo
	ag   Algo
}

func (self *reader) PeriodTime() (uint64, error) {
	return self.info.PlanInterval, nil
}

func (self *reader) TimeToIndex(time time.Time) (uint64, error) {
	return self.info.Time2Index(time), nil
}

func (self *reader) VoteDetails(startIndex, endIndex uint64,
	register *types.Registration,
	r stateCh) (*Detail, error) {

	periodM := make(map[uint64]*PeriodDetails)
	memAllPlanNum := uint64(0)
	memAllActualNum := uint64(0)
	for i := startIndex; i <= endIndex; i++ {
		votes, finalVotes, err := self.voteDetail(i, r)
		if err != nil {
			return nil, err
		}
		for _, v := range finalVotes {
			if v.Name == register.Name {
				memAllPlanNum++
			}
		}
		actualNum, memActualNum, err := self.actualSnapshotBlockNum(i, register, r)
		if err != nil {
			return nil, err
		}

		memAllActualNum += memActualNum
		voteM := make(map[string]*big.Int)
		for _, v := range votes {
			voteM[v.Name] = v.Balance
		}
		periodM[i] = &PeriodDetails{
			ActualNum: actualNum,
			VoteMap:   voteM,
		}
	}
	return &Detail{PlanNum: memAllPlanNum, ActualNum: memAllActualNum, PeriodM: periodM}, nil
}

func (self *reader) voteDetail(index uint64,
	r stateCh) ([]*Vote, []*Vote, error) {

	voteTime := self.info.GenVoteTime(index)
	block, err := r.GetSnapshotBlockBeforeTime(&voteTime)

	hashH := ledger.HashHeight{Hash: block.Hash, Height: block.Height}

	votes, err := CalVotes(self.info, hashH, r)
	if err != nil {
		return nil, nil, err
	}
	// top
	topVotes := self.ag.FilterSimple(votes)
	// filter size of members
	finalVotes := self.ag.FilterVotes(votes, &hashH)
	// shuffle the members
	finalVotes = self.ag.ShuffleVotes(finalVotes, &hashH)
	return topVotes, finalVotes, nil
}
func (self *reader) actualSnapshotBlockNum(index uint64, register *types.Registration, r stateCh) (uint64, uint64, error) {
	result := uint64(0)
	memResult := uint64(0)
	sTime := self.info.GenSTime(index)
	eTime := self.info.GenETime(index + 1)
	first, err := r.GetSnapshotBlockBeforeTime(&eTime)
	if err != nil {
		return 0, 0, err
	}
	m := make(map[types.Address]bool)
	addr := register.HisAddrList
	for _, v := range addr {
		m[v] = true
	}

	tmp := first
	for !tmp.Timestamp.Before(sTime) {
		result++
		_, ok := m[tmp.Producer()]
		if ok {
			memResult++
		}
		if tmp.Height <= types.GenesisHeight {
			break
		}
		tmp, err = r.GetSnapshotBlockByHeight(tmp.Height - 1)
		if err != nil {
			return 0, 0, nil
		}
		if tmp == nil {
			break
		}

	}
	return result, memResult, nil
}
