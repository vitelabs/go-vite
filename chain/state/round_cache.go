package chain_state

import (
	"fmt"
	chain_utils "github.com/vitelabs/go-vite/chain/utils"
	leveldb "github.com/vitelabs/go-vite/common/db/xleveldb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/comparer"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/db/xleveldb/memdb"
	"github.com/vitelabs/go-vite/common/db/xleveldb/util"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"sync"
)

type MemPool struct {
	byteSliceList  [][]byte
	byteSliceLimit int

	intSliceList  [][]int
	intSliceLimit int

	mu sync.RWMutex
}

func NewMemPool(byteSliceLimit int, intSliceLimit int) *MemPool {
	if byteSliceLimit <= 0 {
		byteSliceLimit = 3
	}
	if intSliceLimit <= 0 {
		intSliceLimit = 3
	}

	return &MemPool{
		byteSliceLimit: byteSliceLimit,
		intSliceLimit:  intSliceLimit,
	}
}

func (mp *MemPool) GetByteSlice(n int) []byte {
	if len(mp.byteSliceList) > 0 {
		mp.mu.RLock()
		byteSlice := mp.byteSliceList[0]
		mp.byteSliceList = mp.byteSliceList[1:]
		mp.mu.RUnlock()

		if len(byteSlice) < n {
			byteSlice = append(byteSlice, make([]byte, n-len(byteSlice))...)
		}
		return byteSlice[:n]
	}

	return make([]byte, n)
}

func (mp *MemPool) GetIntSlice(n int) []int {
	if len(mp.intSliceList) > 0 {
		mp.mu.RLock()
		intSlice := mp.intSliceList[0]
		mp.intSliceList = mp.intSliceList[1:]
		mp.mu.RUnlock()

		if len(intSlice) < n {
			intSlice = append(intSlice, make([]int, n-len(intSlice))...)
		}
		return intSlice[:n]
	}

	return make([]int, n)
}

func (mp *MemPool) Put(x interface{}) {
	switch x.(type) {
	case []byte:
		mp.mu.Lock()
		if len(mp.byteSliceList) < mp.byteSliceLimit {
			mp.byteSliceList = append(mp.byteSliceList, x.([]byte))
		} else {
			mp.byteSliceList[0] = x.([]byte)
		}
		mp.mu.Unlock()
	case []int:
		mp.mu.Lock()
		if len(mp.intSliceList) < mp.intSliceLimit {
			mp.intSliceList = append(mp.intSliceList, x.([]int))
		} else {
			mp.intSliceList[0] = x.([]int)
		}
		mp.mu.Unlock()
	default:
		panic(fmt.Sprintf("Unknown type: %t", x))
	}
}

type RoundCacheLogItem struct {
	Storage    [][2][]byte
	BalanceMap map[types.TokenTypeId]*big.Int
}

type RoundCacheSnapshotLog struct {
	LogMap map[types.Address][]RoundCacheLogItem

	SnapshotHeight uint64
}

func newRoundCacheSnapshotLog(capacity int, snapshotHeight uint64) *RoundCacheSnapshotLog {
	return &RoundCacheSnapshotLog{
		LogMap:         make(map[types.Address][]RoundCacheLogItem, capacity),
		SnapshotHeight: snapshotHeight,
	}
}

type RoundCacheRedoLogs struct {
	Logs []*RoundCacheSnapshotLog
}

func NewRoundCacheRedoLogs() *RoundCacheRedoLogs {
	return &RoundCacheRedoLogs{}
}

