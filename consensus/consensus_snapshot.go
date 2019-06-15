package consensus

import (
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/cdb"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

type snapshotCs struct {
	core.GroupInfo
	//voteCache map[int32]*electionResult
	//voteCache *lru.Cache
	rw   *chainRw
	algo core.Algo

	log log15.Logger
}

func (snapshot *snapshotCs) GetSuccessRateByHour(index uint64) (map[types.Address]int32, error) {
	return snapshot.rw.GetSuccessRateByHour(index)
}

func (snapshot *snapshotCs) GetInfo() *core.GroupInfo {
	return &snapshot.GroupInfo
}

// hour stats
func (snapshot *snapshotCs) HourStats(startIndex uint64, endIndex uint64) ([]*core.HourStats, error) {
	genesis := snapshot.rw.rw.GetGenesisSnapshotBlock()
	stats, err := getBaseStats(genesis.Hash, snapshot.rw.hourPoints, startIndex, endIndex)
	if err != nil {
		return nil, err
	}

	result := make([]*core.HourStats, len(stats))

	for k, v := range stats {
		result[k] = &core.HourStats{BaseStats: v}
	}
	return result, nil
}

func (snapshot *snapshotCs) GetHourTimeIndex() core.TimeIndex {
	return snapshot.rw.hourPoints
}

// period stats
func (snapshot *snapshotCs) PeriodStats(startIndex uint64, endIndex uint64) ([]*core.PeriodStats, error) {
	genesis := snapshot.rw.rw.GetGenesisSnapshotBlock()
	stats, err := getBaseStats(genesis.Hash, snapshot.rw.periodPoints, startIndex, endIndex)
	if err != nil {
		return nil, err
	}

	result := make([]*core.PeriodStats, len(stats))

	for k, v := range stats {
		result[k] = &core.PeriodStats{BaseStats: v}
	}
	return result, nil
}

func (snapshot *snapshotCs) GetNodeCount() int {
	return int(snapshot.NodeCount)
}

func (snapshot *snapshotCs) GetPeriodTimeIndex() core.TimeIndex {
	return snapshot.rw.periodPoints
}

func getBaseStats(genesisHash types.Hash, array LinkedArray, startIndex uint64, endIndex uint64) ([]*core.BaseStats, error) {
	// get points from linked array
	points := make(map[uint64]*cdb.Point)

	var proofHash *types.Hash
	for i := endIndex; i >= startIndex && i <= endIndex; i-- {
		var point *cdb.Point
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

func (snapshot *snapshotCs) GetDayTimeIndex() core.TimeIndex {
	return snapshot.rw.dayPoints
}

func (snapshot *snapshotCs) DayStats(startIndex uint64, endIndex uint64) ([]*core.DayStats, error) {
	// get points from linked array
	points := make(map[uint64]*cdb.Point)

	var proofHash *types.Hash
	for i := endIndex; i >= startIndex && i <= endIndex; i-- {

		var point *cdb.Point
		if proofHash == nil {
			tmp, err := snapshot.rw.dayPoints.GetByIndex(i)
			if err != nil {
				return nil, err
			}
			point = tmp
		} else {
			tmp, err := snapshot.rw.dayPoints.GetByIndexWithProof(i, *proofHash)
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
	lastHash := snapshot.rw.GetLatestSnapshotBlock().Hash
	registers, err := snapshot.rw.rw.GetAllRegisterList(lastHash, types.SNAPSHOT_GID)
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

func (snapshot *snapshotCs) dayVoteStat(b byte, index uint64, proofHash types.Hash) (*cdb.VoteContent, error) {
	votes, err := core.CalVotes(snapshot.ConsensusGroupInfo, proofHash, snapshot.rw.rw)
	if err != nil {
		return nil, err
	}
	sort.Sort(core.ByBalance(votes))

	total := big.NewInt(0)

	details := make(map[string]*big.Int)
	for k, v := range votes {
		if k >= int(snapshot.RandRank) {
			break
		}
		details[v.Name] = v.Balance
		total.Add(total, v.Balance)
	}
	result := &cdb.VoteContent{Details: details, Total: total}
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
	cs.GroupInfo = *info
	return cs
}

func (snapshot *snapshotCs) ElectionTime(t time.Time) (*electionResult, error) {
	index := snapshot.Time2Index(t)
	return snapshot.ElectionIndex(index)
}

func (snapshot *snapshotCs) ElectionIndex(index uint64) (*electionResult, error) {
	proofTime, _ := snapshot.genSnapshotProofTimeIndx(index)

	proofBlock, e := snapshot.rw.GetSnapshotBeforeTime(proofTime)
	if e != nil {
		snapshot.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, e
	}

	snapshot.log.Debug(fmt.Sprintf("election index:%d,%s, proofTime:%s", index, proofBlock.Hash, proofTime))

	voteResults, err := snapshot.calVotes(proofBlock, index)
	if err != nil {
		return nil, err
	}

	plans := genElectionResult(&snapshot.GroupInfo, index, voteResults)
	return plans, nil
}

func (snapshot *snapshotCs) calVotes(proofBlock *ledger.SnapshotBlock, index uint64) ([]types.Address, error) {
	hashH := ledger.HashHeight{Hash: proofBlock.Hash, Height: proofBlock.Height}
	// load from cache
	r, ok := snapshot.rw.getSnapshotVoteCache(hashH.Hash)
	if ok {
		//fmt.Println(fmt.Sprintf("hit cache voteIndex:%d,%s,%+v", voteIndex, hashH.Hash, r))
		return r, nil
	}
	seed := core.NewSeedInfo(snapshot.rw.GetSeedsBeforeHashH(hashH.Hash))
	// record vote
	votes, err := snapshot.rw.CalVotes(&snapshot.GroupInfo, hashH)
	if err != nil {
		return nil, err
	}

	var successRate map[types.Address]int32

	_, proofIndex := snapshot.genSnapshotProofTimeIndx(snapshot.Time2Index(*proofBlock.Timestamp))
	if proofIndex > 0 {
		successRate, err = snapshot.rw.GetSuccessRateByHour(proofIndex)
		if err != nil {
			return nil, err
		}
	}

	all := ""
	for _, v := range votes {
		all += fmt.Sprintf("[%s-%s]", v.Name, v.Balance.String())
	}
	snapshot.log.Info(fmt.Sprintf("[%d][%d]pre success rate log: %+v, %s, seed:%d", hashH.Height, index, successRate, all, seed))

	context := core.NewVoteAlgoContext(votes, &hashH, successRate, seed)
	// filter size of members
	finalVotes := snapshot.algo.FilterVotes(context)
	// shuffle the members
	finalVotes = snapshot.algo.ShuffleVotes(finalVotes, &hashH, seed)

	result := fmt.Sprintf("CalVotes result: %d:%d:%s, ", index, hashH.Height, hashH.Hash)
	for _, v := range finalVotes {
		if len(v.Type) > 0 {
			result += fmt.Sprintf("[%s:%+v],", v.Name, v.Type)
		} else {
			result += fmt.Sprintf("[%s],", v.Name)
		}
	}
	snapshot.log.Info(result)
	address := core.ConvertVoteToAddress(finalVotes)

	// update cache
	snapshot.rw.updateSnapshotVoteCache(hashH.Hash, address)
	return address, nil
}
func (snapshot *snapshotCs) triggerLoad(proofBlock *ledger.SnapshotBlock) {
	select {
	case snapshot.rw.snapshotLoadCh <- proofBlock:
	default:
	}
}

func (snapshot *snapshotCs) loadVotes(proofBlock *ledger.SnapshotBlock) ([]types.Address, error) {
	snapshot.log.Info("loadVotes ", "hash", proofBlock.Hash, "height", proofBlock.Height)
	hashH := ledger.HashHeight{Hash: proofBlock.Hash, Height: proofBlock.Height}
	// load from cache
	r, ok := snapshot.rw.getVoteLRUCache(types.SNAPSHOT_GID, hashH.Hash)
	if ok {
		//fmt.Println(fmt.Sprintf("hit cache voteIndex:%d,%s,%+v", voteIndex, hashH.Hash, r))
		return r, nil
	}
	seed := core.NewSeedInfo(snapshot.rw.GetSeedsBeforeHashH(hashH.Hash))
	// record vote
	votes, err := snapshot.rw.CalVotes(&snapshot.GroupInfo, hashH)
	if err != nil {
		return nil, err
	}

	var successRate map[types.Address]int32

	_, proofIndex := snapshot.genSnapshotProofTimeIndx(snapshot.Time2Index(*proofBlock.Timestamp))
	if proofIndex > 0 {
		successRate, err = snapshot.rw.GetSuccessRateByHour(proofIndex)
		if err != nil {
			return nil, err
		}
	}

	all := ""
	for _, v := range votes {
		all += fmt.Sprintf("[%s-%s]", v.Name, v.Balance.String())
	}
	snapshot.log.Info(fmt.Sprintf("[load][%d][%d]pre success rate log: %+v, %s, seed:%d", hashH.Height, hashH.Hash, successRate, all, seed))

	context := core.NewVoteAlgoContext(votes, &hashH, successRate, seed)
	// filter size of members
	finalVotes := snapshot.algo.FilterVotes(context)
	// shuffle the members
	finalVotes = snapshot.algo.ShuffleVotes(finalVotes, &hashH, seed)

	result := fmt.Sprintf("[loadpw]CalVotes result: %d:%s, ", hashH.Height, hashH.Hash)
	for _, v := range finalVotes {
		if len(v.Type) > 0 {
			result += fmt.Sprintf("[%s:%+v],", v.Name, v.Type)
		} else {
			result += fmt.Sprintf("[%s],", v.Name)
		}
	}
	snapshot.log.Info(result)
	address := core.ConvertVoteToAddress(finalVotes)

	// update cache
	snapshot.rw.updateVoteLRUCache(types.SNAPSHOT_GID, hashH.Hash, address)
	return address, nil
}

// generate the vote time for snapshot consensus group
func (snapshot *snapshotCs) GenProofTime(idx uint64) time.Time {
	t, _ := snapshot.genSnapshotProofTimeIndx(idx)
	return t
}

func (snapshot *snapshotCs) genSnapshotProofTimeIndx(idx uint64) (time.Time, uint64) {
	if idx < 2 {
		return snapshot.GenesisTime.Add(time.Second), 0
	}
	_, etime := snapshot.Index2Time(idx - 2)
	return etime, idx - 2
}

func (snapshot *snapshotCs) voteDetailsBeforeTime(t time.Time) ([]*VoteDetails, *ledger.HashHeight, error) {
	block, e := snapshot.rw.GetSnapshotBeforeTime(t)
	if e != nil {
		snapshot.log.Error("geSnapshotBeferTime fail.", "err", e)
		return nil, nil, e
	}

	headH := ledger.HashHeight{Height: block.Height, Hash: block.Hash}
	details, err := snapshot.rw.CalVoteDetails(snapshot.Gid, &snapshot.GroupInfo, headH)
	return details, &headH, err
}

func (snapshot *snapshotCs) VerifySnapshotProducer(header *ledger.SnapshotBlock) (bool, error) {
	electionResult, err := snapshot.ElectionTime(*header.Timestamp)
	if err != nil {
		return false, err
	}
	return snapshot.verifyProducer(*header.Timestamp, header.Producer(), electionResult), nil
}

func (snapshot *snapshotCs) VerifyProducer(address types.Address, t time.Time) (bool, error) {
	electionResult, err := snapshot.ElectionTime(t)
	if err != nil {
		return false, err
	}
	return snapshot.verifyProducer(t, address, electionResult), nil
}

func (snapshot *snapshotCs) verifyProducer(t time.Time, address types.Address, result *electionResult) bool {
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

func (snapshot *snapshotCs) verifyProducerAndSeed(block *ledger.SnapshotBlock) (bool, error) {
	producerTime := *block.Timestamp
	electionResult, err := snapshot.ElectionTime(producerTime)
	if err != nil {
		return false, err
	}
	result := snapshot.verifyProducer(producerTime, block.Producer(), electionResult)
	if !result {
		return false, nil
	}

	if block.Seed != 0 {
		seedBlock, err := snapshot.rw.rw.GetLastUnpublishedSeedSnapshotHeader(block.Producer(), electionResult.STime)
		if err != nil {
			return false, err
		}
		if seedBlock == nil {
			return false, errors.New("get last seed snapshot header nil.")
		}
		hash := ledger.ComputeSeedHash(block.Seed, seedBlock.PrevHash, seedBlock.Timestamp)
		if hash != *seedBlock.SeedHash {
			return false, errors.Errorf("seed verify fail. %s-%d", seedBlock.Hash, seedBlock.Height)
		}
	}
	return true, nil
}
