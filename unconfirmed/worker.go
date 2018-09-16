package unconfirmed

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vm_context"
	"runtime"
)

const (
	Create = iota
	Start
	Stop
	Closed
)

type Worker interface {
	Status() int
	Start()
	Stop()
	Close() error
}

var (
	POMAXPROCS          = uint64(runtime.NumCPU())
	COMMON_FETCH_SIZE   = uint64(4 * POMAXPROCS)
	CONTRACT_TASK_SIZE  = uint64(2 * POMAXPROCS)
	CONTRACT_FETCH_SIZE = uint64(2 * POMAXPROCS)
)

type PoolAccess interface {
	AddBlock(block *ledger.AccountBlock) error
	AddDirectBlock(block *ledger.AccountBlock, blockGen *vm_context.VmAccountBlock) error
	ExistInPool(address *types.Address, fromBlockHash *types.Hash) bool
}
