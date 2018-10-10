package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite/net"
)

var logger = log15.New()

type MockSyncer struct {
}

func (*MockSyncer) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	logger.Info("BroadcastSnapshotBlock")
}

func (*MockSyncer) BroadcastAccountBlock(addr types.Address, block *ledger.AccountBlock) {
	logger.Info("BroadcastAccountBlock")
}

func (*MockSyncer) BroadcastAccountBlocks(addr types.Address, blocks []*ledger.AccountBlock) {
	logger.Info("BroadcastAccountBlocks")
}

func (*MockSyncer) FetchSnapshotBlocks(start types.Hash, count uint64) {
	logger.Info("FetchSnapshotBlocks")
}

func (*MockSyncer) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	logger.Info("FetchAccountBlocks")
}

func (*MockSyncer) SubscribeAccountBlock(fn net.AccountblockCallback) (subId int) {
	logger.Info("SubscribeAccountBlock")
	return 12
}

func (*MockSyncer) UnsubscribeAccountBlock(subId int) {
	logger.Info("UnsubscribeAccountBlock")
}

func (*MockSyncer) SubscribeSnapshotBlock(fn net.SnapshotBlockCallback) (subId int) {
	logger.Info("SubscribeSnapshotBlock")
	return 11
}

func (*MockSyncer) UnsubscribeSnapshotBlock(subId int) {
	logger.Info("UnsubscribeSnapshotBlock")
}

func (*MockSyncer) SubscribeSyncStatus(fn net.SyncStateCallback) (subId int) {
	logger.Info("SubscribeSyncStatus")
	return 10
}

func (*MockSyncer) UnsubscribeSyncStatus(subId int) {
	logger.Info("UnsubscribeSyncStatus")
}
