package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm/contracts/abi"
	"math/big"
	"strconv"
	"testing"
	"time"
)

var (
	genesisTime           = int64(1546275661)
	oneDay                = int64(150)
	limit                 = int64(75)
	oneDayTime            = time.Unix(genesisTime+oneDay+limit, 0)
	twoDayTimeWithinLimit = time.Unix(genesisTime+oneDay*2+limit-1, 0)
	twoDayTime            = time.Unix(genesisTime+oneDay*2+limit, 0)
	firstDayIndex         = "0-2"
	secondDayIndex        = "2-4"
	pledgeAmountForTest   = big.NewInt(100)
)

func TestCalcReward(t *testing.T) {
	testCases := []struct {
		registration       *types.Registration
		detailMap          map[string]map[string]*consensusDetail
		current            *ledger.SnapshotBlock
		startTime, endTime int64
		reward             *big.Int
		err                error
		name               string
	}{
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, types.Address{}, nil, 0, genesisTime + oneDay, 0, nil},
			map[string]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {10, 20, big.NewInt(100)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime + 150, genesisTime + 150, big.NewInt(0), nil, "reward_drained",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, types.Address{}, nil, 0, genesisTime + oneDay + 1, 0, nil},
			map[string]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {10, 20, big.NewInt(100)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime + 300, genesisTime + oneDay, big.NewInt(0), nil, "reward_not_due",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, types.Address{}, nil, 0, 0, genesisTime + oneDay, nil},
			map[string]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime, genesisTime + oneDay, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "canceled",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, types.Address{}, nil, 0, -1, 0, nil},
			map[string]map[string]*consensusDetail{
				firstDayIndex: {
					"s1": {200, 400, big.NewInt(100)},
					"s2": {600, 800, big.NewInt(500)},
				},
			},
			&ledger.SnapshotBlock{Timestamp: &oneDayTime},
			genesisTime, genesisTime + oneDay, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "reward_time_before_genesis_time",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, types.Address{}, nil, 0, genesisTime + 1, 0, nil},
			map[string]map[string]*consensusDetail{
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
			genesisTime + oneDay, genesisTime + oneDay*2, new(big.Int).Mul(rewardPerBlock, big.NewInt(525)), nil, "first_day_no_reward",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, types.Address{}, nil, 0, genesisTime, genesisTime + oneDay + 1, nil},
			map[string]map[string]*consensusDetail{
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
			genesisTime, genesisTime + oneDay, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "last_day_no_reward",
		},
		{&types.Registration{"s1", types.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}, types.Address{}, nil, 0, genesisTime, 0, nil},
			map[string]map[string]*consensusDetail{
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
			genesisTime, genesisTime + oneDay, new(big.Int).Mul(rewardPerBlock, big.NewInt(150)), nil, "reward_time_limit",
		},
	}
	InitContractsConfig(true)
	conditionRegisterData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge, pledgeAmountForTest, ledger.ViteTokenId, uint64(75*2))
	groupInfo := &types.ConsensusGroupInfo{types.SNAPSHOT_GID, 25, 3, 1, 2, 100, 1, 0, ledger.ViteTokenId, 1, conditionRegisterData, 0, nil, types.Address{}, nil, 0}
	for _, testCase := range testCases {
		reader := newConsensusReaderTest(genesisTime, groupInfo, testCase.detailMap)
		startTime, endTime, reward, err := calcReward(testCase.registration, testCase.current, reader)
		if (err == nil && testCase.err != nil) || (err != nil && testCase.err == nil) || (err != nil && testCase.err != nil && err.Error() != testCase.err.Error()) {
			t.Fatalf("%v CalcReward failed, error not match, expected %v, got %v", testCase.name, testCase.err, err)
		}
		if err == nil && (startTime != testCase.startTime || endTime != testCase.endTime || testCase.reward.Cmp(reward) != 0) {
			t.Fatalf("%v CalcReward failed, result not match, expected (%v,%v,%v), got (%v,%v,%v)", testCase.name, testCase.startTime, testCase.endTime, testCase.reward, startTime, endTime, reward)
		}
	}
}

