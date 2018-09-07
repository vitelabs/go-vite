package worker

import "runtime"

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