func (redoLogs *RoundCacheRedoLogs) Add(snapshotHeight uint64, snapshotLog SnapshotLog) {
	filteredSnapshotLog := newRoundCacheSnapshotLog(len(snapshotLog), snapshotHeight)

	for addr, logList := range snapshotLog {
		needSetStorage := addr == types.AddressGovernance

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

}

type RedoCacheData struct {
	roundIndex uint64

	lastSnapshotBlock *ledger.SnapshotBlock

	currentData *memdb.DB

	redoLogs *RoundCacheRedoLogs

	mu sync.RWMutex
}

func NewRedoCacheData(roundIndex uint64, lastSnapshotBlock *ledger.SnapshotBlock, currentData *memdb.DB, redoLogs *RoundCacheRedoLogs) *RedoCacheData {
	return &RedoCacheData{
		roundIndex: roundIndex,

		lastSnapshotBlock: lastSnapshotBlock,

		currentData: currentData,

		redoLogs: redoLogs,
	}
}

const (
	STOP   = 0
	INITED = 1
)

type RoundCache struct {
	chain            Chain
	stateDB          StateDBInterface
	roundCount       uint64
	timeIndex        core.TimeIndex
	latestRoundIndex uint64

	data   []*RedoCacheData
	mu     sync.RWMutex
	mp     *MemPool
	status int
}

func NewRoundCache(chain Chain, stateDB StateDBInterface, roundCount uint8) *RoundCache {
	return &RoundCache{
		chain:      chain,
		stateDB:    stateDB,
		roundCount: uint64(roundCount),
		data:       make([]*RedoCacheData, 0, roundCount),
		status:     STOP,
		mp:         NewMemPool(int(roundCount), int(roundCount)),
	}
}

// build data
func (cache *RoundCache) Init(timeIndex core.TimeIndex) (returnErr error) {
	if cache.status >= INITED {
		return nil
	}

	// set time index
	cache.timeIndex = timeIndex

	// stop chain write
	cache.chain.StopWrite()
	defer func() {
		// recover chain write
		cache.chain.RecoverWrite()

		if returnErr != nil {
			return
		}
		cache.status = INITED
	}()
	// get latest sb
	latestSb := cache.chain.GetLatestSnapshotBlock()

	// clean data
	cache.mu.Lock()
	cache.data = make([]*RedoCacheData, 0, cache.roundCount)
	cache.mu.Unlock()

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
		cache.mu.Lock()
		cache.data = roundsData
		cache.mu.Unlock()
	}

	return nil
}

