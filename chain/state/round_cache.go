package chain_state

import (
	"fmt"
	chain_utils "github.com/vitelabs/go-vite/chain/utils"
	leveldb "github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/comparer"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/db/xleveldb/memdb"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type RoundCacheLogItem struct {
	Storage    [][2][]byte
	BalanceMap map[types.TokenTypeId]*big.Int
}

type RoundCacheSnapshotLog struct {
	LogMap         map[types.Address][]RoundCacheLogItem
	SnapshotHeight uint64
}

func newRoundCacheSnapshotLog(capacity int, snapshotHeight uint64) *RoundCacheSnapshotLog {
	return &RoundCacheSnapshotLog{
		LogMap:         make(map[types.Address][]RoundCacheLogItem, capacity),
		SnapshotHeight: snapshotHeight,
	}
}

// TODO
type RoundCacheRedoLogs struct {
	Logs []*RoundCacheSnapshotLog
}

// TODO
func NewRoundCacheRedoLogs() *RoundCacheRedoLogs {
	return &RoundCacheRedoLogs{}
}

func (redoLogs *RoundCacheRedoLogs) Add(snapshotHeight uint64, snapshotLog SnapshotLog) {
	filteredSnapshotLog := newRoundCacheSnapshotLog(len(snapshotLog), snapshotHeight)

	for addr, logList := range snapshotLog {
		needSetStorage := addr == types.AddressConsensusGroup

		filteredLogList := make([]RoundCacheLogItem, len(logList))

		for index, logItem := range logList {
			filteredLogItem := RoundCacheLogItem{}

			// set storage
			if needSetStorage {
				filteredLogItem.Storage = logItem.Storage
			}

			// set balance
			filteredLogItem.BalanceMap = logItem.BalanceMap

			filteredLogList[index] = filteredLogItem

		}

		filteredSnapshotLog.LogMap[addr] = filteredLogList
	}

	redoLogs.Logs = append(redoLogs.Logs, filteredSnapshotLog)

	// append snapshot height

}

type RedoCacheData struct {
	RoundIndex uint64

	CurrentData *memdb.DB

	RedoLogs *RoundCacheRedoLogs
}

func NewRedoCacheData(roundIndex uint64, currentData *memdb.DB, redoLogs *RoundCacheRedoLogs) *RedoCacheData {
	return &RedoCacheData{
		RoundIndex:  roundIndex,
		CurrentData: currentData,
		RedoLogs:    redoLogs,
	}
}

type RoundCache struct {
	chain            Chain
	stateDB          *StateDB
	roundCount       uint64
	timeIndex        core.TimeIndex // TODO new timeIndex: GetPeriodTimeIndex() TimeIndex
	latestRoundIndex uint64

	data []*RedoCacheData
}

func NewRoundCache(chain Chain, stateDB *StateDB, roundCount uint8) *RoundCache {
	return &RoundCache{
		chain:      chain,
		stateDB:    stateDB,
		roundCount: uint64(roundCount),
		data:       make([]*RedoCacheData, 0, roundCount),
	}
}

// build data
func (cache *RoundCache) Init(latestSb *ledger.SnapshotBlock) error {
	// clean data
	cache.data = make([]*RedoCacheData, 0, cache.roundCount)

	// get latest round index
	latestTime := latestSb.Timestamp

	roundIndex := cache.timeIndex.Time2Index(*latestTime)

	// set round index of latest snapshot block
	cache.latestRoundIndex = roundIndex

	if roundIndex <= cache.roundCount {
		return nil
	}

	startRoundIndex := roundIndex - cache.roundCount + 1

	roundsData, err := cache.initRounds(startRoundIndex, roundIndex)
	if err != nil {
		return err
	}

	if roundsData != nil {
		cache.data = roundsData
	}

	return nil
}

