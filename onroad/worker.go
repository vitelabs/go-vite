package onroad

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
	POMAXPROCS                = 2
	ContractTaskProcessorSize = POMAXPROCS
)
