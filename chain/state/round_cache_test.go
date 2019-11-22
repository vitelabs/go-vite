package chain_state

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/golang/mock/gomock"
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
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"math/big"
	"math/rand"
	"testing"
	"time"
)

type MockSnapshot struct {
	SnapshotHeader *ledger.SnapshotBlock
	Data           *memdb.DB
	Log            SnapshotLog
}

type MockData struct {
	AddrList  []types.Address
	Snapshots []MockSnapshot
}

func NewMockData(addrList []types.Address, snapshotCount int) *MockData {
	data := &MockData{}
	data.Add(addrList, snapshotCount)

	return data
}

func (mockData *MockData) Add(appendAddrList []types.Address, snapshotCount int) *MockData {
	newMockData := &MockData{
		AddrList: appendAddrList,
	}

	mockData.AddrList = append(mockData.AddrList, appendAddrList...)

	startH := uint64(1)
	snapshotData := memdb.New(comparer.DefaultComparer, 0)

	if len(mockData.Snapshots) > 0 {
		lastSnapshot := mockData.Snapshots[len(mockData.Snapshots)-1]

		startH = lastSnapshot.SnapshotHeader.Height + 1
		snapshotData = lastSnapshot.Data

	}

	for h := startH; h < uint64(snapshotCount)+startH; h++ {
		currentTime := genesisTime.Add(time.Duration(h-1) * time.Second)

		// snapshot log
		newSnapshotData := snapshotData.Copy()

		log := mockSnapshotState(newSnapshotData, appendAddrList, rand.Intn(5)+1, rand.Intn(5)+1)

		snapshotHeader := &ledger.SnapshotBlock{
			Height:    h,
			Timestamp: &currentTime,
		}
		snapshotHeader.Hash = snapshotHeader.ComputeHash()

		newMockSnapshot := MockSnapshot{
			SnapshotHeader: snapshotHeader,
			Data:           newSnapshotData,
			Log:            log,
		}

		mockData.Snapshots = append(mockData.Snapshots, newMockSnapshot)
		newMockData.Snapshots = append(newMockData.Snapshots, newMockSnapshot)

		snapshotData = newSnapshotData
	}
	return newMockData
}

func (mockData *MockData) Delete(snapshotCount int) *MockData {

	lowIndex := 1
	if len(mockData.Snapshots) > snapshotCount {
		lowIndex = len(mockData.Snapshots) - snapshotCount
	}

	deletedMockData := &MockData{
		AddrList:  mockData.AddrList,
		Snapshots: mockData.Snapshots[lowIndex:],
	}
	mockData.Snapshots = mockData.Snapshots[:lowIndex]
	return deletedMockData
}

func mockSnapshotState(data *memdb.DB, addrList []types.Address, accountCount, keyLength int) SnapshotLog {
	log := make(SnapshotLog)

	for i := 0; i < accountCount; i++ {
		randNum := rand.Intn(100)
		var addr types.Address
		if randNum < 30 {
			addr = types.AddressGovernance
		} else {
			addr = addrList[rand.Intn(len(addrList))]
		}

		// storage
		//storageMap := make(map[string][]byte)
		var storage [][2][]byte

		for j := 0; j < keyLength; j++ {

			key := createRandomBytes()
			value := createRandomBytes()

			//storageMap[string(key)] = value
			data.Put(mockStorageKey(addr, key), value)
			storage = append(storage, [2][]byte{key, value})
		}

		//contractData[addr] = storageMap

		// balance
		balanceMap := map[types.TokenTypeId]*big.Int{ledger.ViteTokenId: big.NewInt(rand.Int63() % 100000)}
		for tokenId, balance := range balanceMap {
			data.Put(mockBalanceKey(addr, tokenId), balance.Bytes())
		}

		log[addr] = append(log[addr], LogItem{
			Storage:    storage,
			BalanceMap: balanceMap,
		})
	}
	return log
}

var genesisTime = time.Unix(1563182961, 0)

func createAddrList(count int) []types.Address {
	addrList := make([]types.Address, 0, count)
	for i := 0; i < count; i++ {
		addr, _, err := types.CreateAddress()
		if err != nil {
			panic(err)
		}
		addrList = append(addrList, addr)
	}

	return addrList
}

