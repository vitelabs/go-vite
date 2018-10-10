package onroad

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/pool"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/wallet"
)

type Vite interface {
	Net() *net.Net
	Chain() chain.Chain
	WalletManager() *wallet.Manager
	Producer() producer.Producer
	Pool() pool.BlockPool
}

//type PoolReader interface {
//	ExistInPool(address types.Address, fromBlockHash types.Hash) bool
//	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error
//	AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error
//}

//type Producer interface {
//	SetAccountEventFunc(func(producerevent.AccountEvent))
//}

//type Net interface {
//SubscribeSyncStatus(fn func(net.SyncState)) (subId int)
//UnsubscribeSyncStatus(subId int)
//Status() *net.NetStatus
//}
