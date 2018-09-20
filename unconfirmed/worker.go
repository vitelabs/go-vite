package unconfirmed

import (
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
	POMAXPROCS          = runtime.NumCPU()
	COMMON_FETCH_SIZE   = 4 * POMAXPROCS
	CONTRACT_TASK_SIZE  = 2 * POMAXPROCS
	CONTRACT_FETCH_SIZE = 2 * POMAXPROCS
)
