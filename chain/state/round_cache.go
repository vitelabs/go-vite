package chain_state

import (
	"fmt"
	"github.com/vitelabs/go-vite/common/db"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
)

// TODO
type RoundCacheRedoLogs struct{}

type RedoCacheData struct {
	CurrentData *db.MemDB
	RedoLogs    *RoundCacheRedoLogs
}

func NewRedoCacheData(currentData *db.MemDB, redoLogs *RoundCacheRedoLogs) *RedoCacheData {
	return &RedoCacheData{
		CurrentData: currentData,
		RedoLogs:    redoLogs,
	}
}

type RoundCache struct {
	roundCount uint64
	timeIndex  core.TimeIndex // TODO new timeIndex: GetPeriodTimeIndex() TimeIndex

	data map[uint64]*RedoCacheData
}

func NewRoundCache(roundCount uint8) *RoundCache {
	return &RoundCache{
		roundCount: uint64(roundCount),
		data:       make(map[uint64]*RedoCacheData, roundCount),
	}
}

// build data
func (cache *RoundCache) Init(latestSb *ledger.SnapshotBlock) error {
	// clean data
	cache.data = make(map[uint64]*RedoCacheData, cache.roundCount)

	// get latest round index
	latestTime := latestSb.Timestamp

	roundIndex := cache.timeIndex.Time2Index(*latestTime)

	if roundIndex <= cache.roundCount {
		return nil
	}

	startRoundIndex := roundIndex - cache.roundCount + 1

	// [startRoundIndex - roundIndex]

	// init first currentData
	firstCurrentData, err := cache.queryCurrentData(startRoundIndex)
	if err != nil {
		return err
	}
	if firstCurrentData == nil {
		return errors.New(fmt.Sprintf("round index is %d, firstCurrentData is nil", startRoundIndex))
	}

	// init first redo logs
	firstRedoLogs, err := cache.queryRedoLogs(startRoundIndex)
	if err != nil {
		return err
	}

	cache.data[startRoundIndex] = NewRedoCacheData(firstCurrentData, firstRedoLogs)

	prevCurrentData := firstCurrentData
	for index := startRoundIndex + 1; index <= roundIndex; index++ {
		redoLogs, err := cache.queryRedoLogs(index)
		if err != nil {
			return err
		}

		curCurrentData := cache.buildCurrentData(prevCurrentData, redoLogs)

		prevCurrentData = curCurrentData

		cache.data[startRoundIndex] = NewRedoCacheData(curCurrentData, redoLogs)
	}

	return nil
}

// TODO
func (cache *RoundCache) queryCurrentData(roundIndex uint64) (*db.MemDB, error) {
	roundData := db.NewMemDB()

	return roundData, nil
}

// TODO
func (cache *RoundCache) queryRedoLogs(roundIndex uint64) (*RoundCacheRedoLogs, error) {
	// 1. 查出本轮所有snapshot block
	// 2. 根据snapshot block查出所有redoLog，转换为RoundCacheRedoLogs
	return nil, nil
}

// TODO
func (cache *RoundCache) buildCurrentData(prevBaseData *db.MemDB, redoLogs *RoundCacheRedoLogs) *db.MemDB {
	return nil
}
