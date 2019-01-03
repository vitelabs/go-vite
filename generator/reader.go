package generator

import (
	"github.com/vitelabs/go-vite/chain"
	"github.com/vitelabs/go-vite/vm_context"
)

type Chain interface {
	vm_context.Chain
	chain.Fork
}
