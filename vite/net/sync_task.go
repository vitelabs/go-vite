package net

import (
	"sync"

	"github.com/vitelabs/go-vite/ledger"
)

type syncTaskState int

const (
	syncTaskWait syncTaskState = iota
	syncTaskPending
	syncTaskDone
	syncTaskError
)

type syncTaskType int

const (
	syncFileTask syncTaskType = iota
	syncChunkTask
)

type syncTask interface {
	bound() (from, to uint64)
	state() syncTaskState
	setState(st syncTaskState)
	do() error
	taskType() syncTaskType
	info() string
}

type blockReceiver interface {
	receiveAccountBlock(block *ledger.AccountBlock) error
	receiveSnapshotBlock(block *ledger.SnapshotBlock) error
}

type File = *ledger.CompressedFileMeta
type Files []File

func (f Files) Len() int {
	return len(f)
}

func (f Files) Less(i, j int) bool {
	return f[i].StartHeight < f[j].StartHeight
}

func (f Files) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type fileDownloader interface {
	download(file File) error
}

type fileTask struct {
	st         syncTaskState
	file       File
	downloader fileDownloader
}

func (f *fileTask) info() string {
	panic("implement me")
}

func (f *fileTask) state() syncTaskState {
	return f.st
}

func (f *fileTask) setState(st syncTaskState) {
	f.st = st
}

func (f *fileTask) taskType() syncTaskType {
	return syncFileTask
}

func (f *fileTask) bound() (from, to uint64) {
	return f.file.StartHeight, f.file.EndHeight
}

func (f *fileTask) do() error {
	return f.downloader.download(f.file)
}

type chunkDownloader interface {
	download(from, to uint64) error
}

type chunkTask struct {
	from, to   uint64
	st         syncTaskState
	downloader chunkDownloader
}

func (c *chunkTask) info() string {
	panic("implement me")
}

func (c *chunkTask) state() syncTaskState {
	return c.st
}

func (c *chunkTask) setState(st syncTaskState) {
	c.st = st
}

func (c *chunkTask) taskType() syncTaskType {
	return syncChunkTask
}

func (c *chunkTask) bound() (from, to uint64) {
	return c.from, c.to
}

func (c *chunkTask) do() error {
	return c.downloader.download(c.from, c.to)
}

type syncTaskExecutor interface {
	add(t syncTask)
	runTo(to uint64)
	start()
	stop()
}

type syncTaskListener interface {
	done(t syncTask)
	cache(t syncTask, err error)
}

type executor struct {
	mu    sync.Mutex
	tasks []syncTask

	doneIndex      int
	setChainTarget func(to uint64) // when continuous tasks have done, then chain should grow to the specified height.
	listener       syncTaskListener
}

func (e *executor) add(task syncTask) {
	// todo
	//e.mu.Lock()
	//defer e.mu.Unlock()
	//
	//if len(e.tasks) == 0 {
	//	e.tasks = append(e.tasks, task)
	//	return
	//}
	//
	//from, to := task.bound()
	//
	//last := e.tasks[len(e.tasks)-1]
	//f, to := last.bound()
	//
	//for i := 0; i < len(e.tasks); i++ {
	//	f, t := e.tasks[i].bound()
	//	if from > t {
	//		continue
	//	} else if f == from {
	//		return
	//	} else {
	//
	//	}
	//}
}

func (e *executor) runTo(to uint64) {
	var jump = 0
	var index = 0
	var continuous = true // is task done continuously

	for index = e.doneIndex + jump; index < len(e.tasks); index = e.doneIndex + jump {
		t := e.tasks[index]
		st := t.state()

		if st == syncTaskDone && continuous {
			e.doneIndex++
		} else if st == syncTaskPending || st == syncTaskDone {
			continuous = false
			jump++
		} else {
			continuous = false
			if from, _ := t.bound(); from <= to {
				e.run(t)
			} else {
				break
			}
		}
	}

	// chain should grow to the target height
	_, to = e.tasks[e.doneIndex].bound()
	e.setChainTarget(to)
}

func (e *executor) run(t syncTask) {
	t.setState(syncTaskPending)
	go e.do(t)
}

func (e *executor) do(t syncTask) {
	if err := t.do(); err != nil {
		t.setState(syncTaskError)
		e.listener.cache(t, err)
	} else {
		t.setState(syncTaskDone)
		e.listener.done(t)
	}
}

func (e *executor) start() {
	panic("implement me")
}

func (e *executor) stop() {
	panic("implement me")
}
