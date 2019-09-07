package onroad

import (
	"time"

	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vm_db"
)

type mockVite struct {
	chain    chain.Chain
	net      netReader
	producer producer
	pool     pool
}

func NewMockVite(chain chain.Chain) *mockVite {
	return &mockVite{
		chain:    chain,
		net:      new(mockNet),
		producer: new(mockProducer),
		pool:     new(mockPool),
	}
}

func (v *mockVite) Chain() chain.Chain {
	return v.chain
}

func (v *mockVite) Net() netReader {
	return v.net
}

func (v *mockVite) Producer() producer {
	return v.producer
}

func (v *mockVite) Pool() pool {
	return v.pool
}

type mockPool struct{}

func (mp *mockPool) AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_db.VmAccountBlock) error {
	return nil
}

type mockProducer struct {
	Addr types.Address
	f    func(event producerevent.AccountEvent)
}

func (mp *mockProducer) SetAccountEventFunc(f func(producerevent.AccountEvent)) { mp.f = f }

func (mp *mockProducer) produceEvent(duration time.Duration) {
	mp.f(producerevent.AccountStartEvent{
		Gid:     types.DELEGATE_GID,
		Address: mp.Addr,
		Stime:   time.Now(),
		Etime:   time.Now().Add(duration),
	})
}

type mockNet struct {
	fn func(net.SyncState)
}

func (mn *mockNet) SubscribeSyncStatus(fn func(net.SyncState)) (subId int) {
	mn.fn = fn
	return 0
}

func (mn *mockNet) UnsubscribeSyncStatus(subId int) {}

func (mn *mockNet) SyncState() net.SyncState { return net.SyncDone }
