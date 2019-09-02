//

package rpc

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/rpc"
	"github.com/vitelabs/go-vite/rpcapi/api"
)

// OnroadApi ...
type OnroadApi interface {
	GetOnroadBlocksByAddress(address types.Address, index, count uint64) ([]*api.AccountBlock, error)
	GetOnroadInfoByAddress(address types.Address) (*api.RpcAccountInfo, error)
	GetOnroadBlocksInBatch(queryList []api.OnroadPagingQuery) (map[types.Address][]*api.AccountBlock, error)
	GetOnroadInfoInBatch(addrList []types.Address) ([]*api.RpcAccountInfo, error)
	GetContractOnRoadFrontBlocks(addr types.Address, gid *types.Gid) ([]*api.AccountBlock, error)
}

type onroadApi struct {
	cc *rpc.Client
}

func NewOnroadApi(cc *rpc.Client) OnroadApi {
	return &onroadApi{cc: cc}
}

func (oi onroadApi) GetOnroadBlocksByAddress(address types.Address, index, count uint64) (blocks []*api.AccountBlock, err error) {
	err = oi.cc.Call(&blocks, "onroad_getOnroadBlocksByAddress", address, index, count)
	return
}

func (oi onroadApi) GetOnroadInfoByAddress(address types.Address) (info *api.RpcAccountInfo, err error) {
	info = &api.RpcAccountInfo{}
	err = oi.cc.Call(info, "onroad_getOnroadInfoByAddress", address)
	return
}

func (oi onroadApi) GetOnroadBlocksInBatch(queryList []api.OnroadPagingQuery) (result map[types.Address][]*api.AccountBlock, err error) {
	result = make(map[types.Address][]*api.AccountBlock)
	err = oi.cc.Call(&result, "onroad_GetOnroadBlocksInBatch", queryList)
	return
}

func (oi onroadApi) GetOnroadInfoInBatch(addrList []types.Address) (infos []*api.RpcAccountInfo, err error) {
	err = oi.cc.Call(&infos, "onroad_getOnroadInfoInBatch", addrList)
	return
}

func (oi onroadApi) GetContractOnRoadFrontBlocks(addr types.Address, gid *types.Gid) (blocks []*api.AccountBlock, err error) {
	err = oi.cc.Call(&blocks, "onroad_getContractOnRoadFrontBlocks", addr, gid)
	return
}