func (cache *RoundCache) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, snapshotLog SnapshotLog) error {
	roundIndex := cache.timeIndex.Time2Index(*snapshotBlock.Timestamp)

	if roundIndex < cache.latestRoundIndex {
		return errors.New(fmt.Sprintf("roundIndex < cache.latestRoundIndex, %d < %d", roundIndex, cache.latestRoundIndex))
	} else if roundIndex == cache.latestRoundIndex {
		length := len(cache.data)
		// there are no redoCacheData when delete too more snapshot blocks not long ago， or there is no redo logs when init
		if length <= 0 {
			return nil
		}

		currentRedoCacheData := cache.data[length-1]

		currentRedoCacheData.RedoLogs.Add(snapshotBlock.Height, snapshotLog)
		return nil
	}

	// latestRoundIndex is genesis round index
	if cache.latestRoundIndex <= 0 {
		return nil
	}

	// there are no redoCacheData when delete too more snapshot blocks not long ago， or there is no redo logs when init
	dataLength := len(cache.data)
	if dataLength <= 0 {
		roundData, err := cache.queryCurrentData(cache.latestRoundIndex)
		if err != nil {
			return err
		}

		if roundData == nil {
			return errors.New(fmt.Sprintf("roundData is nil, cache.latestRoundIndex is %d", cache.latestRoundIndex))
		}

		cache.data = append(cache.data, NewRedoCacheData(cache.latestRoundIndex, roundData, nil))
	} else if dataLength == 1 {
		return errors.New(fmt.Sprintf("len(cache.data) is 1, cache.data[0] is %+v", cache.data[0]))
	} else {
		prevRoundData := cache.data[dataLength-1]
		if prevRoundData.RedoLogs == nil || len(prevRoundData.RedoLogs.Logs) <= 0 {
			return errors.New(fmt.Sprintf("prevRoundData.RedoLogs == nil || len(prevRoundData.RedoLogs.Logs) <= 0. prevRoundData is %+v", prevRoundData))
		}

		prevBeforePrevRoundData := cache.data[dataLength-2]
		if prevBeforePrevRoundData.CurrentData == nil {
			return errors.New(fmt.Sprintf("prevBeforePrevRoundData.CurrentData == nil, prevBeforePrevRoundData is %+v", prevBeforePrevRoundData))
		}

		// build
		prevRoundData.CurrentData = cache.buildCurrentData(prevBeforePrevRoundData.CurrentData, prevRoundData.RedoLogs)
	}

	redoLogs := NewRoundCacheRedoLogs()
	redoLogs.Add(snapshotBlock.Height, snapshotLog)

	cache.data = append(cache.data, NewRedoCacheData(roundIndex, nil, redoLogs))
	cache.latestRoundIndex = roundIndex

	if uint64(len(cache.data)) > cache.roundCount {
		// remove head
		cache.data = cache.data[1:]
	}
	return nil
}

func (cache *RoundCache) DeleteSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	firstSnapshotBlock := snapshotBlocks[0]
	firstRoundIndex := cache.timeIndex.Time2Index(*firstSnapshotBlock.Timestamp)

	end := len(cache.data)

	for i := end - 1; i >= 0; i-- {
		data := cache.data[i]
		if data.RoundIndex > firstRoundIndex {
			// to be deleted
			end = i
		} else if data.RoundIndex == firstRoundIndex {
			data.CurrentData = nil

			if data.RedoLogs == nil || len(data.RedoLogs.Logs) <= 0 {
				end = i
			} else {
				// remove
				redoLogsEnd := len(data.RedoLogs.Logs)
				for j := redoLogsEnd - 1; j >= 0; j-- {
					redoLog := data.RedoLogs.Logs[j]
					if redoLog.SnapshotHeight >= firstSnapshotBlock.Height {
						redoLogsEnd = j
					} else {
						break
					}
				}

				if redoLogsEnd <= 0 {
					end = i
				} else {
					data.RedoLogs.Logs = data.RedoLogs.Logs[:redoLogsEnd]
				}

			}
		} else if data.RoundIndex < firstRoundIndex {
			break
		}
	}

	if end <= 1 {
		cache.data = cache.data[:0]
	} else {
		cache.data = cache.data[:end]
	}

	// update last round index
	beforeFirstSnapshotBlock, err := cache.chain.GetSnapshotHeaderByHeight(firstSnapshotBlock.Height - 1)
	if err != nil {
		return err
	}
	if beforeFirstSnapshotBlock == nil {
		return errors.New(fmt.Sprintf("beforeFirstSnapshotBlock is nil, height is %d", firstSnapshotBlock.Height-1))
	}
	newLatestRoundIndex := cache.timeIndex.Time2Index(*beforeFirstSnapshotBlock.Timestamp)
	cache.latestRoundIndex = newLatestRoundIndex

	return nil
}

