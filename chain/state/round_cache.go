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

// TODO
type RoundCacheRedoLogs struct {
	Logs []SnapshotLog
}

// TODO
func NewRoundCacheRedoLogs() *RoundCacheRedoLogs {
	return nil
}

// TODO filter
func (redoLogs *RoundCacheRedoLogs) Add(log SnapshotLog) {}

type RedoCacheData struct {
	CurrentData *memdb.DB

	RedoLogs *RoundCacheRedoLogs
}

func NewRedoCacheData(currentData *memdb.DB, redoLogs *RoundCacheRedoLogs) *RedoCacheData {
	return &RedoCacheData{
		CurrentData: currentData,
		RedoLogs:    redoLogs,
	}
}

type RoundCache struct {
	chain      Chain
	stateDB    *StateDB
	roundCount uint64
	timeIndex  core.TimeIndex // TODO new timeIndex: GetPeriodTimeIndex() TimeIndex

	data map[uint64]*RedoCacheData
}

func NewRoundCache(chain Chain, stateDB *StateDB, roundCount uint8) *RoundCache {
	return &RoundCache{
		chain:      chain,
		stateDB:    stateDB,
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

	roundsData, err := cache.initRounds(startRoundIndex, roundIndex)
	if err != nil {
		return err
	}

	if roundsData != nil {
		cache.data = roundsData
	}

	return nil
}

func (cache *RoundCache) initRounds(startRoundIndex, endRoundIndex uint64) (map[uint64]*RedoCacheData, error) {
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

	roundsData := make(map[uint64]*RedoCacheData)

	roundsData[startRoundIndex] = NewRedoCacheData(firstCurrentData, firstRedoLogs)

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

		roundsData[startRoundIndex] = NewRedoCacheData(curCurrentData, redoLogs)
	}

	return roundsData, nil
}

func (cache *RoundCache) queryCurrentData(roundIndex uint64) (*memdb.DB, error) {
	roundData := memdb.New(leveldb.NewIComparer(comparer.DefaultComparer), 0)

	// get the last snapshot block of round
	snapshotBlock, err := cache.roundToSnapshotBlock(roundIndex)
	if err != nil {
		return nil, err
	}
	if snapshotBlock == nil {
		return nil, errors.New(fmt.Sprintf("snapshot block is nil, roundIndex is %d", roundIndex))
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

		roundRedoLogs.Add(redoLog)
	}

	return roundRedoLogs, nil
}

// TODO optimize: build in parallel
func (cache *RoundCache) buildCurrentData(prevCurrentData *memdb.DB, redoLogs *RoundCacheRedoLogs) *memdb.DB {
	// copy
	curCurrentData := prevCurrentData.Copy()

	for _, snapshotLog := range redoLogs.Logs {
		for _, redoLogs := range snapshotLog {
			for _, redoLog := range redoLogs {
				// set storage
				for _, kv := range redoLog.Storage {
					curCurrentData.Put(makeStorageKey(kv[0]), kv[1])
				}

				// TODO set balance

			}

		}

	}
	return nil
}

func (cache *RoundCache) roundToSnapshotBlock(roundIndex uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (cache *RoundCache) getRoundSnapshotBlocks(roundIndex uint64) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
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
