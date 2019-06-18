package chain_state

import (
	"fmt"
	chain_utils "github.com/vitelabs/go-vite/chain/utils"
	"github.com/vitelabs/go-vite/common/db"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

// TODO
type RoundCacheRedoLogs struct{}

// TODO
func NewRoundCacheRedoLogs() *RoundCacheRedoLogs {
	return nil
}

// TODO filter
func (redoLogs *RoundCacheRedoLogs) Add(log SnapshotLog) {}

type RedoCacheData struct {
	CurrentData *db.MemDB

	RedoLogs *RoundCacheRedoLogs
}

func NewRedoCacheData(currentData *db.MemDB, redoLogs *RoundCacheRedoLogs) *RedoCacheData {
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

		if redoLogs == nil {
			// TODO no redo log
			panic("implementation me")
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

// TODO
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

	// 2. 根据snapshot block查出所有redoLog，转换为RoundCacheRedoLogs
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

// TODO
func (cache *RoundCache) buildCurrentData(prevBaseData *db.MemDB, redoLogs *RoundCacheRedoLogs) *db.MemDB {
	return nil
}

func (cache *RoundCache) roundToSnapshotBlock(roundIndex uint64) (*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (cache *RoundCache) getRoundSnapshotBlocks(roundIndex uint64) ([]*ledger.SnapshotBlock, error) {
	return nil, nil
}

func (cache *RoundCache) setAllBalanceToCache(roundData *db.MemDB, snapshotHash types.Hash) error {
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

func (cache *RoundCache) setBalanceToCache(roundData *db.MemDB, snapshotHash types.Hash, addressList []types.Address) error {
	balanceMap := make(map[types.Address]*big.Int, len(addressList))
	if err := cache.stateDB.GetSnapshotBalanceList(balanceMap, snapshotHash, addressList, ledger.ViteTokenId); err != nil {
		return err
	}

	for addr, balance := range balanceMap {
		// balance is zero
		if balance == nil || len(balance.Bytes()) <= 0 {
			continue
		}
		roundData.Put(append([]byte{chain_utils.BalanceHistoryKeyPrefix}, addr.Bytes()...), balance.Bytes())
	}

	return nil
}

func (cache *RoundCache) setStorageToCache(roundData *db.MemDB, contractAddress types.Address, snapshotHash types.Hash) error {
	storageDatabase, err := cache.stateDB.NewStorageDatabase(snapshotHash, contractAddress)
	if err != nil {
		return err
	}
	iter := storageDatabase.NewRawStorageIterator(nil)
	defer iter.Release()

	for iter.Next() {
		roundData.Put(iter.Key(), iter.Value())
	}

	if err := iter.Error(); err != nil {
		return err
	}

	return nil
}
