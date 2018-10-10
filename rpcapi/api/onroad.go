package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/onroad/model"
	"math/big"
)

func onroadInfoToRpcAccountInfo(chain chain.Chain, onroadInfo model.OnroadAccountInfo) *RpcAccountInfo {
	var r RpcAccountInfo
	r.AccountAddress = *onroadInfo.AccountAddress
	r.TotalNumber = string(onroadInfo.TotalNumber)
	r.TokenBalanceInfoMap = make(map[types.TokenTypeId]*RpcTokenBalanceInfo)
	for tti, v := range onroadInfo.TokenBalanceInfoMap {
		if v != nil {
			tinfo := chain.GetTokenInfoById(&tti)
			b := &RpcTokenBalanceInfo{
				TokenInfo:   RawTokenInfoToRpc(tinfo),
				TotalAmount: v.TotalAmount.String(),
				Number:      string(v.Number),
			}
			r.TokenBalanceInfoMap[tti] = b
		}
	}
	return nil
}

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

func (o OnroadApi) GetAccountOnroadInfo(address types.Address) {
	o.manager.GetOnroadBlocksPool().GetCommonAccountInfo(address)

}
