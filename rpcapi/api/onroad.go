package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/onroad/model"
	"math/big"
	"strconv"
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

func (o OnroadApi) GetOnroadBlocksByAddress(address types.Address, index int, count int) ([]*AccountBlock, error) {
	blockList, err := o.manager.DbAccess().GetOnroadBlocks(uint64(index), 1, uint64(count), &address)
	if err != nil {
		return nil, err
	}

	a := make([]*AccountBlock, len(blockList))
	for k, v := range blockList {
		if v != nil {
			confirmedTimes, e := o.manager.Chain().GetConfirmTimes(&v.Hash)
			if e != nil {
				return nil, e

			}
			block := createAccountBlock(v, o.manager.DbAccess().Chain.GetTokenInfoById(&v.TokenId), confirmedTimes)
			a[k] = block
		}
	}
	return a, nil
}

func (o OnroadApi) GetAccountOnroadInfo(address types.Address) (*RpcAccountInfo, error) {
	info, e := o.manager.GetOnroadBlocksPool().GetOnroadAccountInfo(address)
	if e != nil || info == nil {
		return nil, e
	}

	r := onroadInfoToRpcAccountInfo(o.manager.Chain(), *info)
	return r, nil

}

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
				Number:      strconv.FormatUint(v.Number, 10),
			}
			r.TokenBalanceInfoMap[tti] = b
		}
	}
	return nil
}
