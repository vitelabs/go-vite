package util

import (
	"github.com/vitelabs/go-vite/consensus/core"
	"time"
)

// ConsensusReader defines interfaces of consensus reader in vm module used by sbp reward calculation
type ConsensusReader interface {
	GetIndexByStartTime(t int64, genesisTime int64) (startIndex uint64, startTime int64, drained bool)
	GetIndexByEndTime(t int64, genesisTime int64) (endIndex uint64, endTime int64, withinADay bool)
	GetIndexByTime(t int64, genesisTime int64) uint64
	GetEndTimeByIndex(index uint64) int64
	GetConsensusDetailByDay(startIndex, endIndex uint64) ([]*core.DayStats, error)
}

type sbpStatReader interface {
	DayStats(startIndex uint64, endIndex uint64) ([]*core.DayStats, error)
	GetDayTimeIndex() core.TimeIndex
}

// VMConsensusReader implements consensus reader interfaces
type VMConsensusReader struct {
	reader sbpStatReader
}

// NewVMConsensusReader generate VMConsensusReader instance
func NewVMConsensusReader(sr sbpStatReader) *VMConsensusReader {
	return &VMConsensusReader{reader: sr}
}

// GetIndexByStartTime calculate first cycle index after t
// Parameters:
//   t: timestamp
//   genesisTime: genesis snapshot block timestamp
// Returns:
//   1: first cycle index after input timestamp
//   2: start time of cycle
//   3: is reward drained, true - reward is drained,
//      false - reward is not drained
// Notes:
//   index 0 : (genesis time, genesis time + 24h]
//   index 1: (genesis time + 24h, genesis time + 48h]
//   get first index after t and index start time
//   if t == -1, reward drained
//   if t == 0, return index 0
//   if t ∈ (genesis time, genesis time + 24h], return index 1
func (r *VMConsensusReader) GetIndexByStartTime(t int64, genesisTime int64) (uint64, int64, bool) {
	if t < 0 {
		return 0, 0, true
	}
	if t <= genesisTime {
		return 0, genesisTime, false
	}
	index := r.getIndexByTime(t-1) + 1
	startTime := r.getStartTimeByIndex(index)
	return index, startTime, false
}

// GetIndexByEndTime calculate first cycle index before t
// Parameters:
//   t: timestamp
//   genesisTime: genesis snapshot block timestamp
// Returns:
//   1: first cycle index before input timestamp
//   2: start time of cycle
//   3: is reward drained, true - reward is drained,
//      false - reward is not drained
// Notes:
// if t <= genesis time + 24h, return index 0, within one day
// if t ∈ (genesis time + 24h, genesis time + 48h], return index 0
func (r *VMConsensusReader) GetIndexByEndTime(t int64, genesisTime int64) (uint64, int64, bool) {
	if t < genesisTime {
		return 0, genesisTime, true
	}
	index := r.getIndexByTime(t)
	if index == 0 {
		return 0, genesisTime, true
	}
	index = index - 1
	return index, r.getEndTimeByIndex(index), false
}

// GetIndexByTime calculate cycle index of t
// Parameters:
//   t: timestamp
//   genesisTime: genesis snapshot block timestamp
// Returns:
//   cycle index of input timestamp
func (r *VMConsensusReader) GetIndexByTime(t int64, genesisTime int64) uint64 {
	if t < genesisTime {
		return 0
	}
	return r.getIndexByTime(t)
}

// GetEndTimeByIndex calculate cycle end time of index
// Parameters:
//   index: cycle index
// Returns:
//   end time of input cycle
func (r *VMConsensusReader) GetEndTimeByIndex(index uint64) int64 {
	_, endtime := r.reader.GetDayTimeIndex().Index2Time(index)
	return endtime.Unix()
}

// GetConsensusDetailByDay returns the top 100 sbp block producing details
// of each cycle. Used for calculating sbp reward.
// Parameters:
//   startIndex: start cycle index(included)
//   endIndex: end cycle index(included)
// Returns:
//   1: block producing details of each cycle
//   2: db error
func (r *VMConsensusReader) GetConsensusDetailByDay(startIndex, endIndex uint64) ([]*core.DayStats, error) {
	return r.reader.DayStats(startIndex, endIndex)
}

func (r *VMConsensusReader) getIndexByTime(t int64) uint64 {
	return r.reader.GetDayTimeIndex().Time2Index(time.Unix(t, 0))
}
func (r *VMConsensusReader) getStartTimeByIndex(index uint64) int64 {
	t, _ := r.reader.GetDayTimeIndex().Index2Time(index)
	return t.Unix()
}
func (r *VMConsensusReader) getEndTimeByIndex(index uint64) int64 {
	_, t := r.reader.GetDayTimeIndex().Index2Time(index)
	return t.Unix()
}
