package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"testing"
	"time"
)

var (
	genesisTime           = int64(1546275661)
	oneDay                = int64(150)
	limit                 = int64(75)
	oneDayTimeWithinLimit = time.Unix(genesisTime+oneDay+limit-1, 0)
	oneDayTime            = time.Unix(genesisTime+oneDay+limit, 0)
	twoDayTimeWithinLimit = time.Unix(genesisTime+oneDay*2+limit-1, 0)
	twoDayTime            = time.Unix(genesisTime+oneDay*2+limit, 0)
	firstDayIndex         = uint64(0)
	secondDayIndex        = uint64(1)
	pledgeAmountForTest   = big.NewInt(100)
)

func TestCalcReward(t *testing.T) {
	testCases := []struct {
		registration       *types.Registration
		detailMap          map[uint64]map[string]*consensusDetail
		current            *ledger.SnapshotBlock
		startTime, endTime int64
		drained            bool
		reward             *big.Int
		err                error
		name               string
	}{
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, 0, genesisTime + oneDay, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTimeWithinLimit},
			genesisTime, genesisTime, false, big.NewInt(0), nil, "canceled_not_due",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, 0, genesisTime + oneDay, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime, genesisTime + oneDay, true, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "canceled",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, genesisTime + oneDay, 0, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {10, 20, big.NewInt(100)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime + oneDay, genesisTime + oneDay, false, big.NewInt(0), nil, "reward_drained",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, genesisTime + oneDay + 1, 0, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {10, 20, big.NewInt(100)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime + oneDay*2, genesisTime + oneDay, false, big.NewInt(0), nil, "reward_not_due",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, 0, 0, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime, genesisTime + oneDay, false, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "reward_time_before_genesis_time",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, genesisTime + 1, 0, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
				secondDayIndex: {
					"s1": {600, 800, big.NewInt(500)},
					"s2": {200, 400, big.NewInt(100)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &twoDayTime},
			genesisTime + oneDay, genesisTime + oneDay*2, false, new(big.Int).Mul(rewardPerBlock, big.NewInt(525)), nil, "first_day_no_reward",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, genesisTime, genesisTime + oneDay + 1, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
				secondDayIndex: {
					"s1": {600, 800, big.NewInt(500)},
					"s2": {200, 400, big.NewInt(100)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &twoDayTime},
			genesisTime, genesisTime + oneDay, true, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "last_day_no_reward",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1},
			types.Address{}, nil, 0, genesisTime, 0, nil},
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
				secondDayIndex: {
					"s1": {600, 800, big.NewInt(500)},
					"s2": {200, 400, big.NewInt(100)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &twoDayTimeWithinLimit},
			genesisTime, genesisTime + oneDay, false, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "reward_time_limit",
		},
	}
	InitContractsConfig(true)
	for _, testCase := range testCases {
		reader := util.NewVmConsensusReader(newConsensusReaderTest(genesisTime, oneDay, testCase.detailMap))
		startTime, endTime, reward, drained, err := calcReward(testCase.registration, genesisTime, pledgeAmountForTest, testCase.current, reader)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v CalcReward failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && (startTime != testCase.startTime || drained != testCase.drained || endTime != testCase.endTime || testCase.reward.Cmp(reward) != 0) {
			t.Fatalf("%v CalcReward failed, result not match, expected (%v,%v,%v,%v), got (%v,%v,%v,%v)", testCase.name, testCase.startTime, testCase.endTime, testCase.drained, testCase.reward, startTime, endTime, drained, reward)
		}
	}
}

func TestCalcRewardByDay(t *testing.T) {
	testCases := []struct {
		detailMap map[uint64]map[string]*consensusDetail
		day       int64
		rewardMap map[string]*big.Int
		err       error
		name      string
	}{
		{
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
				secondDayIndex: {
					"s1": {600, 800, big.NewInt(500)},
					"s2": {200, 400, big.NewInt(100)},
				},
			},
			genesisTime,
			map[string]*big.Int{
				"s1": new(big.Int).Mul(rewardPerBlock, big.NewInt(150)),
				"s2": new(big.Int).Mul(rewardPerBlock, big.NewInt(525)),
			},
			nil,
			"first_day",
		},
		{
			map[uint64]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
				secondDayIndex: {
					"s1": {600, 800, big.NewInt(500)},
					"s2": {200, 400, big.NewInt(100)},
				},
			},
			genesisTime + oneDay,
			map[string]*big.Int{
				"s1": new(big.Int).Mul(rewardPerBlock, big.NewInt(525)),
				"s2": new(big.Int).Mul(rewardPerBlock, big.NewInt(150)),
			},
			nil,
			"second_day",
		},
	}
	InitContractsConfig(true)
	for _, testCase := range testCases {
		reader := util.NewVmConsensusReader(newConsensusReaderTest(genesisTime, oneDay, testCase.detailMap))
		rewardMap, err := calcRewardByDay(reader, genesisTime, testCase.day, pledgeAmountForTest)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v CalcRewardByDay failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil {
			if len(rewardMap) != len(testCase.rewardMap) {
				t.Fatalf("%v CalcRewardByDay failed, rewardMap len not match, expected %v, got %v", testCase.name, len(testCase.rewardMap), len(rewardMap))
			} else {
				for k, v := range rewardMap {
					if expectedV, ok := testCase.rewardMap[k]; !ok || v.Cmp(expectedV) != 0 {
						t.Fatalf("%v CalcRewardByDay failed, rewardMap not match, expected %v:%v, got %v:%v", testCase.name, k, expectedV, k, v)
					}
				}
			}
		}
	}

}

