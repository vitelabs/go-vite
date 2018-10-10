package onroad

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm_context"
	"github.com/vitelabs/go-vite/wallet"
	"github.com/vitelabs/go-vite/producer/producerevent"
)

type Vite interface {
	Net() Net
	Chain() chain.Chain
	WalletManager() *wallet.Manager
	Producer() Producer
	PoolReader
	ConsensusReader
}

type PoolReader interface {
	ExistInPool(address types.Address, fromBlockHash types.Hash) bool
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error
	AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error
}

type ConsensusReader interface {
	VerifyAccountProducer(block *ledger.AccountBlock) (bool, error)
}

type Producer interface {
	SetAccountEventFunc(func(event producerevent.AccountEvent))
}

type Net interface {
	SubscribeSyncStatus(fn func(net.SyncState)) (subId int)
	UnsubscribeSyncStatus(subId int)
	Status() *net.NetStatus
}

type Chain interface {
}
