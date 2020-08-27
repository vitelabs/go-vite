package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/net"
	"github.com/vitelabs/go-vite/producer/producerevent"
	"github.com/vitelabs/go-vite/vm_db"
)

type pool interface {
	AddDirectAccountBlock(address types.Address, vmAccountBlock *vm_db.VmAccountBlock) error
}

type producer interface {
	SetAccountEventFunc(func(producerevent.AccountEvent))
}

type netReader interface {
	SubscribeSyncStatus(fn func(net.SyncState)) (subId int)
	UnsubscribeSyncStatus(subId int)
	SyncState() net.SyncState
}