type consensusReaderTest struct {
	detailMap map[uint64]map[string]*consensusDetail
	ti        timeIndex
}

type consensusDetail struct {
	blockNum         uint64
	expectedBlockNum uint64
	voteCount        *big.Int
}

func newConsensusReaderTest(genesisTime int64, interval int64, detailMap map[uint64]map[string]*consensusDetail) *consensusReaderTest {
	return &consensusReaderTest{detailMap, timeIndex{time.Unix(genesisTime, 0), time.Second * time.Duration(interval)}}
}

func (r *consensusReaderTest) DayStats(startIndex uint64, endIndex uint64) ([]*core.DayStats, error) {
	list := make([]*core.DayStats, 0)
	for i := startIndex; i <= endIndex; i++ {
		if i > endIndex {
			break
		}
		m, ok := r.detailMap[i]
		if !ok {
			continue
		}
		blockNum := uint64(0)
		expectedBlockNum := uint64(0)
		voteCount := big.NewInt(0)
		statusMap := make(map[string]*core.SbpStats, len(m))
		for name, detail := range m {
			blockNum = blockNum + detail.blockNum
			expectedBlockNum = expectedBlockNum + detail.expectedBlockNum
			voteCount.Add(voteCount, detail.voteCount)
			statusMap[name] = &core.SbpStats{i, detail.blockNum, detail.expectedBlockNum, detail.voteCount, name}
		}
		list = append(list, &core.DayStats{Index: i, Stats: statusMap, VoteSum: voteCount, BlockTotal: blockNum})
	}
	return list, nil
}
func (r *consensusReaderTest) GetDayTimeIndex() core.TimeIndex {
	return r.ti
}

type timeIndex struct {
	GenesisTime time.Time
	Interval    time.Duration
}

func (self timeIndex) Index2Time(index uint64) (time.Time, time.Time) {
	sTime := self.GenesisTime.Add(self.Interval * time.Duration(index))
	eTime := self.GenesisTime.Add(self.Interval * time.Duration(index+1))
	return sTime, eTime
}
func (self timeIndex) Time2Index(t time.Time) uint64 {
	subSec := int64(t.Sub(self.GenesisTime).Seconds())
	i := uint64(subSec) / uint64(self.Interval.Seconds())
	return i
}
