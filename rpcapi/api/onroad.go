package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/onroad/model"
	"math/big"
	"strconv"
)

type PublicOnroadApi struct {
	api *PrivateOnroadApi
}

func (o PublicOnroadApi) String() string {
	return "PublicOnroadApi"
}

func NewPublicOnroadApi(manager *onroad.Manager) *PublicOnroadApi {
	return &PublicOnroadApi{
		api: NewPrivateOnroadApi(manager),
	}
}
func (o PublicOnroadApi) GetOnroadBlocksByAddress(address types.Address, index int, count int) ([]*AccountBlock, error) {
	return o.api.GetOnroadBlocksByAddress(address, index, count)
}

func (o PublicOnroadApi) GetAccountOnroadInfo(address types.Address) (*RpcAccountInfo, error) {
	return o.api.GetAccountOnroadInfo(address)

}

type PrivateOnroadApi struct {
	manager *onroad.Manager
}

func NewPrivateOnroadApi(manager *onroad.Manager) *PrivateOnroadApi {
	return &PrivateOnroadApi{
		manager: manager,
	}
}

func (o PrivateOnroadApi) String() string {
	return "PrivateOnroadApi"
}

func (o PrivateOnroadApi) ListWorkingAutoReceiveWorker() []types.Address {
	return o.manager.ListWorkingAutoReceiveWorker()
}

func (o PrivateOnroadApi) StartAutoReceive(addr types.Address, filter map[types.TokenTypeId]string) error {
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

func (o PrivateOnroadApi) StopAutoReceive(addr types.Address) error {
	return o.manager.StopAutoReceiveWorker(addr)
}

func (o PrivateOnroadApi) GetOnroadBlocksByAddress(address types.Address, index int, count int) ([]*AccountBlock, error) {
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

func (o PrivateOnroadApi) GetAccountOnroadInfo(address types.Address) (*RpcAccountInfo, error) {
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
	r.TotalNumber = strconv.FormatUint(onroadInfo.TotalNumber, 10)
	r.TokenBalanceInfoMap = make(map[types.TokenTypeId]*RpcTokenBalanceInfo)

	for tti, v := range onroadInfo.TokenBalanceInfoMap {
		if v != nil {
			number := strconv.FormatUint(v.Number, 10)
			tinfo := chain.GetTokenInfoById(&tti)
			b := &RpcTokenBalanceInfo{
				TokenInfo:   RawTokenInfoToRpc(tinfo),
				TotalAmount: v.TotalAmount.String(),
				Number:      &number,
			}
			r.TokenBalanceInfoMap[tti] = b
		}
	}
	return nil
}
