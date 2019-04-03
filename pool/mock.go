package pool

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm_db"
)

var logger = log15.New("module", "pool/mock")

type MockSyncer struct {
}

func (*MockSyncer) FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64) {
	panic("implement me")
}

func (*MockSyncer) SyncState() net.SyncState {
	return net.Syncdone
}

func (*MockSyncer) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
	logger.Info("BroadcastSnapshotBlocks")
}

func (*MockSyncer) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
	logger.Info("BroadcastSnapshotBlock")
}

func (*MockSyncer) BroadcastAccountBlock(block *ledger.AccountBlock) {
	logger.Info("BroadcastAccountBlock")
}

func (*MockSyncer) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
	logger.Info("BroadcastAccountBlocks")
}

func (*MockSyncer) FetchSnapshotBlocks(start types.Hash, count uint64) {
	logger.Info("FetchSnapshotBlocks")
}

func (*MockSyncer) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
	logger.Info("FetchAccountBlocks")
}

func (*MockSyncer) SubscribeAccountBlock(fn net.AccountBlockCallback) (subId int) {
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

type MockChain struct {
}

func (*MockChain) InsertAccountBlocks(vmAccountBlocks []*vm_db.VmDb) error {
	logger.Info("InsertAccountBlocks")
	return nil
}

func (*MockChain) GetLatestAccountBlock(addr *types.Address) (*ledger.AccountBlock, error) {
	logger.Info("GetLatestAccountBlock")
	return nil, nil
}

func (*MockChain) GetAccountBlockByHeight(addr *types.Address, height uint64) (*ledger.AccountBlock, error) {
	logger.Info("GetAccountBlockByHeight")
	return nil, nil
}

func (*MockChain) DeleteAccountBlocks(addr *types.Address, toHeight uint64) (map[types.Address][]*ledger.AccountBlock, error) {
	logger.Info("DeleteInvalidAccountBlocks")
	return nil, nil
}

func (*MockChain) GetUnConfirmAccountBlocks(addr *types.Address) []*ledger.AccountBlock {
	logger.Info("GetUnConfirmAccountBlocks")
	return nil
}

func (*MockChain) GetFirstConfirmedAccountBlockBySbHeight(snapshotBlockHeight uint64, addr *types.Address) (*ledger.AccountBlock, error) {
	logger.Info("GetFirstConfirmedAccountBlockBySbHeight")
	return nil, nil
}

func (*MockChain) GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error) {
	logger.Info("GetSnapshotBlockByHeight")
	return nil, nil
}

func (*MockChain) GetLatestSnapshotBlock() *ledger.SnapshotBlock {
	logger.Info("GetLatestSnapshotBlock")
	return nil
}

func (*MockChain) GetSnapshotBlockByHash(hash *types.Hash) (*ledger.SnapshotBlock, error) {
	logger.Info("GetSnapshotBlockByHash")
	return nil, nil
}

func (*MockChain) InsertSnapshotBlock(snapshotBlock *ledger.SnapshotBlock) error {
	logger.Info("InsertSnapshotBlock")
	return nil
}

func (*MockChain) DeleteSnapshotBlocksToHeight(toHeight uint64) ([]*ledger.SnapshotBlock, map[types.Address][]*ledger.AccountBlock, error) {
	logger.Info("DeleteSnapshotBlocksToHeight")
	return nil, nil, nil
}

func (*MockChain) GetAccountBlockByHash(blockHash *types.Hash) (*ledger.AccountBlock, error) {
	logger.Info("GetAccountBlockByHash")
	return nil, nil
}
