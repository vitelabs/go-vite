package producer

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/producer/producerevent"
)

type Producer interface {
	SetAccountEventFunc(func(producerevent.AccountEvent))
	Init() error
	Start() error
	Stop() error
	GetCoinBase() types.Address
}
