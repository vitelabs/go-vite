package onroad

import (
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
	CommonFetchSize           = 4 * POMAXPROCS
	ContractTaskProcessorSize = 2 * POMAXPROCS
	ContractFetchSize         = 2 * POMAXPROCS
)
