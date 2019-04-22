package consensus

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/vitelabs/go-vite/consensus/db"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type snapshotCs struct {
	consensusDpos
	//voteCache map[int32]*electionResult
	//voteCache *lru.Cache
	rw   *chainRw
	algo core.Algo

	log log15.Logger
}

// hour stats
func (self *snapshotCs) HourStats(startIndex uint64, endIndex uint64) ([]*core.HourStats, error) {
	genesis := self.rw.rw.GetGenesisSnapshotBlock()
	stats, err := getBaseStats(genesis.Hash, self.rw.hourPoints, startIndex, endIndex)
	if err != nil {
		return nil, err
	}

	result := make([]*core.HourStats, len(stats))

	for k, v := range stats {
		result[k] = &core.HourStats{BaseStats: v}
	}
	return result, nil
}

func (self *snapshotCs) GetHourTimeIndex() core.TimeIndex {
	return self.rw.hourPoints
}

// period stats
func (self *snapshotCs) PeriodStats(startIndex uint64, endIndex uint64) ([]*core.PeriodStats, error) {
	genesis := self.rw.rw.GetGenesisSnapshotBlock()
	stats, err := getBaseStats(genesis.Hash, self.rw.periodPoints, startIndex, endIndex)
	if err != nil {
		return nil, err
	}

	result := make([]*core.PeriodStats, len(stats))

	for k, v := range stats {
		result[k] = &core.PeriodStats{BaseStats: v}
	}
	return result, nil
}

func (self *snapshotCs) GetPeriodTimeIndex() core.TimeIndex {
	return self.rw.periodPoints
}

func getBaseStats(genesisHash types.Hash, array LinkedArray, startIndex uint64, endIndex uint64) ([]*core.BaseStats, error) {
	// get points from linked array
	points := make(map[uint64]*consensus_db.Point)

	var proofHash *types.Hash
	for i := endIndex; i >= startIndex; i-- {
		var point *consensus_db.Point
		if proofHash == nil {
			tmp, err := array.GetByIndex(i)
			if err != nil {
				return nil, err
			}
			point = tmp
		} else if *proofHash == genesisHash {
			break
		} else {

			tmp, err := array.GetByIndexWithProof(i, *proofHash)
			if err != nil {
				return nil, err
			}
			point = tmp
		}

		if point.IsEmpty() {
			continue
		}
		points[i] = point
		proofHash = &point.PrevHash
	}

	if len(points) == 0 {
		return nil, nil
	}
	var result []*core.BaseStats

	// convert Points to HourStats
	for i := startIndex; i <= endIndex; i++ {
		p := points[i]
		if p == nil || p.IsEmpty() {
			continue
		}

		stats := &core.BaseStats{Stats: make(map[types.Address]*core.SbpStats), Index: i}

		for k, v := range p.Sbps {
			// convert to sbp stats
			sbp := &core.SbpStats{Index: i, BlockNum: uint64(v.FactualNum), ExceptedBlockNum: uint64(v.ExpectedNum)}
			stats.Stats[k] = sbp
		}

		result = append(result, stats)
	}
	return result, nil
}

func (self *snapshotCs) GetDayTimeIndex() core.TimeIndex {
	return self.rw.dayPoints
}

func (self *snapshotCs) DayStats(startIndex uint64, endIndex uint64) ([]*core.DayStats, error) {
	// get points from linked array
	points := make(map[uint64]*consensus_db.Point)

	var proofHash *types.Hash
	for i := endIndex; i >= startIndex; i-- {

		var point *consensus_db.Point
		if proofHash == nil {
			tmp, err := self.rw.dayPoints.GetByIndex(i)
			if err != nil {
				return nil, err
			}
			point = tmp
		} else {
			tmp, err := self.rw.dayPoints.GetByIndexWithProof(i, *proofHash)
			if err != nil {
				return nil, err
			}
			point = tmp
		}

		if point.IsEmpty() {
			continue
		}
		points[i] = point
		proofHash = &point.PrevHash
	}

	if len(points) == 0 {
		return nil, nil
	}
	var result []*core.DayStats

	// get register map for address->name
	registerMap := make(map[types.Address]string)
	lastHash := self.rw.GetLatestSnapshotBlock().Hash
	registers, err := self.rw.rw.GetRegisterList(lastHash, types.SNAPSHOT_GID)
	if err != nil {
		return nil, err
	}
	for _, v := range registers {
		for _, vv := range v.HisAddrList {
			registerMap[vv] = v.Name
		}
	}
	// convert Points to DayStats
	for i := startIndex; i <= endIndex; i++ {
		p := points[i]
		if p == nil || p.IsEmpty() {
			continue
		}
		if p.Votes == nil {
			continue
		}
		stats := &core.DayStats{Stats: make(map[string]*core.SbpStats), Index: i, VoteSum: &core.BigInt{Int: p.Votes.Total}}

		for k, v := range p.Votes.Details {
			stats.Stats[k] = &core.SbpStats{Index: i, VoteCnt: &core.BigInt{Int: v}, Name: k}
		}

		for k, v := range p.Sbps {
			name := registerMap[k]
			if sbp, ok := stats.Stats[name]; ok {
				// just vote
				sbp.BlockNum += uint64(v.FactualNum)
				sbp.ExceptedBlockNum += uint64(v.ExpectedNum)
			}
			// block sum
			stats.BlockTotal += uint64(v.FactualNum)
		}
		result = append(result, stats)
	}
	return result, nil
}

