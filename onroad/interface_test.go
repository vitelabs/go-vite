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
)

type testNet struct {
}

func (testNet) SubscribeSyncStatus(fn func(net.SyncState)) (subId int) {
	return 0
}

func (testNet) UnsubscribeSyncStatus(subId int) {
}

func (testNet) Status() *net.NetStatus {
	return &net.NetStatus{
		Peers:     nil,
		SyncState: net.Syncdone,
		Running:   false,
	}
}

type testProducer struct {
}

func (testProducer) SetAccountEventFunc(func(event producerevent.AccountEvent)) {
}

type testVite struct {
}

func (testVite) Net() onroad.Net {
	return new(testNet)
}

func (testVite) Chain() chain.Chain {
	return chain.NewChain(nil)
}

func (testVite) WalletManager() *wallet.Manager {
	return wallet.New(nil)
}

func (testVite) Producer() onroad.Producer {
	return new(testProducer)
}

func (testVite) ExistInPool(address types.Address, fromBlockHash types.Hash) bool {
	return false
}

func (testVite) AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error {
	return nil
}

func (testVite) AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock,
	sendBlocks []*vm_context.VmAccountBlock) error {
	return nil
}

func (testVite) VerifyAccountProducer(block *ledger.AccountBlock) error {
	return nil
}
