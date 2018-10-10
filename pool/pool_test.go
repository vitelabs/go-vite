package pool

import (
	"testing"

	ch "github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/config"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite/net"
)

var logger = log15.New()

type testSyncer struct {
}

func (*testSyncer) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	logger.Info("BroadcastSnapshotBlock")
}

func (*testSyncer) BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock) {
	logger.Info("BroadcastAccountBlock")
}

func (*testSyncer) BroadcastAccountBlocks(addr types.Address, blocks []*ledger.AccountBlock) {
	logger.Info("BroadcastAccountBlocks")
}

func (*testSyncer) FetchSnapshotBlocks(start types.Hash, count uint64) {
	logger.Info("FetchSnapshotBlocks")
}

func (*testSyncer) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	logger.Info("FetchAccountBlocks")
}

func (*testSyncer) SubscribeAccountBlock(fn net.AccountblockCallback) (subId int) {
	logger.Info("SubscribeAccountBlock")
	return 12
}

func (*testSyncer) UnsubscribeAccountBlock(subId int) {
	logger.Info("UnsubscribeAccountBlock")
}

func (*testSyncer) SubscribeSnapshotBlock(fn net.SnapshotBlockCallback) (subId int) {
	logger.Info("SubscribeSnapshotBlock")
	return 11
}

func (*testSyncer) UnsubscribeSnapshotBlock(subId int) {
	logger.Info("UnsubscribeSnapshotBlock")
}

func (*testSyncer) SubscribeSyncStatus(fn net.SyncStateCallback) (subId int) {
	logger.Info("SubscribeSyncStatus")
	return 10
}

func (*testSyncer) UnsubscribeSyncStatus(subId int) {
	logger.Info("UnsubscribeSyncStatus")
}

func TestNewPool(t *testing.T) {
	innerChainInstance := ch.NewChain(&config.Config{
		DataDir: common.DefaultDataDir(),
		//Chain: &config.Chain{
		//	KafkaProducers: []*config.KafkaProducer{{
		//		Topic:      "test",
		//		BrokerList: []string{"abc", "def"},
		//	}},
		//},
	})
	innerChainInstance.Init()
	innerChainInstance.Start()
	newPool := NewPool(innerChainInstance)
	newPool.Init(&testSyncer{})
	newPool.Start()

	make(chan int) <- 1
}
