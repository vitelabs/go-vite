package pool

import "runtime"

var (
	GOMAXPROCS       = runtime.NumCPU()
	ACCOUNT_PARALLEL = GOMAXPROCS
)