// panic when return error
func (cache *RoundCache) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock, snapshotLog SnapshotLog) (returnErr error) {
	if cache.status < INITED {
		return nil
	}
	roundIndex := cache.timeIndex.Time2Index(*snapshotBlock.Timestamp)
	defer func() {
		if returnErr != nil {
			return
		}
		cache.latestRoundIndex = roundIndex
	}()

	// roundIndex can't be smaller than cache.latestRoundIndex
	if roundIndex < cache.latestRoundIndex {
		return errors.New(fmt.Sprintf("roundIndex < cache.latestRoundIndex, %d < %d", roundIndex, cache.latestRoundIndex))
	} else if roundIndex == cache.latestRoundIndex {
		// the roundIndex of snapshot block equals to cache.latestRoundIndex

		length := len(cache.data)
		// there are no redoCacheData when delete too more snapshot blocks not long ago， or there is no redo logs when init
		if length <= 0 {
			return nil
		}

		currentRedoCacheData := cache.data[length-1]

		currentRedoCacheData.redoLogs.Add(snapshotBlock.Height, snapshotLog)
		return nil
	}

	// there are no redoCacheData when delete too more snapshot blocks not long ago， or there is no redo logs when init
	dataLength := len(cache.data)

	if dataLength <= 0 {
		roundData, lastSnapshotBlock, err := cache.queryCurrentData(cache.latestRoundIndex)
		if err != nil {
			return err
		}

		if lastSnapshotBlock == nil {
			return errors.New(fmt.Sprintf("lastSnapshotBlock is nil, cache.latestRoundIndex is %d", cache.latestRoundIndex))
		}

		if roundData == nil {
			return errors.New(fmt.Sprintf("roundData is nil, cache.latestRoundIndex is %d", cache.latestRoundIndex))
		}

		cache.mu.Lock()
		cache.data = append(cache.data, NewRedoCacheData(cache.latestRoundIndex, lastSnapshotBlock, roundData, nil))
		cache.mu.Unlock()

	} else if dataLength == 1 {
		return errors.New(fmt.Sprintf("len(cache.data) is 1, cache.data[0] is %+v", cache.data[0]))
	} else {
		prevRoundData := cache.data[dataLength-1]
		if prevRoundData.redoLogs == nil || len(prevRoundData.redoLogs.Logs) <= 0 {
			return errors.New(fmt.Sprintf("prevRoundData.redoLogs == nil || len(prevRoundData.redoLogs.Logs) <= 0. prevRoundData is %+v", prevRoundData))
		}

		prevBeforePrevRoundData := cache.data[dataLength-2]
		if prevBeforePrevRoundData.currentData == nil {
			return errors.New(fmt.Sprintf("prevBeforePrevRoundData.currentData == nil, prevBeforePrevRoundData is %+v", prevBeforePrevRoundData))
		}

		// get lastSnapshotBlock
		lastSnapshotBlock, err := cache.chain.GetSnapshotHeaderByHeight(snapshotBlock.Height - 1)
		if err != nil {
			return err
		}
		if lastSnapshotBlock == nil {
			return errors.New(fmt.Sprintf("lastSnapshotBlock is nil, height is %d", snapshotBlock.Height-1))
		}

		prevRoundData.mu.Lock()
		// build prevRoundData.currentData
		prevRoundData.currentData = cache.buildCurrentData(prevBeforePrevRoundData.currentData, prevRoundData.redoLogs)

		// set prevRoundData.lastSnapshotBlock
		prevRoundData.lastSnapshotBlock = lastSnapshotBlock

		//fmt.Printf("buildCurrentData %d, %s\n", lastSnapshotBlock.Height, lastSnapshotBlock.Hash)

		prevRoundData.mu.Unlock()

	}

	redoLogs := NewRoundCacheRedoLogs()
	redoLogs.Add(snapshotBlock.Height, snapshotLog)

	cache.mu.Lock()
	cache.data = append(cache.data, NewRedoCacheData(roundIndex, nil, nil, redoLogs))

	if uint64(len(cache.data)) > cache.roundCount {
		// destroy memdb
		cache.destroyMemDb(cache.data[0].currentData)

		// remove head
		cache.data = cache.data[1:]
	}
	cache.mu.Unlock()

	return nil
}