func TestCalcRewardByDay(t *testing.T) {
	testCases := []struct {
		detailMap map[string]map[string]*consensusDetail
		day       int64
		rewardMap map[string]*big.Int
		err       error
		name      string
	}{
		{
			map[string]map[string]*consensusDetail{
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
			map[string]map[string]*consensusDetail{
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
	conditionRegisterData, _ := abi.ABIConsensusGroup.PackVariable(abi.VariableNameConditionRegisterOfPledge, pledgeAmountForTest, ledger.ViteTokenId, uint64(75*2))
	groupInfo := &types.ConsensusGroupInfo{types.SNAPSHOT_GID, 25, 3, 1, 2, 100, 1, 0, ledger.ViteTokenId, 1, conditionRegisterData, 0, nil, types.Address{}, nil, 0}
	for _, testCase := range testCases {
		reader := newConsensusReaderTest(genesisTime, groupInfo, testCase.detailMap)
		rewardMap, err := calcRewardByDay(reader, testCase.day, pledgeAmountForTest)
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
	genesisTime int64
	groupInfo   *types.ConsensusGroupInfo
	detailMap   map[string]map[string]*consensusDetail
}

func newConsensusReaderTest(genesisTime int64, groupInfo *types.ConsensusGroupInfo, detailMap map[string]map[string]*consensusDetail) *consensusReaderTest {
	return &consensusReaderTest{genesisTime, groupInfo, detailMap}
}

func planInterval(info *types.ConsensusGroupInfo) uint64 {
	return uint64(info.Interval) * uint64(info.NodeCount) * uint64(info.PerCount)
}

func timeToIndex(time, genesisTime int64, interval uint64) uint64 {
	return uint64(time-genesisTime) / interval
}

func (r *consensusReaderTest) TimeToPeriodIndex(time time.Time) uint64 {
	return timeToIndex(time.Unix(), r.genesisTime, planInterval(r.groupInfo))
}

func (r *consensusReaderTest) TimeToRewardStartIndex(t int64) uint64 {
	startDayTime := r.TimeToRewardStartDayTime(t)
	return timeToIndex(startDayTime, r.genesisTime, planInterval(r.groupInfo))
}

func (r *consensusReaderTest) TimeToRewardEndIndex(t int64) uint64 {
	endDayTime := r.TimeToRewardEndDayTime(t)
	return timeToIndex(endDayTime, r.genesisTime, planInterval(r.groupInfo))
}

// Inclusive
func (r *consensusReaderTest) TimeToRewardStartDayTime(currentTime int64) int64 {
	delta := (currentTime - r.genesisTime + nodeConfig.params.RewardTimeUnit - 1) / nodeConfig.params.RewardTimeUnit
	return r.genesisTime + delta*nodeConfig.params.RewardTimeUnit
}

// Exclusive
func (r *consensusReaderTest) TimeToRewardEndDayTime(currentTime int64) int64 {
	delta := (currentTime - r.genesisTime) / nodeConfig.params.RewardTimeUnit
	return r.genesisTime + delta*nodeConfig.params.RewardTimeUnit
}

func (r *consensusReaderTest) GetIndexInDay() uint64 {
	return uint64(nodeConfig.params.RewardTimeUnit) / planInterval(r.groupInfo)
}

func (r *consensusReaderTest) GetConsensusDetailByDay(startIndex, endIndex uint64) (map[string]*consensusDetail, *consensusDetail) {
	blockNum := uint64(0)
	expectedBlockNum := uint64(0)
	voteCount := big.NewInt(0)
	index := strconv.Itoa(int(startIndex)) + "-" + strconv.Itoa(int(endIndex))
	for _, detail := range r.detailMap[index] {
		blockNum = blockNum + detail.blockNum
		expectedBlockNum = expectedBlockNum + detail.expectedBlockNum
		voteCount.Add(voteCount, detail.voteCount)
	}
	return r.detailMap[index], &consensusDetail{blockNum, expectedBlockNum, voteCount}
}

func (r *consensusReaderTest) GenesisTime() int64 {
	return r.genesisTime
}

func (r *consensusReaderTest) GroupInfo() *types.ConsensusGroupInfo {
	return r.groupInfo
}
