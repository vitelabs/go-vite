package test_tools

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"time"
)

type MockConsensus struct {
	Cr *MockConsensusReader
}

func (c *MockConsensus) SBPReader() core.SBPStatReader {
	return c.Cr
}

func (c *MockConsensus) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	return true, nil
}

type MockCssVerifier struct{}

func (c *MockCssVerifier) VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	return nil, nil
}

func (c *MockCssVerifier) VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error) {
	return true, nil
}

func (c *MockCssVerifier) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	return true, nil
}

type MockConsensusReader struct {
	DayTimeIndex MockTimeIndex
}

func (c *MockConsensusReader) DayStats(startIndex uint64, endIndex uint64) ([]*core.DayStats, error) {
	return nil, nil
}
func (c *MockConsensusReader) GetDayTimeIndex() core.TimeIndex {
	return c.DayTimeIndex
}
func (c *MockConsensusReader) HourStats(startIndex uint64, endIndex uint64) ([]*core.HourStats, error) {
	return nil, nil
}
func (c *MockConsensusReader) GetHourTimeIndex() core.TimeIndex {
	return nil
}
func (c *MockConsensusReader) PeriodStats(startIndex uint64, endIndex uint64) ([]*core.PeriodStats, error) {
	return nil, nil
}
func (c *MockConsensusReader) GetPeriodTimeIndex() core.TimeIndex {
	return nil
}
func (c *MockConsensusReader) GetSuccessRateByHour(index uint64) (map[types.Address]int32, error) {
	return nil, nil
}
func (c *MockConsensusReader) GetNodeCount() int {
	return 1
}

type MockTimeIndex struct {
	GenesisTime time.Time
	Interval    time.Duration
}

func (self MockTimeIndex) Index2Time(index uint64) (time.Time, time.Time) {
	sTime := self.GenesisTime.Add(self.Interval * time.Duration(index))
	eTime := self.GenesisTime.Add(self.Interval * time.Duration(index+1))
	return sTime, eTime
}
func (self MockTimeIndex) Time2Index(t time.Time) uint64 {
	subSec := int64(t.Sub(self.GenesisTime).Seconds())
	i := uint64(subSec) / uint64(self.Interval.Seconds())
	return i
}
