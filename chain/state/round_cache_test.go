package chain_state

import (
	"encoding/binary"
	"github.com/golang/mock/gomock"
	chain_utils "github.com/vitelabs/go-vite/chain/utils"
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

//
//func copyContractData(data map[types.Address]map[string][]byte) map[types.Address]map[string][]byte {
//	newData := make(map[types.Address]map[string][]byte, len(data))
//	for addr, dataMap := range data {
//		newData[addr] = make(map[string][]byte, len(dataMap))
//		for key, value := range dataMap {
//			newData[addr][key] = value
//		}
//	}
//	return newData
//}
//
//func copyAllBalance(allBalance map[types.Address]map[types.TokenTypeId]*big.Int) map[types.Address]map[types.TokenTypeId]*big.Int {
//	newAllBalance := make(map[types.Address]map[types.TokenTypeId]*big.Int, len(allBalance))
//	for addr, balanceMap := range allBalance {
//		newBalanceMap := make(map[types.TokenTypeId]*big.Int, len(balanceMap))
//		for tokenTypeId, balance := range balanceMap {
//			newBalanceMap[tokenTypeId] = balance
//		}
//		newAllBalance[addr] = newBalanceMap
//
//	}
//	return newAllBalance
//}
func mockStorageKey(addr types.Address, key []byte) []byte {
	return append(append([]byte{chain_utils.StorageKeyPrefix}, addr.Bytes()...), key...)

}

func mockBalanceKey(addr types.Address, tokenId types.TokenTypeId) []byte {
	return append(append([]byte{chain_utils.BalanceKeyPrefix}, addr.Bytes()...), tokenId.Bytes()...)
}

func mockSnapshotState(data *memdb.DB,
	addrList []types.Address, accountCount, keyLength int) SnapshotLog {
	log := make(SnapshotLog)

	for i := 0; i < accountCount; i++ {
		addr := addrList[rand.Intn(len(addrList))]

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

func getMockSnapshotData(addrList []types.Address, snapshotCount uint64) []MockSnapshot {
	var data []MockSnapshot

	//contractData := make(map[types.Address]map[string][]byte)
	//allBalance := make(map[types.Address]map[types.TokenTypeId]*big.Int)
	snapshotData := memdb.New(comparer.DefaultComparer, 0)

	for h := uint64(1); h < snapshotCount; h++ {
		currentTime := genesisTime.Add(time.Duration(h-1) * time.Second)

		//newContractData := copyContractData(contractData)
		//newAllBalance := copyAllBalance(allBalance)

		// snapshot log
		//log := mockSnapshotState(newContractData, newAllBalance, addrList, rand.Intn(5)+1, rand.Intn(5)+1)
		newSnapshotData := snapshotData.Copy()
		log := mockSnapshotState(newSnapshotData, addrList, rand.Intn(5)+1, rand.Intn(5)+1)

		data = append(data, MockSnapshot{
			SnapshotHeader: &ledger.SnapshotBlock{
				Height:    h,
				Timestamp: &currentTime,
			},
			Data: newSnapshotData,
			Log:  log,
		})

		//contractData = newContractData
		//allBalance = newAllBalance
	}
	return data

}

func getMockChain(ctrl *gomock.Controller, mockData []MockSnapshot) *MockChain {

	// mock chain
	mockChain := NewMockChain(ctrl)

	// mock chain.StopWrite
	mockChain.EXPECT().StopWrite().Times(1)
	// mock chain.RecoverWrite
	mockChain.EXPECT().RecoverWrite().Times(1)
	// mock chain.GetLatestSnapshotBlock
	mockChain.EXPECT().GetLatestSnapshotBlock().Return(mockData[len(mockData)-1].SnapshotHeader)
	// mock chain.GetSnapshotHeaderBeforeTime
	mockChain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Any()).DoAndReturn(func(timestamp *time.Time) (*ledger.SnapshotBlock, error) {
		for i := len(mockData) - 1; i >= 0; i-- {
			dataItem := mockData[i]
			sbHeader := dataItem.SnapshotHeader
			if sbHeader.Timestamp.Before(*timestamp) {
				return sbHeader, nil
			}
		}
		return nil, nil
	})
	return mockChain
}

func getMockStateDB(ctrl *gomock.Controller, mockData []MockSnapshot) *MockStateDBInterface {
	// mock state db
	mockStateDb := NewMockStateDBInterface(ctrl)

	// mock stateDb.NewStorageDatabase
	mockStateDb.EXPECT().NewStorageDatabase(gomock.Any(), gomock.Any()).
		DoAndReturn(func(snapshotHash types.Hash, addr types.Address) (StorageDatabaseInterface, error) {
			var snapshotData *MockSnapshot
			for _, dataItem := range mockData {
				if dataItem.SnapshotHeader.Hash == snapshotHash {
					snapshotData = &dataItem
				}
			}
			if snapshotData == nil {
				return nil, errors.New("snapshotData is nil")
			}

			// mock storage database
			storageDatabase := NewMockStorageDatabaseInterface(ctrl)

			// mock address
			storageDatabase.EXPECT().Address().Return(addr)

			// mock get value
			storageDatabase.EXPECT().GetValue(gomock.Any()).DoAndReturn(func(key []byte) ([]byte, error) {
				return snapshotData.Data.Get(mockStorageKey(addr, key))
			})

			// mock new storage iterator
			storageDatabase.EXPECT().NewStorageIterator(gomock.Any()).DoAndReturn(func(prefix []byte) (interfaces.StorageIterator, error) {
				return snapshotData.Data.NewIterator(util.BytesPrefix(mockStorageKey(addr, prefix))), nil
			})

			// *StorageDatabase, error
			//NewStorageDatabase(snapshotHash, addr)(*StorageDatabase, error)

			return storageDatabase, nil
		})
	return mockStateDb
}

func TestRoundCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	// mock address list
	mockAddrList := createAddrList(20)

	// mock data
	mockData := getMockSnapshotData(mockAddrList, 180)

	// mock chain
	mockChain := getMockChain(ctrl, mockData)

	// mock state db
	mockStateDb := getMockStateDB(ctrl, mockData)

	// mock time2index
	mockGenesisTime := genesisTime
	mockTimeIndex := core.NewTimeIndex(mockGenesisTime, time.Second)

	// new round cache
	roundCache := NewRoundCache(mockChain, mockStateDb, 3)

	// after new round cache
	t.Run("after NewRoundCache", func(t *testing.T) {
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
	})

	// test init
	if err := roundCache.Init(mockTimeIndex); err != nil {
		t.Fatal(err)
		t.FailNow()
	}

	// after init
	t.Run("after init", func(t *testing.T) {
		// check status
		assert.Equal(t, roundCache.status, INITED)
	})

}
