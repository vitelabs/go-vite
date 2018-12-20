package chain_cache

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
)

type Chain interface {
	GetUnConfirmedSubLedger() (map[types.Address][]*ledger.AccountBlock, error)
	GetUnConfirmedPartSubLedger(addrList []types.Address) (map[types.Address][]*ledger.AccountBlock, error)
}
