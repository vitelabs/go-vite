package net

import (
	"context"
	"strconv"
	"sync"

	"github.com/vitelabs/go-vite/ledger"
)

type syncTaskType int

const (
	syncFileTask syncTaskType = iota
	syncChunkTask
)

type reqState byte

const (
	reqWaiting reqState = iota
	reqPending
	reqRespond
	reqDone
	reqError
	reqCancel
)

var reqStatus = [...]string{
	reqWaiting: "waiting",
	reqPending: "pending",
	reqRespond: "respond",
	reqDone:    "done",
	reqError:   "error",
	reqCancel:  "canceled",
}

func (s reqState) String() string {
	if s > reqCancel {
		return "unknown request state"
	}
	return reqStatus[s]
}

type syncTask struct {
	task
	st     reqState
	typ    syncTaskType
	cancel func()
	ctx    context.Context
}

func (t *syncTask) do() error {
	return t.task.do(t.ctx)
}

func (t *syncTask) String() string {
	return t.task.String() + " " + t.st.String()
}

type task interface {
	bound() (from, to uint64)
	do(ctx context.Context) error
	String() string
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
	download(ctx context.Context, file File) <-chan error
}

type fileTask struct {
	file       File
	downloader fileDownloader
}

func (f *fileTask) String() string {
	return f.file.Filename
}

func (f *fileTask) bound() (from, to uint64) {
	return f.file.StartHeight, f.file.EndHeight
}

func (f *fileTask) do(ctx context.Context) error {
	return <-f.downloader.download(ctx, f.file)
}

type chunkDownloader interface {
	// only return error when no suitable peers to request the subledger
	download(ctx context.Context, from, to uint64) <-chan error
}

type chunkTask struct {
	from, to   uint64
	downloader chunkDownloader
}

func (c *chunkTask) String() string {
	return "chunk<" + strconv.FormatUint(c.from, 10) + "-" + strconv.FormatUint(c.to, 10) + ">"
}

func (c *chunkTask) bound() (from, to uint64) {
	return c.from, c.to
}

func (c *chunkTask) do(ctx context.Context) error {
	return <-c.downloader.download(ctx, c.from, c.to)
}

type syncTaskExecutor interface {
	add(t *syncTask)
	exec(t *syncTask)
	runTo(to uint64)
	last() *syncTask
	terminate()
	status() ExecutorStatus
}

type syncTaskListener interface {
	taskDone(t *syncTask, err error) // one task done
	allTaskDone(last *syncTask)      // all task done
}

type executor struct {
	mu    sync.Mutex
	tasks []*syncTask

	doneIndex int
	listener  syncTaskListener

	ctx       context.Context
	ctxCancel func()
}

type ExecutorStatus struct {
	Tasks []string
}

func newExecutor(listener syncTaskListener) syncTaskExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	return &executor{
		listener:  listener,
		ctx:       ctx,
		ctxCancel: cancel,
	}
}

func (e *executor) status() ExecutorStatus {
	e.mu.Lock()
	defer e.mu.Unlock()

	tasks := make([]string, len(e.tasks))
	i := 0
	for _, t := range e.tasks {
		tasks[i] = t.String()
		i++
	}

	return ExecutorStatus{
		tasks,
	}
}

// add from low to high
func (e *executor) add(t *syncTask) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tasks = append(e.tasks, t)
}

func (e *executor) runTo(to uint64) {
	e.mu.Lock()

	var skip = 0 // pending / cancel / done but not continue
	var index = 0
	var continuous = true // is task done consecutively

	for index = e.doneIndex + skip; index < len(e.tasks); index = e.doneIndex + skip {
		t := e.tasks[index]

		if t.st == reqDone && continuous {
			e.doneIndex++
		} else if t.st == reqPending || t.st == reqDone {
			continuous = false
			skip++
		} else {
			continuous = false
			if from, _ := t.bound(); from <= to {
				e.run(t)
			} else {
				e.mu.Unlock()
				return
			}
		}
	}

	last := e.tasks[index-1]
	e.mu.Unlock()

	// no tasks remand
	e.listener.allTaskDone(last)
}

func (e *executor) run(t *syncTask) {
	t.st = reqPending
	go e.exec(t)
}

func (e *executor) exec(t *syncTask) {
	t.ctx, t.cancel = context.WithCancel(e.ctx)
	if err := t.do(); err != nil {
		t.st = reqError
		e.listener.taskDone(t, err)
	} else {
		t.st = reqDone
		e.listener.taskDone(t, nil)
	}
}

func (e *executor) last() *syncTask {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.tasks[len(e.tasks)-1]
}

func (e *executor) terminate() {
	if e.ctx != nil {
		e.ctxCancel()
	}
}
