package worker

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