func createRandomBytes() []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, rand.Uint64())
	return bytes
}

func mockStorageKey(addr types.Address, key []byte) []byte {
	return append(append([]byte{chain_utils.StorageKeyPrefix}, addr.Bytes()...), key...)

}

func mockBalanceKey(addr types.Address, tokenId types.TokenTypeId) []byte {
	return append(append([]byte{chain_utils.BalanceKeyPrefix}, addr.Bytes()...), tokenId.Bytes()...)
}

func getMockChain(ctrl *gomock.Controller, mockData *MockData) *MockChain {
	// mock chain
	mockChain := NewMockChain(ctrl)

	// mock chain.StopWrite
	mockChain.EXPECT().StopWrite().Times(1)

	// mock chain.RecoverWrite
	mockChain.EXPECT().RecoverWrite().Times(1)

	// mock chain.GetLatestSnapshotBlock
	mockChain.EXPECT().GetLatestSnapshotBlock().DoAndReturn(func() *ledger.SnapshotBlock {
		return mockData.Snapshots[len(mockData.Snapshots)-1].SnapshotHeader
	}).AnyTimes()

	// mock chain.GetSnapshotHeaderBeforeTime
	mockChain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Any()).DoAndReturn(func(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
		for i := len(mockData.Snapshots) - 1; i >= 0; i-- {
			dataItem := mockData.Snapshots[i]
			sbHeader := dataItem.SnapshotHeader
			if sbHeader.Timestamp.Before(*timestamp) {
				return sbHeader, nil
			}
		}

		return nil, nil
	}).AnyTimes()

	// mock GetSnapshotHeadersAfterOrEqualTime
	mockChain.EXPECT().GetSnapshotHeadersAfterOrEqualTime(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(endHashHeight *ledger.HashHeight, startTime *time.Time, producer *types.Address) ([]*ledger.SnapshotBlock, error) {
			var snapshotHeaders []*ledger.SnapshotBlock
			for _, dataItem := range mockData.Snapshots {
				snapshotHeader := dataItem.SnapshotHeader
				if snapshotHeader.Height > endHashHeight.Height {
					break
				}

				if snapshotHeader.Timestamp.After(*startTime) ||
					snapshotHeader.Timestamp.Equal(*startTime) {

					snapshotHeaders = append(snapshotHeaders, dataItem.SnapshotHeader)
				}
			}

			if producer != nil && len(snapshotHeaders) > 0 {
				var result = make([]*ledger.SnapshotBlock, 0, len(snapshotHeaders)/75+3)
				for _, snapshotHeader := range snapshotHeaders {
					if snapshotHeader.Producer() == *producer {
						result = append(result)
					}
				}
				return result, nil
			}

			return snapshotHeaders, nil
		}).AnyTimes()

	// GetSnapshotHeaderByHeight
	mockChain.EXPECT().GetSnapshotHeaderByHeight(gomock.Any()).DoAndReturn(func(height uint64) (*ledger.SnapshotBlock, error) {

		for _, snapshot := range mockData.Snapshots {
			snapshotHeader := snapshot.SnapshotHeader
			if snapshotHeader.Height == height {
				return snapshotHeader, nil
			}
		}

		return nil, nil
	}).AnyTimes()

	// iterate accounts
	mockChain.EXPECT().IterateAccounts(gomock.Any()).DoAndReturn(func(iterFunc func(addr types.Address, accountId uint64, err error) bool) {
		for i := 0; i < len(mockData.AddrList); i++ {
			addr := mockData.AddrList[i]
			if !iterFunc(addr, uint64(i+1), nil) {
				break
			}

		}
	}).AnyTimes()

	return mockChain
}

func findSnapshotData(mockData []MockSnapshot, snapshotHash types.Hash) *MockSnapshot {
	for _, dataItem := range mockData {
		if dataItem.SnapshotHeader.Hash == snapshotHash {
			return &dataItem
		}
	}

	return nil
}
func findSnapshotDataByHeight(mockData []MockSnapshot, snapshotHeight uint64) *MockSnapshot {
	for _, dataItem := range mockData {
		if dataItem.SnapshotHeader.Height == snapshotHeight {
			return &dataItem
		}
	}
	return nil
}

