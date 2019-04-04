package contracts

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/vm/util"
	"math/big"
	"time"
)

type consensusReader interface {
	TimeToPeriodIndex(time time.Time) uint64
	TimeToRewardStartIndex(t int64) uint64
	TimeToRewardEndIndex(t int64) uint64
	TimeToRewardStartDayTime(currentTime int64) int64
	TimeToRewardEndDayTime(currentTime int64) int64
	GetIndexInDay() uint64
	GetConsensusDetailByDay(startIndex, endIndex uint64) (detailMap map[string]*consensusDetail, summary *consensusDetail)
	GenesisTime() int64
	GroupInfo() *types.ConsensusGroupInfo
}
type vmConsensusReader struct {
	reader      core.ConsensusReader
	genesisTime int64
	groupInfo   *types.ConsensusGroupInfo
}

func newVmConsensusReader(genesisTime *time.Time, groupInfo *types.ConsensusGroupInfo) *vmConsensusReader {
	return &vmConsensusReader{core.NewReader(*genesisTime, groupInfo), genesisTime.Unix(), groupInfo}
}

func (r *vmConsensusReader) TimeToPeriodIndex(time time.Time) uint64 {
	i, err := r.reader.TimeToIndex(time)
	util.DealWithErr(err)
	return i
}

func (r *vmConsensusReader) TimeToRewardStartIndex(t int64) uint64 {
	startDayTime := r.TimeToRewardStartDayTime(t)
	i, err := r.reader.TimeToIndex(time.Unix(startDayTime, 0))
	util.DealWithErr(err)
	return i
}

func (r *vmConsensusReader) TimeToRewardEndIndex(t int64) uint64 {
	endDayTime := r.TimeToRewardEndDayTime(t)
	i, err := r.reader.TimeToIndex(time.Unix(endDayTime, 0))
	util.DealWithErr(err)
	return i
}

// Inclusive
func (r *vmConsensusReader) TimeToRewardStartDayTime(currentTime int64) int64 {
	delta := (currentTime - r.genesisTime + nodeConfig.params.RewardTimeUnit - 1) / nodeConfig.params.RewardTimeUnit
	return r.genesisTime + delta*nodeConfig.params.RewardTimeUnit
}

// Exclusive
func (r *vmConsensusReader) TimeToRewardEndDayTime(currentTime int64) int64 {
	delta := (currentTime - r.genesisTime) / nodeConfig.params.RewardTimeUnit
	return r.genesisTime + delta*nodeConfig.params.RewardTimeUnit
}

func (r *vmConsensusReader) GetIndexInDay() uint64 {
	periodTime, err := r.reader.PeriodTime()
	util.DealWithErr(err)
	return uint64(nodeConfig.params.RewardTimeUnit) / periodTime
}

type consensusDetail struct {
	blockNum         uint64
	expectedBlockNum uint64
	voteCount        *big.Int
}

func (r *vmConsensusReader) GetConsensusDetailByDay(startIndex, endIndex uint64) (detailMap map[string]*consensusDetail, summary *consensusDetail) {
	// TODO
	return make(map[string]*consensusDetail), &consensusDetail{0, 0, big.NewInt(0)}
}

func (r *vmConsensusReader) GenesisTime() int64 {
	return r.genesisTime
}

func (r *vmConsensusReader) GroupInfo() *types.ConsensusGroupInfo {
	return r.groupInfo
}