func (cache *RoundCache) initRounds(startRoundIndex, endRoundIndex uint64) ([]*RedoCacheData, error) {
	// [startRoundIndex - roundIndex]
	// init first currentData
	firstCurrentData, err := cache.queryCurrentData(startRoundIndex)
	if err != nil {
		return nil, err
	}
	if firstCurrentData == nil {
		return nil, nil
	}

	// init first redo logs
	firstRedoLogs, err := cache.queryRedoLogs(startRoundIndex)
	if err != nil {
		return nil, err
	}

	roundsData := make([]*RedoCacheData, 0, cache.roundCount)

	roundsData = append(roundsData, NewRedoCacheData(startRoundIndex, firstCurrentData, firstRedoLogs))

	prevCurrentData := firstCurrentData
	for index := startRoundIndex + 1; index <= endRoundIndex; index++ {
		redoLogs, err := cache.queryRedoLogs(index)
		if err != nil {
			return nil, err
		}

		if redoLogs == nil {
			return cache.initRounds(index, endRoundIndex)
		}

		curCurrentData := cache.buildCurrentData(prevCurrentData, redoLogs)

		prevCurrentData = curCurrentData

		roundsData = append(roundsData, NewRedoCacheData(index, curCurrentData, redoLogs))
	}

	return roundsData, nil
}

func (cache *RoundCache) queryCurrentData(roundIndex uint64) (*memdb.DB, error) {
	roundData := memdb.New(leveldb.NewIComparer(comparer.DefaultComparer), 0)

	// get the last snapshot block of round
	snapshotBlock, err := cache.roundToLastSnapshotBlock(roundIndex)
	if err != nil {
		return nil, err
	}
	if snapshotBlock == nil {
		return nil, nil
	}

	// set vote list cache
	if err := cache.setStorageToCache(roundData, types.AddressConsensusGroup, snapshotBlock.Hash); err != nil {
		return nil, err
	}

	// set address balances cache
	if err := cache.setAllBalanceToCache(roundData, snapshotBlock.Hash); err != nil {
		return nil, err
	}

	return roundData, nil
}

func (cache *RoundCache) queryRedoLogs(roundIndex uint64) (*RoundCacheRedoLogs, error) {
	roundRedoLogs := NewRoundCacheRedoLogs()
	// query the snapshot blocks of the round
	snapshotBlocks, err := cache.getRoundSnapshotBlocks(roundIndex)
	if err != nil {
		return nil, err
	}

	if len(snapshotBlocks) <= 0 {
		return roundRedoLogs, nil
	}

	// assume snapshot blocks is sorted by height asc, query redoLog by snapshot block height
	for _, snapshotBlock := range snapshotBlocks {
		redoLog, hasRedoLog, err := cache.stateDB.redo.QueryLog(snapshotBlock.Height)
		if err != nil {
			return nil, err
		}
		if !hasRedoLog {
			return nil, nil
		}

		roundRedoLogs.Add(snapshotBlock.Height, redoLog)
	}

	return roundRedoLogs, nil
}

// TODO optimize: build in parallel
func (cache *RoundCache) buildCurrentData(prevCurrentData *memdb.DB, redoLogs *RoundCacheRedoLogs) *memdb.DB {
	// copy
	curCurrentData := prevCurrentData.Copy()

	for _, snapshotLog := range redoLogs.Logs {
		for addr, redoLogs := range snapshotLog.LogMap {
			for _, redoLog := range redoLogs {
				// set storage
				for _, kv := range redoLog.Storage {
					curCurrentData.Put(makeStorageKey(kv[0]), kv[1])
				}

				// set balance
				for tokenId, balance := range redoLog.BalanceMap {
					if tokenId == ledger.ViteTokenId {
						curCurrentData.Put(makeBalanceKey(addr.Bytes()), balance.Bytes())
					}
				}
			}

		}

	}
	return curCurrentData
}