func getMockStateDB(ctrl *gomock.Controller, mockData *MockData) *MockStateDBInterface {
	// mock state db
	mockStateDb := NewMockStateDBInterface(ctrl)

	// mock stateDb.NewStorageDatabase
	mockStateDb.EXPECT().NewStorageDatabase(gomock.Any(), gomock.Any()).
		DoAndReturn(func(snapshotHash types.Hash, addr types.Address) (StorageDatabaseInterface, error) {

			snapshotData := findSnapshotData(mockData.Snapshots, snapshotHash)

			if snapshotData == nil {
				return nil, errors.New("snapshotData is nil")
			}

			// mock storage database
			storageDatabase := NewMockStorageDatabaseInterface(ctrl)

			// mock address
			storageDatabase.EXPECT().Address().Return(&addr).AnyTimes()

			// mock get value
			storageDatabase.EXPECT().GetValue(gomock.Any()).DoAndReturn(func(key []byte) ([]byte, error) {
				return snapshotData.Data.Get(mockStorageKey(addr, key))
			}).AnyTimes()

			// mock new storage iterator
			storageDatabase.EXPECT().NewStorageIterator(gomock.Any()).DoAndReturn(func(prefix []byte) (interfaces.StorageIterator, error) {
				return NewTransformIterator(snapshotData.Data.NewIterator(util.BytesPrefix(mockStorageKey(addr, prefix))), 1+types.AddressSize), nil
			})

			// *StorageDatabase, error
			//NewStorageDatabase(snapshotHash, addr)(*StorageDatabase, error)

			return storageDatabase, nil
		}).AnyTimes()

	// mock GetSnapshotBalanceList
	mockStateDb.EXPECT().GetSnapshotBalanceList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Eq(ledger.ViteTokenId)).
		DoAndReturn(func(balanceMap map[types.Address]*big.Int, snapshotBlockHash types.Hash, addrList []types.Address, tokenId types.TokenTypeId) error {
			snapshotData := findSnapshotData(mockData.Snapshots, snapshotBlockHash)

			if snapshotData == nil {
				return errors.New("snapshotData is nil")
			}

			for _, addr := range addrList {
				value, err := snapshotData.Data.Get(mockBalanceKey(addr, tokenId))
				if err == leveldb.ErrNotFound {
					continue
				}

				balanceMap[addr] = big.NewInt(0).SetBytes(value)
			}

			return nil
		}).AnyTimes()

	// mock Redo
	mockStateDb.EXPECT().Redo().DoAndReturn(func() RedoInterface {
		mockRedo := NewMockRedoInterface(ctrl)
		mockRedo.EXPECT().QueryLog(gomock.Any()).DoAndReturn(func(snapshotHeight uint64) (SnapshotLog, bool, error) {
			snapshotData := findSnapshotDataByHeight(mockData.Snapshots, snapshotHeight)
			if snapshotData == nil {
				return nil, false, errors.New("snapshotData is nil")
			}

			if snapshotData.Log == nil {
				return nil, false, nil
			}
			return snapshotData.Log, true, nil
		})

		return mockRedo
	}).AnyTimes()
	return mockStateDb
}

type MockRoundSnapshots struct {
	Snapshots    []*MockSnapshot
	LastSnapshot *MockSnapshot
}

func getRoundSnapshotData(mockData []MockSnapshot, roundIndex uint64, index TimeIndex) *MockRoundSnapshots {

	var mockRoundSnapshots *MockRoundSnapshots
	for i := len(mockData) - 1; i >= 0; i-- {
		dataItem := mockData[i]
		currentRoundIndex := index.Time2Index(*dataItem.SnapshotHeader.Timestamp)
		if currentRoundIndex < roundIndex {
			break
		} else if currentRoundIndex == roundIndex {
			if mockRoundSnapshots == nil {
				mockRoundSnapshots = &MockRoundSnapshots{}
			}

			mockRoundSnapshots.Snapshots = append([]*MockSnapshot{
				&dataItem,
			}, mockRoundSnapshots.Snapshots...)
		}
	}

	latestRoundIndex := index.Time2Index(*mockData[len(mockData)-1].SnapshotHeader.Timestamp)
	if mockRoundSnapshots != nil && roundIndex < latestRoundIndex {
		mockRoundSnapshots.LastSnapshot = mockRoundSnapshots.Snapshots[len(mockRoundSnapshots.Snapshots)-1]
	}
	return mockRoundSnapshots
}

