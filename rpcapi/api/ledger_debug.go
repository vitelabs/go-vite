package api

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/vite"
)

type LedgerDebugApi struct {
	unreceived *UnreceivedDebugApi
}

func NewLedgerDebugApi(vite *vite.Vite) *LedgerDebugApi {
	return &LedgerDebugApi{
		unreceived: NewUnreceivedDebugApi(vite),
	}
}
func (ld LedgerDebugApi) String() string {
	return "LedgerDebugApi"
}

type UnreceivedDebugApi struct {
	manager *onroad.Manager
	chain   chain.Chain
}

func NewUnreceivedDebugApi(vite *vite.Vite) *UnreceivedDebugApi {
	return &UnreceivedDebugApi{
		manager: vite.OnRoad(),
		chain:   vite.Chain(),
	}
}

// private: unreceived_getContractUnreceivedTransactionCount <- onroad_getContractOnRoadTotalNum
func (ud *UnreceivedDebugApi) GetContractUnreceivedTransactionCount(addr types.Address, gid *types.Gid) (uint64, error) {
	if !types.IsContractAddr(addr) {
		return 0, errors.New("Address must be the type of Contract.")
	}

	var g types.Gid
	if gid == nil {
		g = types.DELEGATE_GID
	} else {
		g = *gid
	}

	num, err := ud.manager.GetOnRoadTotalNumByAddr(g, addr)
	if err != nil {
		return 0, err
	}
	log.Info("GetContractUnreceivedTransactionCount", "gid", gid, "addr", addr, "num", num)
	return num, nil
}

// private: unreceived_getContractUnreceivedFrontBlocks <- onorad_getContractOnRoadFrontBlocks
func (pu *UnreceivedDebugApi) GetContractUnreceivedFrontBlocks(addr types.Address, gid *types.Gid) ([]*AccountBlock, error) {
	if !types.IsContractAddr(addr) {
		return nil, errors.New("Address must be the type of Contract.")
	}
	var g types.Gid
	if gid == nil {
		g = types.DELEGATE_GID
	} else {
		g = *gid
	}

	blockList, err := pu.manager.GetAllCallersFrontOnRoad(g, addr)
	if err != nil {
		return nil, err
	}
	log.Info("GetContractUnreceivedFrontBlocks", "gid", gid, "addr", addr, "len", len(blockList))
	rpcBlockList := make([]*AccountBlock, len(blockList))
	sum := 0
	for _, v := range blockList {
		if v != nil {
			accountBlock, e := ledgerToRpcBlock(pu.chain, v)
			if e != nil {
				return nil, e
			}
			rpcBlockList[sum] = accountBlock
			sum++
		}
	}
	return rpcBlockList[:sum], nil
}