func (cache *RoundCache) roundToLastSnapshotBlock(roundIndex uint64) (*ledger.SnapshotBlock, error) {
	if roundIndex <= 0 {
		return nil, nil
	}

	nextRoundIndex := roundIndex + 1

	startTime, _ := cache.timeIndex.Index2Time(nextRoundIndex)

	sb, err := cache.chain.GetSnapshotHeaderBeforeTime(&startTime)
	if err != nil {
		return nil, err
	}
	if sb == nil {
		panic(fmt.Sprintf("startTime is %s, sb is nil", startTime))
	}

	if sbRoundIndex := cache.timeIndex.Time2Index(*sb.Timestamp); sbRoundIndex == roundIndex {
		return sb, nil
	}

	return nil, nil
}

func (cache *RoundCache) getRoundSnapshotBlocks(roundIndex uint64) ([]*ledger.SnapshotBlock, error) {
	startTime, _ := cache.timeIndex.Index2Time(roundIndex)
	nextStartTime, _ := cache.timeIndex.Index2Time(roundIndex + 1)

	currentLastSb, err := cache.chain.GetSnapshotHeaderBeforeTime(&nextStartTime)
	if err != nil {
		return nil, err
	}

	if currentLastSb == nil {
		panic(fmt.Sprintf("currentLastSb is nil, nextStartTime is %s", nextStartTime))
	}

	nextFirstSbRoundIndex := cache.timeIndex.Time2Index(*currentLastSb.Timestamp)
	// means there are no snapshot blocks in the round of roundIndex.
	if nextFirstSbRoundIndex < roundIndex {
		return nil, nil
	}

	if nextFirstSbRoundIndex > roundIndex {
		panic(fmt.Sprintf("nextFirstSbRoundIndex should be smaller than roundIndex. nextFirstSbRoundIndex: %d, roundIndex: %d", nextFirstSbRoundIndex, roundIndex))
	}

	return cache.chain.GetSnapshotHeadersAfterOrEqualTime(&ledger.HashHeight{
		Hash:   currentLastSb.Hash,
		Height: currentLastSb.Height,
	}, &startTime, nil)
}

func (cache *RoundCache) setAllBalanceToCache(roundData *memdb.DB, snapshotHash types.Hash) error {
	batchSize := 10000
	addressList := make([]types.Address, 0, batchSize)

	var iterErr error
	cache.chain.IterateAccounts(func(addr types.Address, accountId uint64, err error) bool {

		addressList = append(addressList, addr)

		if len(addressList)%batchSize == 0 {
			if err := cache.setBalanceToCache(roundData, snapshotHash, addressList); err != nil {
				iterErr = err
				return false
			}

			addressList = make([]types.Address, 0, batchSize)
		}

		return true
	})

	if iterErr != nil {
		return iterErr
	}

	if len(addressList) > 0 {
		if err := cache.setBalanceToCache(roundData, snapshotHash, addressList); err != nil {
			return err
		}
	}

	return nil
}

func (cache *RoundCache) setBalanceToCache(roundData *memdb.DB, snapshotHash types.Hash, addressList []types.Address) error {
	balanceMap := make(map[types.Address]*big.Int, len(addressList))
	if err := cache.stateDB.GetSnapshotBalanceList(balanceMap, snapshotHash, addressList, ledger.ViteTokenId); err != nil {
		return err
	}

	for addr, balance := range balanceMap {
		// balance is zero
		if balance == nil || len(balance.Bytes()) <= 0 {
			continue
		}
		roundData.Put(makeBalanceKey(addr.Bytes()), balance.Bytes())
	}

	return nil
}

func (cache *RoundCache) setStorageToCache(roundData *memdb.DB, contractAddress types.Address, snapshotHash types.Hash) error {
	storageDatabase, err := cache.stateDB.NewStorageDatabase(snapshotHash, contractAddress)
	if err != nil {
		return err
	}
	iter, err := storageDatabase.NewStorageIterator(nil)
	if err != nil {
		return err
	}

	defer iter.Release()

	for iter.Next() {
		roundData.Put(makeStorageKey(iter.Key()), iter.Value())
	}

	if err := iter.Error(); err != nil {
		return err
	}

	return nil
}

func makeStorageKey(key []byte) []byte {
	return append([]byte{chain_utils.StorageKeyPrefix}, key...)
}

func makeBalanceKey(key []byte) []byte {
	return append([]byte{chain_utils.BalanceKeyPrefix}, key...)
}