// panic when return error
func (cache *RoundCache) DeleteSnapshotBlocks(snapshotBlocks []*ledger.SnapshotBlock) error {
	if cache.status < INITED {
		return nil
	}

	if len(snapshotBlocks) <= 0 {
		return errors.New("Length of snapshotBlocks to deleting is 0")
	}

	firstSnapshotBlock := snapshotBlocks[0]
	firstRoundIndex := cache.timeIndex.Time2Index(*firstSnapshotBlock.Timestamp)

	end := len(cache.data)

	for i := end - 1; i >= 0; i-- {
		data := cache.data[i]
		if data.roundIndex > firstRoundIndex {
			// to be deleted
			end = i
		} else if data.roundIndex == firstRoundIndex {
			// remove currentData
			data.mu.Lock()
			data.currentData = nil

			// remove lastSnapshotBlock
			data.lastSnapshotBlock = nil
			data.mu.Unlock()

			if data.redoLogs == nil || len(data.redoLogs.Logs) <= 0 {
				end = i
			} else {
				// remove
				redoLogsEnd := len(data.redoLogs.Logs)
				for j := redoLogsEnd - 1; j >= 0; j-- {
					redoLog := data.redoLogs.Logs[j]
					if redoLog.SnapshotHeight >= firstSnapshotBlock.Height {
						redoLogsEnd = j
					} else {
						break
					}
				}

				if redoLogsEnd <= 0 {
					end = i
				} else {
					data.redoLogs.Logs = data.redoLogs.Logs[:redoLogsEnd]
				}

			}
		} else if data.roundIndex < firstRoundIndex {
			break
		}
	}

	// set cache.data
	cache.mu.Lock()
	deleteTo := end
	if end <= 1 {
		deleteTo = 0
	}
	for _, item := range cache.data[deleteTo:] {
		cache.destroyMemDb(item.currentData)
	}
	cache.data = cache.data[:deleteTo]
	cache.mu.Unlock()

	// delete currentData and lastSnapshotBlock of last round data
	if len(cache.data) > 0 {
		lastRoundCache := cache.data[len(cache.data)-1]

		lastRoundCache.mu.Lock()
		cache.destroyMemDb(lastRoundCache.currentData)

		lastRoundCache.lastSnapshotBlock = nil
		lastRoundCache.currentData = nil
		lastRoundCache.mu.Unlock()

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

// tokenId is viteTokenID
func (cache *RoundCache) GetSnapshotViteBalanceList(snapshotHash types.Hash, addrList []types.Address) (map[types.Address]*big.Int, []types.Address, error) {
	if cache.status < INITED {
		return nil, nil, nil
	}

	currentData := cache.getCurrentData(snapshotHash)

	if currentData == nil {
		return nil, nil, nil
	}
	//fmt.Printf("GetSnapshotViteBalanceList %s\n", snapshotHash)

	balanceMap := make(map[types.Address]*big.Int)
	var notFoundAddressList []types.Address

	for _, addr := range addrList {
		// query balance
		value, err := currentData.Get(makeBalanceKey(addr.Bytes()))
		if err == leveldb.ErrNotFound {
			notFoundAddressList = append(notFoundAddressList, addr)
		} else {
			balanceMap[addr] = big.NewInt(0).SetBytes(value)
		}
	}

	return balanceMap, notFoundAddressList, nil
}

func (cache *RoundCache) StorageIterator(snapshotHash types.Hash) interfaces.StorageIterator {
	if cache.status < INITED {
		return nil
	}

	currentData := cache.getCurrentData(snapshotHash)

	if currentData == nil {
		return nil
	}
	//fmt.Printf("StorageIterator %s\n", snapshotHash)

	return NewTransformIterator(currentData.NewIterator(util.BytesPrefix(makeStorageKey(nil))), 1)
}

func (cache *RoundCache) getCurrentData(snapshotHash types.Hash) *memdb.DB {
	if len(cache.data) <= 0 {
		return nil
	}

	cache.mu.RLock()

	dataCopied := make([]*RedoCacheData, len(cache.data))
	copy(dataCopied, cache.data)

	cache.mu.RUnlock()

	var currentData *memdb.DB
	for _, dataItem := range dataCopied {
		dataItem.mu.RLock()
		lastSnapshotBlock := dataItem.lastSnapshotBlock
		tmpCurrentData := dataItem.currentData
		dataItem.mu.RUnlock()

		if lastSnapshotBlock == nil {
			return nil
		}

		if lastSnapshotBlock.Hash == snapshotHash {
			currentData = tmpCurrentData
			break
		}

	}

	return currentData

}

// 9、11、12
// startRoundIndex 9, end 12
// 9、11、12
// 9、11、12
// 8、12

func (cache *RoundCache) initRounds(startRoundIndex, endRoundIndex uint64) ([]*RedoCacheData, error) {
	if startRoundIndex >= endRoundIndex {
		return nil, nil
	}

	// [startRoundIndex - roundIndex]
	// init first currentData
	firstCurrentData, lastSnapshotBlock, err := cache.queryCurrentData(startRoundIndex)
	if err != nil {
		return nil, err
	}
	if firstCurrentData == nil {
		return nil, nil
	}

	// init first redo logs
	firstRedoLogs, _, err := cache.queryRedoLogs(startRoundIndex)
	if err != nil {
		return nil, err
	}

	roundsData := make([]*RedoCacheData, 0, cache.roundCount)

	roundsData = append(roundsData, NewRedoCacheData(startRoundIndex, lastSnapshotBlock, firstCurrentData, firstRedoLogs))

	prevCurrentData := firstCurrentData

	for index := startRoundIndex + 1; index <= endRoundIndex; index++ {

		redoLogs, hasStoreRedoLogs, err := cache.queryRedoLogs(index)
		if err != nil {
			return nil, err
		}

		if !hasStoreRedoLogs {
			return cache.initRounds(index, endRoundIndex)
		}

		if redoLogs == nil {
			continue
		}

		var curCurrentData *memdb.DB
		var lastSnapshotBlock *ledger.SnapshotBlock

		if index < endRoundIndex {
			var err error
			lastSnapshotBlock, err = cache.roundToLastSnapshotBlock(index)
			if err != nil {
				return nil, err
			}
			if lastSnapshotBlock == nil {
				return nil, errors.New(fmt.Sprintf("lastSnapshotBlock is nil, index is %d", index))
			}

			curCurrentData = cache.buildCurrentData(prevCurrentData, redoLogs)

			prevCurrentData = curCurrentData
		}

		roundsData = append(roundsData, NewRedoCacheData(index, lastSnapshotBlock, curCurrentData, redoLogs))
	}

	return roundsData, nil
}

func (cache *RoundCache) queryCurrentData(roundIndex uint64) (*memdb.DB, *ledger.SnapshotBlock, error) {
	roundData := memdb.New(comparer.DefaultComparer, 0)

	// get the last snapshot block of round
	snapshotBlock, err := cache.roundToLastSnapshotBlock(roundIndex)
	if err != nil {
		return nil, nil, err
	}
	if snapshotBlock == nil {
		return nil, nil, nil
	}

	// set vote list cache
	if err := cache.setStorageToCache(roundData, types.AddressGovernance, snapshotBlock.Hash); err != nil {
		return nil, nil, err
	}

	// set address balances cache
	if err := cache.setAllBalanceToCache(roundData, snapshotBlock.Hash); err != nil {
		return nil, nil, err
	}

	return roundData, snapshotBlock, nil
}

func (cache *RoundCache) queryRedoLogs(roundIndex uint64) (returnRedoLogs *RoundCacheRedoLogs, isStoreRedoLogs bool, returnErr error) {

	// query the snapshot blocks of the round
	snapshotBlocks, err := cache.getRoundSnapshotBlocks(roundIndex)
	if err != nil {
		return nil, false, err
	}

	if len(snapshotBlocks) <= 0 {
		return nil, true, nil
	}

	roundRedoLogs := NewRoundCacheRedoLogs()

	// assume snapshot blocks is sorted by height asc, query redoLog by snapshot block height
	for _, snapshotBlock := range snapshotBlocks {
		redoLog, hasRedoLog, err := cache.stateDB.Redo().QueryLog(snapshotBlock.Height)
		if err != nil {
			return nil, false, err
		}
		if !hasRedoLog {
			return nil, false, nil
		}

		roundRedoLogs.Add(snapshotBlock.Height, redoLog)
	}

	return roundRedoLogs, true, nil
}

// TODO optimize: build in parallel
func (cache *RoundCache) buildCurrentData(prevCurrentData *memdb.DB, redoLogs *RoundCacheRedoLogs) *memdb.DB {
	// copy
	curCurrentData := prevCurrentData.Copy2(cache.mp.GetByteSlice, cache.mp.GetIntSlice)

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
		// balance is nil
		if balance == nil {
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

func (cache *RoundCache) destroyMemDb(db *memdb.DB) {
	if db == nil {
		return
	}
	db.Destroy2(cache.mp.Put)
}

func makeStorageKey(key []byte) []byte {
	return append([]byte{chain_utils.StorageKeyPrefix}, key...)
}

func makeBalanceKey(key []byte) []byte {
	return append([]byte{chain_utils.BalanceKeyPrefix}, key...)
}
