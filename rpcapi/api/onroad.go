package api

import (
	"github.com/vitelabs/go-vite/v2"
	"github.com/vitelabs/go-vite/v2/common/types"
)

type PublicOnroadApi struct {
	api *PrivateOnroadApi
}

func NewPublicOnroadApi(vite *vite.Vite) *PublicOnroadApi {
	return &PublicOnroadApi{
		api: NewPrivateOnroadApi(vite)}
}

func (pub PublicOnroadApi) String() string {
	return "PublicOnroadApi"
}

type PrivateOnroadApi struct {
	ledgerApi *LedgerApi
}

func NewPrivateOnroadApi(vite *vite.Vite) *PrivateOnroadApi {
	return &PrivateOnroadApi{
		ledgerApi: NewLedgerApi(vite),
	}
}

func (pri PrivateOnroadApi) String() string {
	return "PrivateOnroadApi"
}

type OnroadPagingQuery struct {
	Addr types.Address `json:"addr"`

	PageNum   uint64 `json:"pageNum"`
	PageCount uint64 `json:"pageCount"`
}

// ------------------------------------------------------------------------------------------------------------------------
// ---------------------deprecated-----------------------------------------------------------------------------------------
// ------------------------------------------------------------------------------------------------------------------------

// Deprecated: to use ledger_getUnreceivedBlocksByAddress instead
func (pri PrivateOnroadApi) GetOnroadBlocksByAddress(address types.Address, index, count uint64) ([]*AccountBlock, error) {
	return pri.ledgerApi.GetUnreceivedBlocksByAddress(address, index, count)
}

// Deprecated: to use ledger_getUnreceivedTransactionSummaryByAddress instead
func (pri PrivateOnroadApi) GetOnroadInfoByAddress(address types.Address) (*RpcAccountInfo, error) {
	log.Info("GetUnreceivedTransactionSummaryByAddress", "addr", address)

	info, e := pri.ledgerApi.chain.GetAccountOnRoadInfo(address)
	if e != nil || info == nil {
		return nil, e
	}
	return ToRpcAccountInfo(pri.ledgerApi.chain, info), nil
}

// Deprecated: to use unreceived_getUnreceivedBlocksInBatch instead
func (pri PrivateOnroadApi) GetOnroadBlocksInBatch(queryList []OnroadPagingQuery) (map[types.Address][]*AccountBlock, error) {
	querys := make([]PagingQueryBatch, 0)
	for _, v := range queryList {
		querys = append(querys, PagingQueryBatch{
			Address:    v.Addr,
			PageNumber: v.PageNum,
			PageCount:  v.PageCount,
		})
	}
	return pri.ledgerApi.GetUnreceivedBlocksInBatch(querys)
}

// Deprecated: to use unreceived_getUnreceivedTransactionSummaryInBatch instead
func (pri PrivateOnroadApi) GetOnroadInfoInBatch(addrList []types.Address) ([]*RpcAccountInfo, error) {
	// Remove duplicate
	addrMap := make(map[types.Address]bool, 0)
	for _, v := range addrList {
		addrMap[v] = true
	}

	resultList := make([]*RpcAccountInfo, 0)
	for addr, _ := range addrMap {
		info, err := pri.ledgerApi.chain.GetAccountOnRoadInfo(addr)
		if err != nil {
			return nil, err
		}
		if info == nil {
			continue
		}
		resultList = append(resultList, ToRpcAccountInfo(pri.ledgerApi.chain, info))
	}
	return resultList, nil
}
