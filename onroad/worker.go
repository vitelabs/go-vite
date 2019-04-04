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
	ContractTaskProcessorSize = 2 * POMAXPROCS
)
