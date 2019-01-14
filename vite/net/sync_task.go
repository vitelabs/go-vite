package net

type syncTaskState int

const (
	syncTaskWait syncTaskState = iota
	syncTaskPending
	syncTaskDone
	syncTaskError
)

type syncTask interface {
	bound() (from, to uint64)
	state() syncTaskState
	setState(st syncTaskState)
	do() error
}

type fileTask struct {
}

func (f *fileTask) bound() (from, to uint64) {
	panic("implement me")
}

func (f *fileTask) do() error {
	panic("implement me")
}

type chunkTask struct {
}

func (c *chunkTask) bound() (from, to uint64) {
	panic("implement me")
}

func (c *chunkTask) do() error {
	panic("implement me")
}

type syncTaskExecutor interface {
	add(t syncTask)
	runTo(to uint64)
	start()
	stop()
}

type executor struct {
	tasks          []syncTask
	doneIndex      int
	setChainTarget func(to uint64) // when continuous tasks have done, then chain should grow to the specified height.
}

func (e *executor) add(t syncTask) {
	panic("implement me")
}

func (e *executor) runTo(to uint64) {
	var pending = 0
	var index = 0
	var growTarget uint64

	for index = e.doneIndex + pending; index < len(e.tasks); index = e.doneIndex + pending {
		t := e.tasks[index]
		st := t.state()

		if st ==

		if ; st == syncTaskPending {
			pending++
		} else if st == syncTaskDone {
			e.doneIndex++
		} else {
			from, _ := t.bound()
			if from <= to {
				e.run(t)
			} else {
				break
			}
		}
	}

	e.setChainTarget()
}

func (e *executor) run(t syncTask) {

}

func (e *executor) start() {
	panic("implement me")
}

func (e *executor) stop() {
	panic("implement me")
}