func checkRedoLogs(t *testing.T, redoLogs *RoundCacheRedoLogs,
	snapshots []*MockSnapshot) {

	assert.Equal(t, len(redoLogs.Logs), len(snapshots))

	for index, log := range redoLogs.Logs {
		snapshot := snapshots[index]
		mockLog := snapshot.Log
		assert.Equal(t, len(log.LogMap), len(mockLog))

		for addr, logItems := range log.LogMap {
			mockLogItems, ok := mockLog[addr]

			assert.Equal(t, ok, true)
			assert.Equal(t, len(logItems), len(mockLogItems))

			for logIndex, logItem := range logItems {
				mockLogItem := mockLogItems[logIndex]
				// check storage
				if addr == types.AddressGovernance {
					assert.Equal(t, len(logItem.Storage), len(mockLogItem.Storage),
						fmt.Sprintf("height: %d, address: %s", snapshot.SnapshotHeader.Height, addr))
					for i, kv := range logItem.Storage {
						mockKv := mockLogItem.Storage[i]

						assert.Check(t, bytes.Equal(kv[0], mockKv[0]))
						assert.Check(t, bytes.Equal(kv[1], mockKv[1]))
					}
				}

				assert.Check(t, logItem.BalanceMap[ledger.ViteTokenId].Cmp(mockLogItem.BalanceMap[ledger.ViteTokenId]) == 0)
			}
		}
	}

}

func compareIter(t *testing.T, iter1 interfaces.StorageIterator, iter2 interfaces.StorageIterator, fastCheck bool) {
	var list1 [][2][]byte
	for iter1.Next() {
		key := make([]byte, len(iter1.Key()))
		copy(key[:], iter1.Key())

		value := make([]byte, len(iter1.Value()))
		copy(value[:], iter1.Value())

		list1 = append(list1, [2][]byte{key, value})
	}

	var list2 [][2][]byte
	for iter2.Next() {
		key := make([]byte, len(iter2.Key()))
		copy(key[:], iter2.Key())

		value := make([]byte, len(iter2.Value()))
		copy(value[:], iter2.Value())

		list2 = append(list2, [2][]byte{key, value})
	}

	assert.Equal(t, len(list1), len(list2))

	end := len(list1)
	if fastCheck {
		end = 10
	}
	for i := 0; i < end; i++ {
		index := i

		if fastCheck {
			index = rand.Intn(len(list1))
		}

		kv1 := list1[index]
		kv2 := list2[index]
		if !assert.Check(t, bytes.Equal(kv1[0], kv2[0]), fmt.Sprintf("%d. %d != %d. %v", index, kv1[0], kv2[0], fastCheck)) {
			t.FailNow()
		}
		if !assert.Check(t, bytes.Equal(kv1[1], kv2[1]), fmt.Sprintf("%d. %d != %d. %v", index, kv1[1], kv2[1], fastCheck)) {
			t.FailNow()
		}
	}

}

func checkStorage(t *testing.T, redoCacheData *memdb.DB,
	mockData *memdb.DB) {

	iter1 := NewTransformIterator(redoCacheData.NewIterator(util.BytesPrefix(makeStorageKey(nil))), 1)
	defer iter1.Release()

	iter2 := NewTransformIterator(mockData.NewIterator(util.BytesPrefix(mockStorageKey(types.AddressGovernance, nil))), 1+types.AddressSize)
	defer iter2.Release()

	compareIter(t, iter1, iter2, false)
}

