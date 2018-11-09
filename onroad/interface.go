package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vite/net"
	"github.com/vitelabs/go-vite/vm_context"
)

type Pool interface {
	ExistInPool(address types.Address, fromBlockHash types.Hash) bool
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_context.VmAccountBlock) error
	AddDirectAccountBlocks(address types.Address, received *vm_context.VmAccountBlock, sendBlocks []*vm_context.VmAccountBlock) error
}

type Producer interface {
	SetAccountEventFunc(func(producerevent.AccountEvent))
}

type Net interface {
	SubscribeSyncStatus(fn func(net.SyncState)) (subId int)
	UnsubscribeSyncStatus(subId int)
	SyncState() net.SyncState
}