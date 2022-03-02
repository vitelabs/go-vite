package onroad

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/interfaces"
	"github.com/vitelabs/go-vite/v2/net"
	"github.com/vitelabs/go-vite/v2/producer/producerevent"
)

type pool interface {
	AddDirectAccountBlock(address types.Address, vmAccountBlock *interfaces.VmAccountBlock) error
}

type producer interface {
	SetAccountEventFunc(func(producerevent.AccountEvent))
}

type netReader interface {
	SubscribeSyncStatus(fn func(net.SyncState)) (subId int)
	UnsubscribeSyncStatus(subId int)
	SyncState() net.SyncState
}
