package api

import (
	"github.com/vitelabs/go-vite/common/db/xleveldb/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/onroad"
	"github.com/vitelabs/go-vite/vite"
)

type UnreceivedApi struct {
	manager   *onroad.Manager
	ledgerApi *LedgerApi
}

func NewUnreceivedApi(vite *vite.Vite) *UnreceivedApi {
	return &UnreceivedApi{
		manager:   vite.OnRoad(),
		ledgerApi: NewLedgerApi(vite),
	}
}

func (pu UnreceivedApi) String() string {
	return "UnreceivedApi"
}

type UnreceivedPagingQuery struct {
	Address types.Address `json:"address"`

	PageNumber uint64 `json:"pageNumber"`
	PageCount  uint64 `json:"pageCount"`
}

// private: unreceived_getUnreceivedBlocksInBatch <- onroad_getOnroadBlocksInBatch
func (pu *UnreceivedApi) GetUnreceivedBlocksInBatch(queryList []UnreceivedPagingQuery) (map[types.Address][]*AccountBlock, error) {
	resultMap := make(map[types.Address][]*AccountBlock)
	for _, q := range queryList {
		if l, ok := resultMap[q.Address]; ok && l != nil {
			continue
		}
		blockList, err := pu.ledgerApi.GetUnreceivedBlocksByAddress(q.Address, q.PageNumber, q.PageCount)
		if err != nil {
			return nil, err
		}
		if len(blockList) <= 0 {
			continue
		}
		resultMap[q.Address] = blockList
	}
	return resultMap, nil
}

// private: unreceived_getUnreceivedTransactionSummaryInBatch <-  onroad_getOnroadInfoInBatch
func (pu *UnreceivedApi) GetUnreceivedTransactionSummaryInBatch(addressList []types.Address) ([]*AccountInfo, error) {

	// Remove duplicate
	addrMap := make(map[types.Address]bool, 0)
	for _, v := range addressList {
		addrMap[v] = true
	}

	resultList := make([]*AccountInfo, 0)
	for addr, _ := range addrMap {
		info, err := pu.ledgerApi.GetUnreceivedTransactionSummaryByAddress(addr)
		if err != nil {
			return nil, err
		}
		if info == nil {
			continue
		}
		resultList = append(resultList, info)
	}
	return resultList, nil
}

// private: unreceived_getContractUnreceivedTransactionCount <- onroad_getContractOnRoadTotalNum
func (pu *UnreceivedApi) GetContractUnreceivedTransactionCount(addr types.Address, gid *types.Gid) (uint64, error) {
	if !types.IsContractAddr(addr) {
		return 0, errors.New("Address must be the type of Contract.")
	}

	var g types.Gid
	if gid == nil {
		g = types.DELEGATE_GID
	} else {
		g = *gid
	}

	num, err := pu.manager.GetOnRoadTotalNumByAddr(g, addr)
	if err != nil {
		return 0, err
	}
	log.Info("GetContractUnreceivedTransactionCount", "gid", gid, "addr", addr, "num", num)
	return num, nil
}

// private: unreceived_getContractUnreceivedFrontBlocks <- onorad_getContractOnRoadFrontBlocks
func (pu *UnreceivedApi) GetContractUnreceivedFrontBlocks(addr types.Address, gid *types.Gid) ([]*AccountBlock, error) {
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
			accountBlock, e := ledgerToRpcBlock(pu.ledgerApi.chain, v)
			if e != nil {
				return nil, e
			}
			rpcBlockList[sum] = accountBlock
			sum++
		}
	}
	return rpcBlockList[:sum], nil
}
