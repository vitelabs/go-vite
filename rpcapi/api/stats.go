package api

import (
	"time"

	"github.com/vitelabs/go-vite/consensus"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite"
)

type StatsApi struct {
	cs  consensus.Consensus
	log log15.Logger
}

func NewStatsApi(vite *vite.Vite) *StatsApi {
	return &StatsApi{
		cs:  vite.Consensus(),
		log: log15.New("module", "rpc_api/stats_api"),
	}
}

func (c StatsApi) String() string {
	return "StatsApi"
}

func (c StatsApi) Time2Index(t *time.Time) uint64 {
	if t == nil {
		now := time.Now()
		t = &now
	}
	index := c.cs.SBPReader().GetPeriodTimeIndex()
	time2Index := index.Time2Index(*t)
	return time2Index
}
func (c StatsApi) Index2Time(i uint64) map[string]time.Time {
	result := make(map[string]time.Time)
	index := c.cs.SBPReader().GetPeriodTimeIndex()
	stime, etime := index.Index2Time(i)
	result["stime"] = stime
	result["etime"] = etime
	return result
}

func (c StatsApi) GetHourSBPStats(startIdx uint64, endIdx uint64) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	reader := c.cs.SBPReader()

	timeIndex := reader.GetHourTimeIndex()
	if startIdx > endIdx {
		startIdx, endIdx = c.reIndex(timeIndex)
	}
	// hour
	stats, err := reader.HourStats(startIdx, endIdx)
	if err != nil {
		return nil, err
	}

	for _, v := range stats {
		r := make(map[string]interface{})
		stime, etime := timeIndex.Index2Time(v.Index)

		r["stime"] = stime.String()
		r["etime"] = etime.String()
		r["stat"] = v

		result = append(result, r)
	}
	return result, nil
}

func (c StatsApi) GetPeriodSBPStats(startIdx uint64, endIdx uint64) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	reader := c.cs.SBPReader()

	timeIndex := reader.GetPeriodTimeIndex()
	if startIdx > endIdx {
		startIdx, endIdx = c.reIndex(timeIndex)
	}
	// hour
	stats, err := reader.PeriodStats(startIdx, endIdx)
	if err != nil {
		return nil, err
	}

	for _, v := range stats {
		r := make(map[string]interface{})
		stime, etime := timeIndex.Index2Time(v.Index)

		r["stime"] = stime.String()
		r["etime"] = etime.String()
		r["stat"] = v

		result = append(result, r)
	}
	return result, nil
}

func (c StatsApi) GetDaySBPStats(startIdx uint64, endIdx uint64) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	reader := c.cs.SBPReader()
	timeIndex := reader.GetDayTimeIndex()
	if startIdx > endIdx {
		startIdx, endIdx = c.reIndex(timeIndex)
	}
	// day
	stats, err := reader.DayStats(startIdx, endIdx)
	if err != nil {
		return nil, err
	}

	for _, v := range stats {
		r := make(map[string]interface{})
		stime, etime := timeIndex.Index2Time(v.Index)

		r["stime"] = stime.String()
		r["etime"] = etime.String()
		r["stat"] = v

		result = append(result, r)
	}
	return result, nil
}

func (c StatsApi) reIndex(timeIndex core.TimeIndex) (uint64, uint64) {
	startIdx := uint64(0)
	endIdx := timeIndex.Time2Index(time.Now())
	N := uint64(5)
	if endIdx >= N {
		startIdx = endIdx - N
	}
	return startIdx, endIdx
}