func checkPrevRoundIndex(t *testing.T, mockData []MockSnapshot, prevRoundIndex uint64, roundIndex uint64, timeIndex TimeIndex) {
	var highIndex *uint64
	for i := len(mockData) - 1; i >= 0; i-- {
		dataItem := mockData[i]
		currentIndex := timeIndex.Time2Index(*dataItem.SnapshotHeader.Timestamp)
		if highIndex != nil && *highIndex != currentIndex {
			if *highIndex == roundIndex && currentIndex == prevRoundIndex {
				return
			}
		}
		highIndex = &currentIndex
	}
	t.Fatal(fmt.Sprintf("prevRoundIndex is %d, roundIndex is %d", prevRoundIndex, roundIndex))
	t.FailNow()
}

func checkRoundCache(t *testing.T, mockData *MockData,
	roundCache *RoundCache, chain Chain, timeIndex TimeIndex) {
	// check latestRoundIndex
	latestSnapshotBlock := chain.GetLatestSnapshotBlock()
	latestIndex := timeIndex.Time2Index(*latestSnapshotBlock.Timestamp)

	assert.Equal(t, latestIndex, roundCache.latestRoundIndex)

	// check data
	var prevRoundIndex *uint64
	fmt.Printf("Check roundCache, length: %d\n", len(roundCache.data))
	for index, dataItem := range roundCache.data {
		roundSnapshotData := getRoundSnapshotData(mockData.Snapshots, dataItem.roundIndex, timeIndex)
		if roundSnapshotData == nil {
			t.Fatal(fmt.Sprintf("roundSnapshotData is nil, roundIndex is %d", dataItem.roundIndex))
			t.FailNow()
		}

		// check round index
		snapshotRoundIndex := timeIndex.Time2Index(*roundSnapshotData.Snapshots[0].SnapshotHeader.Timestamp)
		assert.Equal(t, snapshotRoundIndex, dataItem.roundIndex)

		if prevRoundIndex != nil {
			checkPrevRoundIndex(t, mockData.Snapshots, *prevRoundIndex, snapshotRoundIndex, timeIndex)
		}

		prevRoundIndex = &snapshotRoundIndex

		// check redo logs
		if dataItem.redoLogs != nil {
			fmt.Printf("Check %d round redo logs\n", snapshotRoundIndex)
			checkRedoLogs(t, dataItem.redoLogs, roundSnapshotData.Snapshots)

			if dataItem.lastSnapshotBlock == nil {
				assert.Check(t, dataItem.currentData == nil)
				assert.Check(t, roundSnapshotData.LastSnapshot == nil)
				continue
			}
		} else if index != 0 {
			t.Fatal(fmt.Sprintf("dataItem.redoLogs is nil, index is %d", index))
			t.FailNow()
		}

		// check roundSnapshotData.LastSnapshot
		assert.Check(t, roundSnapshotData.LastSnapshot != nil)

		lastSnapshotHeader := roundSnapshotData.LastSnapshot.SnapshotHeader

		// check last snapshot block
		assert.Equal(t, dataItem.lastSnapshotBlock.Hash, lastSnapshotHeader.Hash)
		assert.Equal(t, dataItem.lastSnapshotBlock.Height, lastSnapshotHeader.Height)

		// check storage
		fmt.Printf("Check %d round storage\n", snapshotRoundIndex)
		checkStorage(t, dataItem.currentData, roundSnapshotData.LastSnapshot.Data)

	}
}

