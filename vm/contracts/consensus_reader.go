package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"time"
)

type consensusReader struct {
	reader      core.ConsensusReader
	genesisTime int64
}

func newConsensusReader(genesisTime *time.Time, groupInfo *types.ConsensusGroupInfo) *consensusReader {
	return &consensusReader{core.NewReader(*genesisTime, groupInfo), genesisTime.Unix()}
}

func (r *consensusReader) timeToPeriodIndex(time time.Time) uint64 {
	i, err := r.reader.TimeToIndex(time)
	util.DealWithErr(err)
	return i
}

func (r *consensusReader) timeToRewardStartIndex(t int64) uint64 {
	startDayTime := r.timeToRewardStartDayTime(t)
	i, err := r.reader.TimeToIndex(time.Unix(startDayTime, 0))
	util.DealWithErr(err)
	return i
}

func (r *consensusReader) timeToRewardEndIndex(t int64) uint64 {
	endDayTime := r.timeToRewardEndDayTime(t)
	i, err := r.reader.TimeToIndex(time.Unix(endDayTime, 0))
	util.DealWithErr(err)
	return i
}

// Inclusive
func (r *consensusReader) timeToRewardStartDayTime(currentTime int64) int64 {
	delta := (currentTime - r.genesisTime - 1 + nodeConfig.params.RewardTimeUnit) / nodeConfig.params.RewardTimeUnit
	return r.genesisTime + delta*nodeConfig.params.RewardTimeUnit
}

// Exclusive
func (r *consensusReader) timeToRewardEndDayTime(currentTime int64) int64 {
	delta := (currentTime - r.genesisTime) / nodeConfig.params.RewardTimeUnit
	return r.genesisTime + delta*nodeConfig.params.RewardTimeUnit
}

func (r *consensusReader) getIndexInDay() uint64 {
	periodTime, err := r.reader.PeriodTime()
	util.DealWithErr(err)
	return uint64(nodeConfig.params.RewardTimeUnit) / periodTime
}

type consensusDetail struct {
	blockNum         uint64
	expectedBlockNum uint64
	voteCount        *big.Int
}

func (r *consensusReader) getConsensusDetailByDay(startIndex, endIndex uint64) (detailMap map[string]*consensusDetail, summary *consensusDetail) {
	// TODO
	return make(map[string]*consensusDetail), &consensusDetail{0, 0, big.NewInt(0)}
}
