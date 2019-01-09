package consensus

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
)

var mockLogger = log15.New("module", "consensusMock")

type MockConsensus struct {
}

func (*MockConsensus) SubscribeProducers(gid types.Gid, id string, fn func(event ProducersEvent)) {
	panic("implement me")
}

func (*MockConsensus) UnSubscribeProducers(gid types.Gid, id string) {
	panic("implement me")
}

func (*MockConsensus) Subscribe(gid types.Gid, id string, addr *types.Address, fn func(Event)) {
	mockLogger.Info("Subscribe")
}

func (*MockConsensus) UnSubscribe(gid types.Gid, id string) {
	mockLogger.Info("UnSubscribe")
}

func (*MockConsensus) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	mockLogger.Info("VerifyAccountProducer")
	return true, nil
}

func (*MockConsensus) VerifySnapshotProducer(block *ledger.SnapshotBlock) (bool, error) {
	mockLogger.Info("VerifySnapshotProducer")
	return true, nil
}
