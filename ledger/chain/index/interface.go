package chain_index

import (
	"github.com/vitelabs/go-vite/v2/common/types"
	ledger "github.com/vitelabs/go-vite/v2/interfaces/core"
)

type Chain interface {
	IterateContracts(iterateFunc func(addr types.Address, meta *ledger.ContractMeta, err error) bool)
}
