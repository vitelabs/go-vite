package onroad_test

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"time"
)

type testNet struct {
	fn func(net.SyncState)
}

func (t *testNet) SubscribeSyncStatus(fn func(net.SyncState)) (subId int) {
	t.fn = fn
	return 0
}

func (testNet) UnsubscribeSyncStatus(subId int) {
}

func (testNet) SyncState() net.SyncState {
	return net.Syncdone
}

type testProducer struct {
	Addr types.Address
	f    func(event producerevent.AccountEvent)
}

type testPool struct {
}

func (t testProducer) produceEvent(duration time.Duration) {
	t.f(producerevent.AccountStartEvent{
		Gid:            types.SNAPSHOT_GID,
		Address:        t.Addr,
		Stime:          time.Now(),
		Etime:          time.Now().Add(duration),
		Timestamp:      time.Now(),
		SnapshotHash:   types.Hash{},
		SnapshotHeight: 0,
	})
}

func (t *testProducer) SetAccountEventFunc(f func(event producerevent.AccountEvent)) {
	t.f = f
}

type testVite struct {
	chain    chain.Chain
	wallet   *wallet.Manager
	producer onroad.Producer
	pool     onroad.Pool
}

func (testVite) Net() onroad.Net {
	return new(testNet)
}

func (t testVite) Chain() chain.Chain {
	return t.chain
}

func (t testVite) WalletManager() *wallet.Manager {
	return t.wallet
}

func (t testVite) Producer() onroad.Producer {
	return t.producer
}
func (t testVite) Pool() onroad.Pool {
	return t.pool
}

func (testPool) ExistInPool(address types.Address, fromBlockHash types.Hash) bool {
	return false
}

func (testPool) AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error {
	return nil
}

func (testPool) AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock,
	sendBlocks []*vm_context.VmAccountBlock) error {
	return nil
}

func (testVite) VerifyAccountProducer(block *ledger.AccountBlock) (bool, error) {
	return false, nil
}
