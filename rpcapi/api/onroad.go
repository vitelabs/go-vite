package api

import (
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
