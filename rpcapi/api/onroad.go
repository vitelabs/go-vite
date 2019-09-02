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

// Deprecated: to use ledger_getUnreceivedBlocksByAddress instead
func (pri PrivateOnroadApi) GetOnroadBlocksByAddress(address types.Address, index, count uint64) ([]*AccountBlock, error) {
	return pri.ledgerApi.GetUnreceivedBlocksByAddress(address, index, count)
}

// Deprecated: to use ledger_getUnreceivedTransactionSummaryByAddress instead
func (pri PrivateOnroadApi) GetOnroadInfoByAddress(address types.Address) (*AccountInfo, error) {
	return pri.ledgerApi.GetUnreceivedTransactionSummaryByAddress(address)
}

type OnroadPagingQuery struct {
	Addr types.Address `json:"addr"`

	PageNum   uint64 `json:"pageNum"`
	PageCount uint64 `json:"pageCount"`
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
func (pri PrivateOnroadApi) GetOnroadInfoInBatch(addrList []types.Address) ([]*AccountInfo, error) {
	return pri.ledgerApi.GetUnreceivedTransactionSummaryInBatch(addrList)
}
