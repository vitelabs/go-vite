package onroad

const (
	create = iota
	start
	stop
)

// Worker lists the methods that the "Worker" need to be implemented.
type Worker interface {
	Status() int
	Start()
	Stop()
	Close() error
}

var (
	// POMAXPROCS is used to limit the use of number of operating system threads.
	POMAXPROCS = 2
	// ContractTaskProcessorSize is used to limit the number of processors.
	ContractTaskProcessorSize = POMAXPROCS
)

type inferiorState int

const (
	// RETRY represents a state which the processor can retry to handle the onroad from a particular caller
	// to a particular contract in the next second during a block period.
	RETRY inferiorState = iota
	// OUT represents a state which the processor won't handle the onroad from a particular caller
	// to a particular contract during a block period any more.
	OUT
)
