package core

import (
	"time"
)

type TimeIndex interface {
	Index2Time(index uint64) (time.Time, time.Time)
	Time2Index(t time.Time) uint64
}

type timeIndex struct {
	GenesisTime time.Time
	// second
	Interval time.Duration
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

func NewTimeIndex(t time.Time, interval time.Duration) TimeIndex {
	return &timeIndex{GenesisTime: t, Interval: interval}
}
