package chain_state

import (
	"github.com/golang/mock/gomock"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
	"gotest.tools/assert"
	is "gotest.tools/assert/cmp"
	"testing"
	"time"
)

type MockSnapshot struct {
	SnapshotHeader *ledger.SnapshotBlock
}

var genesisTime = time.Unix(1563182961, 0)

func getMockSnapshotData() []MockSnapshot {
	var data []MockSnapshot

	for h := uint64(1); h < 180; h++ {
		currentTime := genesisTime.Add(time.Duration(h-1) * time.Second)
		data = append(data, MockSnapshot{
			SnapshotHeader: &ledger.SnapshotBlock{
				Height:    h,
				Timestamp: &currentTime,
			},
		})
	}
	return data

}

func TestRoundCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()

	mockData := getMockSnapshotData()

	// mock chain
	mockChain := NewMockChain(ctrl)

	// mock chain.StopWrite
	mockChain.EXPECT().StopWrite().Times(1)
	// mock chain.RecoverWrite
	mockChain.EXPECT().RecoverWrite().Times(1)
	// mock chain.GetLatestSnapshotBlock
	mockChain.EXPECT().GetLatestSnapshotBlock().Return(mockData[len(mockData)-1].SnapshotHeader)
	// mock chain.GetSnapshotHeaderBeforeTime
	mockChain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Any()).DoAndReturn(func(timestamp time.Time) *ledger.SnapshotBlock {
		for i := len(mockData) - 1; i >= 0; i-- {
			dataItem := mockData[i]
			sbHeader := dataItem.SnapshotHeader
			if sbHeader.Timestamp.Before(timestamp) {
				return sbHeader
			}
		}
		return nil
	})

	// mock state db
	mockStateDb := NewMockStateDBInterface(ctrl)

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
