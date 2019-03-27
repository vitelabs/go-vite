package api

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/vite"
)

type PublicOnroadApi struct {
	api *PrivateOnroadApi
}

func (o PublicOnroadApi) String() string {
	return "PublicOnroadApi"
}

func NewPublicOnroadApi(vite *vite.Vite) *PublicOnroadApi {
	return &PublicOnroadApi{
		api: NewPrivateOnroadApi(vite),
	}
}
func (o PublicOnroadApi) GetOnroadBlocksByAddress(address types.Address, index int, count int) ([]*AccountBlock, error) {
	return o.api.GetOnroadBlocksByAddress(address, index, count)
}

/*func (o PublicOnroadApi) GetAccountOnroadInfo(address types.Address) (*RpcAccountInfo, error) {
	return o.api.GetAccountOnroadInfo(address)
}*/

func (o PrivateOnroadApi) GetContractAddrListByGid(gid types.Gid) ([]types.Address, error) {
	return o.manager.Chain().GetContractList(gid)
}

type PrivateOnroadApi struct {
	manager *onroad.Manager
}

func NewPrivateOnroadApi(vite *vite.Vite) *PrivateOnroadApi {
	return &PrivateOnroadApi{
		manager: vite.OnRoad(),
	}
}

func (o PrivateOnroadApi) String() string {
	return "PrivateOnroadApi"
}
func (o PrivateOnroadApi) GetOnroadBlocksByAddress(address types.Address, index int, count int) ([]*AccountBlock, error) {
	log.Info("GetOnroadBlocksByAddress", "addr", address, "index", index, "count", count)
	blockList, err := o.manager.GetOnRoadBlockByAddr(&address, uint64(index), uint64(count))
	if err != nil {
		return nil, err
	}

	a := make([]*AccountBlock, len(blockList))
	sum := 0
	for k, v := range blockList {
		if v != nil {
			accountBlock, e := ledgerToRpcBlock(v, o.manager.Chain())
			if e != nil {
				return nil, e
			}
			a[k] = accountBlock
			sum++
		}
	}
	return a[:sum], nil
}

/*
func (o PrivateOnroadApi) ListWorkingAutoReceiveWorker() []types.Address {
	log.Info("ListWorkingAutoReceiveWorker")
	return o.manager.ListWorkingAutoReceiveWorker()
}

func (o PrivateOnroadApi) StartAutoReceive(entropystore string, addr types.Address, filter map[string]string, powDifficulty *string) error {
	log.Info("StartAutoReceive", "addr", addr, "entropystore", entropystore)

	rawfilter := make(map[types.TokenTypeId]big.Int)
	if filter != nil {
		for k, v := range filter {
			b, ok := new(big.Int).SetString(v, 10)
			if !ok {
				return ErrStrToBigInt
			}
			ids, e := types.HexToTokenTypeId(k)
			if e != nil {
				return e
			}
			rawfilter[ids] = *b
		}
	}

	var realDifficulty *big.Int = nil
	if powDifficulty != nil {
		b, ok := new(big.Int).SetString(*powDifficulty, 10)
		if !ok {
			return ErrStrToBigInt
		}
		realDifficulty = b
	}

	return o.manager.StartAutoReceiveWorker(entropystore, addr, rawfilter, realDifficulty)
}

func (o PrivateOnroadApi) StopAutoReceive(addr types.Address) error {
	log.Info("StopAutoReceive", "addr", addr)
	return o.manager.StopAutoReceiveWorker(addr)
}

func (o PrivateOnroadApi) GetAccountOnroadInfo(address types.Address) (*RpcAccountInfo, error) {
	log.Info("GetAccountOnroadInfo", "addr", address)
	info, e := o.manager.GetOnroadBlocksPool().GetOnroadAccountInfo(address)
	if e != nil || info == nil {
		return nil, e
	}

	r := onroadInfoToRpcAccountInfo(o.manager.Chain(), *info)
	return r, nil

}

func onroadInfoToRpcAccountInfo(chain chain.Chain, onroadInfo model.OnroadAccountInfo) *RpcAccountInfo {
	onroadInfo.Mutex.RLock()
	defer onroadInfo.Mutex.RUnlock()

	var r RpcAccountInfo
	r.AccountAddress = *onroadInfo.AccountAddress
	r.TotalNumber = strconv.FormatUint(onroadInfo.TotalNumber, 10)
	r.TokenBalanceInfoMap = make(map[types.TokenTypeId]*RpcTokenBalanceInfo)

	for tti, v := range onroadInfo.TokenBalanceInfoMap {
		if v != nil {
			number := strconv.FormatUint(v.Number, 10)
			tinfo, _ := chain.GetTokenInfoById(&tti)
			b := &RpcTokenBalanceInfo{
				TokenInfo:   RawTokenInfoToRpc(tinfo, tti),
				TotalAmount: v.TotalAmount.String(),
				Number:      &number,
			}
			r.TokenBalanceInfoMap[tti] = b
		}
	}
	return &r
}
*/
