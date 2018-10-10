package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/onroad"
	"math/big"
)

type OnroadApi struct {
	manager *onroad.Manager
}

func NewOnroadApi(manager *onroad.Manager) *OnroadApi {
	return &OnroadApi{
		manager: manager,
	}
}

func (o OnroadApi) String() string {
	return "OnroadApi"
}

func (o OnroadApi) ListWorkingAutoReceiveWorker() []types.Address {
	return o.manager.ListWorkingAutoReceiveWorker()
}

func (o OnroadApi) StartAutoReceive(addr types.Address, filter map[types.TokenTypeId]string) error {
	rawfilter := make(map[types.TokenTypeId]big.Int)
	if filter != nil {
		for k, v := range filter {
			b, ok := new(big.Int).SetString(v, 10)
			if !ok {
				return ErrStrToBigInt
			}
			rawfilter[k] = *b
		}
	}

	return o.manager.StartAutoReceiveWorker(addr, rawfilter)
}

func (o OnroadApi) StopAutoReceive(addr types.Address) error {
	return o.manager.StopAutoReceiveWorker(addr)
}

func (o OnroadApi) GetOnroadBlocksByAddress(address types.Address, index int, count int) {

}

func (o OnroadApi) GetAccountOnroadInfo() {

}
