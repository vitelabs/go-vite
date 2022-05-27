package test_tools

import (
	"time"

	"github.com/golang/mock/gomock"
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
)

type MockConsensus struct{
	sbpReader *core.MockSBPStatReader
}

func NewMockConsensus(genesisTime *time.Time, ctrl *gomock.Controller) *MockConsensus {
	mock := &MockConsensus{}
	mock.sbpReader = core.NewMockSBPStatReader(ctrl)
	genesis := time.Unix(genesisTime.Unix(), 0)
	ti := core.NewTimeIndex(genesis, time.Hour*24)
	mock.sbpReader.EXPECT().GetPeriodTimeIndex().Return(ti).AnyTimes()
	return mock
}

func (c *MockConsensus) VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	return nil, nil
}

func (c *MockConsensus) SBPReader() core.SBPStatReader {
	return c.sbpReader
}

func (c *MockConsensus) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	return true, nil
}

type MockCssVerifier struct{}

func (c *MockCssVerifier) VerifyABsProducer(abs map[types.Gid][]*ledger.AccountBlock) ([]*ledger.AccountBlock, error) {
	return nil, nil
}

func (c *MockCssVerifier) VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error) {
	return true, nil
}

func (c *MockCssVerifier) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	return true, nil
}
