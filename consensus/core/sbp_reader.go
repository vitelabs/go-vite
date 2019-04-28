package core

import (
	"fmt"
	"math/big"

	"github.com/vitelabs/go-vite/common/types"
)

type BigInt struct {
	*big.Int
}

func (i *BigInt) MarshalText() ([]byte, error) {
	if i == nil || i.Int == nil {
		return []byte("0"), nil
	}
	return []byte(fmt.Sprintf(`"%s"`, i.String())), nil
}

func (x *BigInt) MarshalJSON() ([]byte, error) {
	return x.MarshalText()
}

type SbpStats struct {
	Index            uint64  `json:"index"`
	BlockNum         uint64  `json:"blockNum"`
	ExceptedBlockNum uint64  `json:"exceptedBlockNum"`
	VoteCnt          *BigInt `json:"voteCnt"`
	Name             string  `json:"name"`
}

type DayStats struct {
	Index uint64               `json:"index"`
	Stats map[string]*SbpStats `json:"stats"`

	VoteSum *BigInt `json:"voteSum"`
	// block total in one day
	BlockTotal uint64 `json:"blockTotal"`
}

type BaseStats struct {
	Index uint64                      `json:"index"`
	Stats map[types.Address]*SbpStats `json:"stats"`
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

	GetSuccessRateByHour(index uint64) (map[types.Address]int32, error)
}