func (self *snapshotCs) dayVoteStat(b byte, index uint64, proofHash types.Hash) (*consensus_db.VoteContent, error) {
	votes, err := core.CalVotes(self.info.ConsensusGroupInfo, proofHash, self.rw.rw)
	if err != nil {
		return nil, err
	}
	sort.Sort(core.ByBalance(votes))

	total := big.NewInt(0)

	details := make(map[string]*big.Int)
	for k, v := range votes {
		if k >= int(self.info.RandRank) {
			break
		}
		details[v.Name] = v.Balance
		total.Add(total, v.Balance)
	}
	result := &consensus_db.VoteContent{Details: details, Total: total}
	return result, nil
}

func newSnapshotCs(rw *chainRw, log log15.Logger) *snapshotCs {
	cs := &snapshotCs{}
	cs.rw = rw
	cs.log = log.New("gid", "snapshot")
	info, err := rw.GetMemberInfo(types.SNAPSHOT_GID)
	if err != nil {
		panic(err)
	}
	cs.algo = core.NewAlgo(info)
	cs.info = info
	return cs
}

func (self *snapshotCs) ElectionTime(t time.Time) (*electionResult, error) {
	index := self.Time2Index(t)
	return self.ElectionIndex(index)
}

func (self *snapshotCs) ElectionIndex(index uint64) (*electionResult, error) {
	proofTime, proofIndex := self.genSnapshotProofTimeIndx(index)

	block, e := self.rw.GetSnapshotBeforeTime(proofTime)
	if e != nil {
		self.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, e
	}
	seeds := self.rw.GetSeedsBeforeHashH(block)
	// todo
	self.log.Debug(fmt.Sprintf("election index:%d,%s, proofTime:%s, seeds:%d", index, block.Hash, proofTime, seeds))
	seed := core.NewSeedInfo(seeds)
	voteResults, err := self.calVotes(ledger.HashHeight{Hash: block.Hash, Height: block.Height}, seed, index, proofIndex)
	if err != nil {
		return nil, err
	}

	plans := genElectionResult(self.info, index, voteResults)
	return plans, nil
}

func (self *snapshotCs) calVotes(hashH ledger.HashHeight, seed *core.SeedInfo, index uint64, proofIndex uint64) ([]types.Address, error) {
	// load from cache
	r, ok := self.rw.getSnapshotVoteCache(hashH.Hash)
	if ok {
		//fmt.Println(fmt.Sprintf("hit cache voteIndex:%d,%s,%+v", voteIndex, hashH.Hash, r))
		return r, nil
	}
	// record vote
	votes, err := self.rw.CalVotes(self.info, hashH)
	if err != nil {
		return nil, err
	}

	var successRate map[types.Address]int32

	if proofIndex > 0 {
		successRate, err = self.rw.GetSuccessRateByHour(proofIndex)
		if err != nil {
			return nil, err
		}
	}

	all := ""
	for _, v := range votes {
		all += fmt.Sprintf("[%s-%s]", v.Name, v.Balance.String())
	}
	self.log.Info(fmt.Sprintf("[%d][%d]pre success rate log: %+v, %s", hashH.Height, index, successRate, all))

	context := core.NewVoteAlgoContext(votes, &hashH, successRate, seed)
	// filter size of members
	finalVotes := self.algo.FilterVotes(context)
	// shuffle the members
	finalVotes = self.algo.ShuffleVotes(finalVotes, &hashH, seed)

	result := fmt.Sprintf("CalVotes result: %d:%d:%s, ", index, hashH.Height, hashH.Hash)
	for _, v := range finalVotes {
		if len(v.Type) > 0 {
			result += fmt.Sprintf("[%s:%+v],", v.Name, v.Type)
		} else {
			result += fmt.Sprintf("[%s],", v.Name)
		}
	}
	self.log.Info(result)
	address := core.ConvertVoteToAddress(finalVotes)

	// update cache
	self.rw.updateSnapshotVoteCache(hashH.Hash, address)
	return address, nil
}

// generate the vote time for snapshot consensus group
func (self *snapshotCs) GenProofTime(idx uint64) time.Time {
	t, _ := self.genSnapshotProofTimeIndx(idx)
	return t
}

func (self *snapshotCs) genSnapshotProofTimeIndx(idx uint64) (time.Time, uint64) {
	if idx < 2 {
		return self.info.GenesisTime.Add(time.Second), 0
	}
	return self.info.GenETime(idx - 2), idx - 2
}

func (self *snapshotCs) voteDetailsBeforeTime(t time.Time) ([]*VoteDetails, *ledger.HashHeight, error) {
	block, e := self.rw.GetSnapshotBeforeTime(t)
	if e != nil {
		self.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, nil, e
	}

	headH := ledger.HashHeight{Height: block.Height, Hash: block.Hash}
	details, err := self.rw.CalVoteDetails(self.info.Gid, self.info, headH)
	return details, &headH, err
}

func (self *snapshotCs) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	electionResult, err := self.ElectionTime(*header.Timestamp)
	if err != nil {
		return false, err
	}

	return self.verifyProducer(*header.Timestamp, header.Producer(), electionResult), nil
}

func (self *snapshotCs) VerifyProducer(address types.Address, t time.Time) (bool, error) {
	electionResult, err := self.ElectionTime(t)
	if err != nil {
		return false, err
	}

	return self.verifyProducer(t, address, electionResult), nil
}

func (self *snapshotCs) verifyProducer(t time.Time, address types.Address, result *electionResult) bool {
	if result == nil {
		return false
	}
	for _, plan := range result.Plans {
		if plan.Member == address {
			if plan.STime == t {
				return true
			}
		}
	}
	return false
}
