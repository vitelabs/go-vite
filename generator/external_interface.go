package generator

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vm_context"
)

type Chain interface {
	vm_context.Chain
	AccountType(address *types.Address) (uint64, error)
}
