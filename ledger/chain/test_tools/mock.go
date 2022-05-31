package test_tools

import (
	"time"

	"github.com/golang/mock/gomock"
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/ledger/consensus/core"
)

func NewPeriodTimeIndex(genesisTime *time.Time) interfaces.TimeIndex {
	genesis := time.Unix(genesisTime.Unix(), 0)
	ti := core.NewTimeIndex(genesis, time.Second*75)
	return ti
}

func NewSbpStatReader(ctrl *gomock.Controller) *core.MockSBPStatReader {
	return core.NewMockSBPStatReader(ctrl)
}

func NewVerifier() interfaces.ConsensusVerifier {
	return &MockCssVerifier{}
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
