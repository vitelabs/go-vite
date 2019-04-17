package core

import (
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
)

type SbpStats struct {
	Index            uint64
	BlockNum         uint64
	ExceptedBlockNum uint64
	VoteCnt          *big.Int
	Name             string
}

type DayStats struct {
	Index uint64
	Stats map[string]*SbpStats

	VoteSum *big.Int
	// block total in one day
	BlockTotal uint64
}

type BaseStats struct {
	Index uint64
	Stats map[types.Address]*SbpStats
}

type HourStats struct {
	*BaseStats
}

type PeriodStats struct {
	*BaseStats
}

type SBPStatReader interface {
	DayStats(startIndex uint64, endIndex uint64) ([]*DayStats, error)
	GetDayTimeIndex() TimeIndex

	HourStats(startIndex uint64, endIndex uint64) ([]*HourStats, error)
	GetHourTimeIndex() TimeIndex

	PeriodStats(startIndex uint64, endIndex uint64) ([]*PeriodStats, error)
	GetPeriodTimeIndex() TimeIndex
}
