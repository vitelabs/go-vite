package net

import (
	"context"
	"strconv"
	"sync"

	"github.com/vitelabs/go-vite/ledger"
)

const SYNC_TASK_BATCH = 1000

type syncTaskType byte

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

var syncTaskPool sync.Pool

func backSyncTask(t *syncTask) {
	t.task = nil
	t.st = reqWaiting
	t.ctx = nil
	t.cancel = nil
	syncTaskPool.Put(t)
}

type syncTask struct {
	task
	st     reqState
	typ    syncTaskType
	cancel func()
	ctx    context.Context
}

func newSyncTask() *syncTask {
	v := syncTaskPool.Get()
	if v == nil {
		return &syncTask{}
	}
	return v.(*syncTask)
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
	end() (uint64, syncTaskType)
	add(t *syncTask)
	exec(t *syncTask)
	runTo(to uint64)
	deleteFrom(start uint64) (nextStart uint64)
	last() *syncTask
	terminate()
	status() ExecutorStatus
	clean() int // remove tasks done
}

type syncTaskListener interface {
	taskDone(t *syncTask, err error) // one task done
	allTaskDone(last *syncTask)      // all task done
}

type executor struct {
	mu sync.Mutex

	tasks     []*syncTask
	doneIndex int

	listener syncTaskListener

	ctx       context.Context
	ctxCancel func()
}

type ExecutorStatus struct {
	Tasks []string
}

func newExecutor(listener syncTaskListener) syncTaskExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	return &executor{
		tasks:     make([]*syncTask, 0, SYNC_TASK_BATCH),
		listener:  listener,
		ctx:       ctx,
		ctxCancel: cancel,
	}
}

func (e *executor) end() (uint64, syncTaskType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.tasks) > 0 {
		t := e.tasks[len(e.tasks)-1]
		_, to := t.bound()
		return to, t.typ
	}

	return 0, 0
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

	var last *syncTask
	if len(e.tasks) > 0 {
		last = e.tasks[len(e.tasks)-1]
	}
	e.mu.Unlock()

	// no tasks remand
	if last != nil {
		e.listener.allTaskDone(last)
	}
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

// delete tasks those start is larger than start
// if add chunk task first, and get fileList later, then can delete chunk tasks and add file tasks
func (e *executor) deleteFrom(start uint64) (nextStart uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := e.doneIndex; i < len(e.tasks); i++ {
		t := e.tasks[i]
		if t.st == reqDone {
			continue
		}

		tStart, _ := t.bound()
		if tStart >= start {
			for j := i; j < len(e.tasks); j++ {
				t = e.tasks[j]
				// cancel tasks
				if t.cancel != nil {
					t.cancel()
				}

				// put back to pool
				backSyncTask(t)
			}

			e.tasks = e.tasks[:i]
			break
		}
	}

	if len(e.tasks) > 0 {
		last := e.tasks[len(e.tasks)-1]
		_, to := last.bound()
		return to + 1
	}

	return 0
}

func (e *executor) clean() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	j := 0
	for i := 0; i < len(e.tasks); i++ {
		t := e.tasks[i]
		if t.st != reqDone {
			e.tasks[j] = e.tasks[i]
			j++
		} else {
			backSyncTask(t)
			e.tasks[i] = nil
		}
	}

	e.tasks = e.tasks[:j]
	e.doneIndex = 0

	return j
}

func (e *executor) terminate() {
	if e.ctx != nil {
		e.ctxCancel()
	}

	e.mu.Lock()
	e.ctx, e.ctxCancel = context.WithCancel(context.Background())
	e.tasks = nil
	e.doneIndex = 0
	e.mu.Unlock()
}
