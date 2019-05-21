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
