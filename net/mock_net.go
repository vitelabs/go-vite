package net

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/crypto/ed25519"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
	"github.com/vitelabs/go-vite/v2/net/vnode"
	_version "github.com/vitelabs/go-vite/v2/version"
)

type mockNet struct {
	chain Chain
}

func (n *mockNet) PeerKey() ed25519.PrivateKey {
	return nil
}

func (n *mockNet) SubscribeSyncStatus(fn SyncStateCallback) (subId int) {
	return 0
}

func (n *mockNet) UnsubscribeSyncStatus(subId int) {
}

func (n *mockNet) SyncState() SyncState {
	return SyncDone
}

func (n *mockNet) Peek() *Chunk {
	return nil
}

func (n *mockNet) Pop(endHash types.Hash) {
}

func (n *mockNet) Status() SyncStatus {
	return SyncStatus{
		Current: n.chain.GetLatestSnapshotBlock().Height,
		State:   SyncDone,
	}
}

func (n *mockNet) Detail() SyncDetail {
	return SyncDetail{
		SyncStatus:       n.Status(),
		DownloaderStatus: DownloaderStatus{},
	}
}

func (n *mockNet) FetchSnapshotBlocks(start types.Hash, count uint64) {
}

func (n *mockNet) FetchSnapshotBlocksWithHeight(hash types.Hash, height uint64, count uint64) {
}

func (n *mockNet) FetchAccountBlocks(start types.Hash, count uint64, address *types.Address) {
}

func (n *mockNet) FetchAccountBlocksWithHeight(start types.Hash, count uint64, address *types.Address, sHeight uint64) {
}

func (n *mockNet) BroadcastSnapshotBlock(block *ledger.SnapshotBlock) {
}

func (n *mockNet) BroadcastSnapshotBlocks(blocks []*ledger.SnapshotBlock) {
}

func (n *mockNet) BroadcastAccountBlock(block *ledger.AccountBlock) {
}

func (n *mockNet) BroadcastAccountBlocks(blocks []*ledger.AccountBlock) {
}

func (n *mockNet) SubscribeAccountBlock(fn AccountBlockCallback) (subId int) {
	return 0
}

func (n *mockNet) UnsubscribeAccountBlock(subId int) {
}

func (n *mockNet) SubscribeSnapshotBlock(fn SnapshotBlockCallback) (subId int) {
	return 0
}

func (n *mockNet) UnsubscribeSnapshotBlock(subId int) {
}

func (n *mockNet) Stop() error {
	return nil
}

func (n *mockNet) Start() error {
	return nil
}

func (n *mockNet) Info() NodeInfo {
	info := NodeInfo{
		Version:      version,
		BuildVersion: _version.VITE_BUILD_VERSION,
	}
	return info
}

func (n *mockNet) Nodes() []*vnode.Node {
	return nil
}

func (n *mockNet) PeerCount() int {
	return 0
}

func mock(chain Chain) Net {
	return &mockNet{
		chain: chain,
	}
}

func Mock(chain Chain) Net {
	return mock(chain)
}
