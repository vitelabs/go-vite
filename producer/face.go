package producer

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	"github.com/vitelabs/go-vite/v2/producer/producerevent"
)

type Producer interface {
	SetAccountEventFunc(func(producerevent.AccountEvent))
	Init() error
	Start() error
	Stop() error
	GetCoinBase() types.Address
	SnapshotOnce() error
}
