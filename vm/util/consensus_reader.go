package util

import (
	"github.com/vitelabs/go-vite/consensus/core"
	"time"
)

type ConsensusReader interface {
	GetIndexByStartTime(t int64, genesisTime int64) (startIndex uint64, startTime int64, drained bool)
	GetIndexByEndTime(t int64, genesisTime int64) (endIndex uint64, endTime int64, withinADay bool)
	GetIndexByTime(t int64, genesisTime int64) uint64
	GetConsensusDetailByDay(startIndex, endIndex uint64) ([]*core.DayStats, error)
}
type VMConsensusReader struct {
	reader core.SBPStatReader
}

func NewVmConsensusReader(sr core.SBPStatReader) *VMConsensusReader {
	return &VMConsensusReader{reader: sr}
}

// index 0 : (genesis time, genesis time + 24h]
// index 1: (genesis time + 24h, genesis time + 48h]
// get first index after t and index start time
// if t == -1, reward drained
// if t == 0, return index 0
// if t ∈ (genesis time, genesis time + 24h], return index 1
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

// get first index before t and index start time
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
func (r *VMConsensusReader) GetIndexByTime(t int64, genesisTime int64) uint64 {
	if t < genesisTime {
		return 0
	}
	return r.getIndexByTime(t)
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
func (r *VMConsensusReader) GetConsensusDetailByDay(startIndex, endIndex uint64) ([]*core.DayStats, error) {
	return r.reader.DayStats(startIndex, endIndex)
}
