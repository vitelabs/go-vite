package api

import (
	"fmt"
	"github.com/go-errors/errors"
	"github.com/vitelabs/go-vite/common/math"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vite"
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

// Deprecated: to use ledger_getUnreceivedBlocksByAddress instead
func (pub PublicOnroadApi) GetOnroadBlocksByAddress(address types.Address, index, count uint64) ([]*AccountBlock, error) {
	if count > math.MaxUint16+1 {
		return nil, errors.New(fmt.Sprintf("maximum number per page allowed is %d", math.MaxUint16+1))
	}
	return pub.api.GetOnroadBlocksByAddress(address, index, count)
}

// Deprecated: to use ledger_getUnreceivedTransactionSummaryByAddress instead
func (pub PublicOnroadApi) GetOnroadInfoByAddress(address types.Address) (*AccountInfo, error) {
	return pub.api.GetOnroadInfoByAddress(address)
}

type PrivateOnroadApi struct {
	unreceivedApi *UnreceivedApi
}

func NewPrivateOnroadApi(vite *vite.Vite) *PrivateOnroadApi {
	return &PrivateOnroadApi{
		unreceivedApi: NewUnreceivedApi(vite),
	}
}

func (pri PrivateOnroadApi) String() string {
	return "PrivateOnroadApi"
}

// Deprecated: to use ledger_getUnreceivedBlocksByAddress instead
func (pri PrivateOnroadApi) GetOnroadBlocksByAddress(address types.Address, index, count uint64) ([]*AccountBlock, error) {
	return pri.unreceivedApi.ledgerApi.GetUnreceivedBlocksByAddress(address, index, count)
}

// Deprecated: to use ledger_getUnreceivedTransactionSummaryByAddress instead
func (pri PrivateOnroadApi) GetOnroadInfoByAddress(address types.Address) (*AccountInfo, error) {
	return pri.unreceivedApi.ledgerApi.GetUnreceivedTransactionSummaryByAddress(address)
}

type OnroadPagingQuery struct {
	Addr types.Address `json:"addr"`

	PageNum   uint64 `json:"pageNum"`
	PageCount uint64 `json:"pageCount"`
}

// Deprecated: to use unreceived_getUnreceivedBlocksInBatch instead
func (pri PrivateOnroadApi) GetOnroadBlocksInBatch(queryList []OnroadPagingQuery) (map[types.Address][]*AccountBlock, error) {
	querys := make([]UnreceivedPagingQuery, 0)
	for _, v := range queryList {
		querys = append(querys, UnreceivedPagingQuery{
			Address:    v.Addr,
			PageNumber: v.PageNum,
			PageCount:  v.PageCount,
		})
	}
	return pri.unreceivedApi.GetUnreceivedBlocksInBatch(querys)
}

// Deprecated: to use unreceived_getUnreceivedTransactionSummaryInBatch instead
func (pri PrivateOnroadApi) GetOnroadInfoInBatch(addrList []types.Address) ([]*AccountInfo, error) {
	return pri.unreceivedApi.GetUnreceivedTransactionSummaryInBatch(addrList)
}

// Deprecated: to use unreceived_getContractUnreceivedTransactionCount instead
func (pri PrivateOnroadApi) GetContractOnRoadTotalNum(addr types.Address, gid *types.Gid) (uint64, error) {
	return pri.unreceivedApi.GetContractUnreceivedTransactionCount(addr, gid)
}

// Deprecated: to use unreceived_getContractUnreceivedFrontBlocks instead
func (pri PrivateOnroadApi) GetContractOnRoadFrontBlocks(addr types.Address, gid *types.Gid) ([]*AccountBlock, error) {
	return pri.unreceivedApi.GetContractUnreceivedFrontBlocks(addr, gid)
}
