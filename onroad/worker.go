package onroad

import (
	"github.com/vitelabs/go-vite/common/types"
	"runtime"
)

const (
	Create = iota
	Start
	Stop
)

type Worker interface {
	Status() int
	Start()
	Stop()
	Close() error
}

var (
	POMAXPROCS                = runtime.NumCPU()
	ContractTaskProcessorSize = 2 * POMAXPROCS
)

type subBlackList map[types.Address]bool
