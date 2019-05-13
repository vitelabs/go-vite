package net

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"

	"github.com/vitelabs/go-vite/p2p"
)

type mockNet struct {
	chain Chain
}

func (n *mockNet) ProtoData() (key []byte, height uint64, genesis types.Hash) {
	return
}

func (n *mockNet) ReceiveHandshake(msg *p2p.HandshakeMsg) (level p2p.Level, err error) {
	return
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
		From:    0,
		To:      0,
		Current: n.chain.GetLatestSnapshotBlock().Height,
		State:   SyncDone,
	}
}

func (n *mockNet) Detail() SyncDetail {
	return SyncDetail{
		From:    0,
		To:      0,
		Current: n.chain.GetLatestSnapshotBlock().Height,
		State:   SyncDone,
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

func (n *mockNet) Trace() {

}

func (n *mockNet) Stop() error {
	return nil
}

func (n *mockNet) Start(svr p2p.P2P) error {
	return nil
}

func (n *mockNet) Name() string {
	return "mock_net"
}

func (n *mockNet) Auth(input []byte) (output []byte) {
	return nil
}

func (n *mockNet) Handshake(their []byte) error {
	return nil
}

func (n *mockNet) Handle(msg p2p.Msg) error {
	return nil
}

func (n *mockNet) State() []byte {
	return nil
}

func (n *mockNet) SetState(state []byte, peer p2p.Peer) {
	return
}

func (n *mockNet) OnPeerAdded(peer p2p.Peer) error {
	return nil
}

func (n *mockNet) OnPeerRemoved(peer p2p.Peer) error {
	return nil
}

func (n *mockNet) Info() NodeInfo {
	return NodeInfo{}
}

func mock(cfg Config) Net {
	return &mockNet{
		chain: cfg.Chain,
	}
}
