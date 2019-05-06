package consensus

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/vitelabs/go-vite/consensus/core"
	"github.com/vitelabs/go-vite/ledger"
)

func TestRollbackProof_ProofEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	// Assert that Bar() is invoked.
	defer ctrl.Finish()
	mock_chain := NewMockChain(ctrl)
	proof := newRollbackProof(mock_chain)

	now := time.Now()
	timeIndex := core.NewTimeIndex(now, time.Second*8)
	stime, etime := timeIndex.Index2Time(9)
	{
		t1 := stime.Add(time.Second)
		mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&etime)).Return(&ledger.SnapshotBlock{Timestamp: &t1}, nil)
		result, err := proof.ProofEmpty(stime, etime)

		assert.NoError(t, err)
		assert.False(t, result)
	}

	{
		t1 := stime.Add(-time.Second)
		mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&etime)).Return(&ledger.SnapshotBlock{Timestamp: &t1}, nil)
		result, err := proof.ProofEmpty(stime, etime)

		assert.NoError(t, err)
		assert.True(t, result)
	}

	{
		t1 := stime.Add(0)
		mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&etime)).Return(&ledger.SnapshotBlock{Timestamp: &t1}, nil)
		result, err := proof.ProofEmpty(stime, etime)

		assert.NoError(t, err)
		assert.False(t, result)
	}

	{
		t1 := etime
		mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&etime)).Return(&ledger.SnapshotBlock{Timestamp: &t1}, nil)
		result, err := proof.ProofEmpty(stime, etime)

		assert.Error(t, err)
		assert.False(t, result)
	}

	{
		t1 := etime.Add(time.Second)
		mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&etime)).Return(&ledger.SnapshotBlock{Timestamp: &t1}, nil)
		result, err := proof.ProofEmpty(stime, etime)

		assert.Error(t, err)
		assert.False(t, result)
	}

	{
		t1 := stime.Add(time.Second)
		mock_chain.EXPECT().GetSnapshotHeaderBeforeTime(gomock.Eq(&etime)).Return(&ledger.SnapshotBlock{Timestamp: &t1}, errors.New("mock error"))
		result, err := proof.ProofEmpty(stime, etime)

		assert.Error(t, err)
		assert.False(t, result)
	}
}