func checkRoundCacheQuery(t *testing.T, roundCache *RoundCache, mockData *MockData, timeIndex TimeIndex) {
	fmt.Printf("Check roundCache query, length: %d\n", len(roundCache.data))

	for index, snapshot := range mockData.Snapshots {

		// compute hasCache
		hasCache := false

		nextIndex := index + 1
		if nextIndex < len(mockData.Snapshots) {

			currentRoundIndex := timeIndex.Time2Index(*snapshot.SnapshotHeader.Timestamp)

			minRoundIndex := uint64(0)
			if roundCache.latestRoundIndex >= uint64(len(roundCache.data)) {
				minRoundIndex = roundCache.latestRoundIndex + 1 - uint64(len(roundCache.data))
			}
			maxRoundIndex := uint64(0)
			if roundCache.latestRoundIndex > 0 {
				maxRoundIndex = roundCache.latestRoundIndex - 1
			}
			if currentRoundIndex >= minRoundIndex &&
				currentRoundIndex <= maxRoundIndex {
				nextSnapshot := mockData.Snapshots[nextIndex]
				nextRoundIndex := timeIndex.Time2Index(*nextSnapshot.SnapshotHeader.Timestamp)
				if nextRoundIndex > currentRoundIndex {
					hasCache = true
					fmt.Printf("%d %d %d~%d prev-current: %d %s %d - %d %s %d\n",
						roundCache.latestRoundIndex,
						len(roundCache.data),
						minRoundIndex,
						maxRoundIndex,
						currentRoundIndex,
						snapshot.SnapshotHeader.Hash,
						snapshot.SnapshotHeader.Height,
						nextRoundIndex,
						nextSnapshot.SnapshotHeader.Hash,
						nextSnapshot.SnapshotHeader.Height)
				}
			}

		}

		// check GetSnapshotViteBalanceList
		balanceMap, notFoundAddrList, err := roundCache.GetSnapshotViteBalanceList(snapshot.SnapshotHeader.Hash, mockData.AddrList)
		if err != nil {
			t.Fatal(err)
			t.FailNow()
		}

		storIter := roundCache.StorageIterator(snapshot.SnapshotHeader.Hash)

		if hasCache {
			defer storIter.Release()

			mockBalanceMap := make(map[types.Address]*big.Int)
			var mockNotFoundAddrList []types.Address
			for _, addr := range mockData.AddrList {
				value, err := snapshot.Data.Get(mockBalanceKey(addr, ledger.ViteTokenId))
				if err == leveldb.ErrNotFound {
					mockNotFoundAddrList = append(mockNotFoundAddrList, addr)
				} else {
					mockBalanceMap[addr] = big.NewInt(0).SetBytes(value)
				}

			}

			// check balance map
			var parseData = func(data []*RedoCacheData) string {
				str := ""
				for _, item := range data {
					str += fmt.Sprintf("\n%+v | ", item.lastSnapshotBlock)
				}
				return str
			}
			assert.Equal(t, len(mockBalanceMap), len(balanceMap),
				fmt.Sprintf("mockNotFoundAddrList.length is %d, notFoundAddrList.length is %d, snapshot hash: %s, data: %+v",
					len(mockNotFoundAddrList), len(notFoundAddrList), snapshot.SnapshotHeader.Hash, parseData(roundCache.data)))
			for addr, balance := range balanceMap {
				mockBalance, ok := mockBalanceMap[addr]
				assert.Equal(t, ok, true)
				assert.Check(t, balance.Cmp(mockBalance) == 0)
			}

			// check not found addr list
			assert.Equal(t, len(mockNotFoundAddrList), len(notFoundAddrList))
			for index, addr := range notFoundAddrList {
				assert.Equal(t, addr, mockNotFoundAddrList[index])
			}

			// check iter
			mockIter := NewTransformIterator(snapshot.Data.NewIterator(util.BytesPrefix(mockStorageKey(types.AddressGovernance, nil))), 1+types.AddressSize)
			defer mockIter.Release()
			compareIter(t, storIter, mockIter, true)

		} else {
			// check query balance
			assert.Check(t, balanceMap == nil)
			assert.Check(t, notFoundAddrList == nil)
			assert.NilError(t, err)

			// check storage iterator
			assert.Check(t, storIter == nil)

		}
	}

}

func TestRoundCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	// mock data
	mockData := NewMockData(append(createAddrList(19), types.AddressGovernance), 90)

	// mock chain
	mockChain := getMockChain(ctrl, mockData)

	// mock state db
	mockStateDb := getMockStateDB(ctrl, mockData)

	// mock time2index
	mockGenesisTime := genesisTime
	mockTimeIndex := core.NewTimeIndex(mockGenesisTime, 15*time.Second)

	// new round cache
	roundCount := uint8(3)
	roundCache := NewRoundCache(mockChain, mockStateDb, roundCount)
	fmt.Printf("New round cache, roundCount: %d\n", roundCount)

	// check status
	assert.Equal(t, roundCache.status, STOP)

	//  TODO real address check GetSnapshotViteBalanceList
	balanceMap, notFoundAddressList, err := roundCache.GetSnapshotViteBalanceList(types.Hash{}, []types.Address{})

	assert.Assert(t, is.Nil(balanceMap))
	assert.Assert(t, is.Nil(notFoundAddressList))
	assert.NilError(t, err)

	// check StorageIterator
	iter := roundCache.StorageIterator(types.Hash{})
	assert.Equal(t, iter, nil)

	// test init
	if err := roundCache.Init(mockTimeIndex); err != nil {
		t.Fatal(err)
		t.FailNow()
	}
	fmt.Printf("Init round cache\n")

	// after init

	// check status
	assert.Equal(t, roundCache.status, INITED)

	// check cache length
	checkRoundCache(t, mockData, roundCache, mockChain, mockTimeIndex)

	checkTime := 0
	testInsertSnapshots := func(times int, maxCount int) {
		for i := 0; i < times; i++ {
			count := rand.Intn(maxCount) + 1

			newMockData := mockData.Add(createAddrList(1), count)
			for _, snapshot := range newMockData.Snapshots {

				if err := roundCache.InsertSnapshotBlock(snapshot.SnapshotHeader, snapshot.Log); err != nil {
					t.Fatal(err)
					t.FailNow()
				}
			}
			checkTime++
			fmt.Printf("%d.Insert %d snapshot blocks. Latest: %d\n", checkTime, count, mockChain.GetLatestSnapshotBlock().Height)

			checkRoundCache(t, mockData, roundCache, mockChain, mockTimeIndex)

			checkRoundCacheQuery(t, roundCache, mockData, mockTimeIndex)

		}

	}
	testInsertSnapshots(20, 3)

	testDeleteSnapshots := func(times int, maxCount int) {
		for i := 0; i < times; i++ {
			count := rand.Intn(maxCount) + 1

			deletedMockData := mockData.Delete(count)

			deletedSnapshots := make([]*ledger.SnapshotBlock, 0, len(deletedMockData.Snapshots))

			for _, snapshot := range deletedMockData.Snapshots {
				deletedSnapshots = append(deletedSnapshots, snapshot.SnapshotHeader)
			}

			if len(deletedSnapshots) > 0 {
				if err := roundCache.DeleteSnapshotBlocks(deletedSnapshots); err != nil {
					t.Fatal(err)
					t.FailNow()
				}
			}

			checkTime++

			fmt.Printf("%d.Delete %d snapshot blocks. Latest: %d\n", checkTime, count, mockChain.GetLatestSnapshotBlock().Height)

			checkRoundCache(t, mockData, roundCache, mockChain, mockTimeIndex)

			checkRoundCacheQuery(t, roundCache, mockData, mockTimeIndex)
		}
	}

	testDeleteSnapshots(20, 3)

	for j := 0; j < 10; j++ {
		fmt.Printf("Test %d.1\n", j)
		for i := 0; i < 60; i++ {
			randNum := rand.Intn(100)
			if randNum < 70 {
				testInsertSnapshots(3, 1)
			} else {
				testDeleteSnapshots(5, 1)
			}
		}

		fmt.Printf("Test %d.2\n", j)
		for i := 0; i < 60; i++ {
			randNum := rand.Intn(100)
			if randNum < 70 {
				testInsertSnapshots(1, 1)
			} else {
				testDeleteSnapshots(1, 1)
			}
		}

		fmt.Printf("Test %d.3\n", j)
		for i := 0; i < 60; i++ {
			randNum := rand.Intn(100)
			if randNum < 95 {
				testInsertSnapshots(1, 1)
			} else {
				testDeleteSnapshots(1, 30)
			}
		}

		fmt.Printf("Test %d.4\n", j)
		for i := 0; i < 500; i++ {
			randNum := rand.Intn(100)
			if randNum < 95 {
				testInsertSnapshots(1, 1)
			} else {
				testDeleteSnapshots(1, 80)
			}
		}

		fmt.Printf("Test %d.5\n", j)
		for i := 0; i < 500; i++ {
			randNum := rand.Intn(100)
			if randNum < 20 {
				testInsertSnapshots(1, 3)
			} else {
				testDeleteSnapshots(1, 3)
			}
		}
	}

}
