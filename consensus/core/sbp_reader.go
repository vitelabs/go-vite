package core

import "math/big"

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
}

type SBPStatReader interface {
	DayStats(startIndex uint64, endIndex uint64) ([]*DayStats, error)
	GetDayTimeIndex() TimeIndex
}
