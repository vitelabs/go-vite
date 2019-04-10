package net

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type reqState byte

const (
	reqWaiting reqState = iota
	reqPending
	reqDone
	reqError
	reqCancel
)

var reqStatus = [...]string{
	reqWaiting: "waiting",
	reqPending: "db",
	reqDone:    "done",
	reqError:   "error",
	reqCancel:  "canceled",
}

func (s reqState) String() string {
	if s > reqCancel {
		return "unknown request state_bak"
	}
	return reqStatus[s]
}

type syncTask struct {
	task
	st     reqState
	doneAt time.Time
	cancel func()
	ctx    context.Context
}

func (t syncTask) do() error {
	err := t.task.do(t.ctx)
	if err == nil {
		t.doneAt = time.Now()
	}

	return err
}

func (t syncTask) String() string {
	return t.task.String() + " " + t.st.String()
}

type task interface {
	bound() (from, to uint64)
	do(ctx context.Context) error
	String() string
}

type chunkDownloader interface {
	// only return error when no suitable peers to request the subledger
	download(ctx context.Context, from, to uint64) <-chan error
	start()
	stop()
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

type syncTasks []syncTask

func (s syncTasks) Len() int {
	return len(s)
}

func (s syncTasks) Less(i, j int) bool {
	fi, _ := s[i].bound()
	fj, _ := s[j].bound()
	return fi < fj
}

func (s syncTasks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type syncTaskExecutor interface {
	start()
	stop()
	// add a task into queue, task will be sorted from small to large.
	// if task is overlapped, then the task will be split to small tasks.
	add(t syncTask)
	// execute a task directly, NOT put into queue.
	// if a previous same from-to task has done, the gap time must longer than 1 min, or the task will not exec.
	exec(t syncTask)
	// runTo will execute those from is small than to tasks
	runTo(to uint64)
	// deleteFrom will delete those from is large than start tasks.
	// return (the last task`s to + 1)
	deleteFrom(start uint64) (nextStart uint64)
	// last return the last task in queue
	// ok is false if task queue is empty
	last() (t syncTask, ok bool)
	status() ExecutorStatus
	// clear all tasks
	reset()
	addListener(listener syncTaskListener)
}

type syncTaskListener interface {
	taskDone(t syncTask, err error) // one task done
	allTaskDone(last syncTask)
}

type executor struct {
	mu         sync.Mutex
	tasks      syncTasks
	max, batch int
	doneIndex  int
	listeners  []syncTaskListener
	ctx        context.Context
	ctxCancel  context.CancelFunc
	running    int32
	wg         sync.WaitGroup
}

type ExecutorStatus struct {
	Tasks []string
}

func newExecutor(max, batch int) syncTaskExecutor {
	return &executor{
		max:   max,
		batch: batch,
		tasks: make(syncTasks, 0, max),
	}
}

func (e *executor) start() {
	if atomic.CompareAndSwapInt32(&e.running, 0, 1) {
		e.ctx, e.ctxCancel = context.WithCancel(context.Background())

		e.tasks = e.tasks[:0]
		e.doneIndex = 0

		e.wg.Add(1)
		go e.loop()
	}
}
func (e *executor) stop() {
	if atomic.CompareAndSwapInt32(&e.running, 1, 0) {
		e.ctxCancel()
		e.wg.Wait()
	}
}

func (e *executor) addListener(listener syncTaskListener) {
	e.listeners = append(e.listeners, listener)
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
func (e *executor) add(t syncTask) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tasks = append(e.tasks, t)
}

func (e *executor) reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tasks = e.tasks[:0]
	e.doneIndex = 0
}

func (e *executor) loop() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		default:
			var skip = 0 // db / cancel / done but not continue
			var index = 0
			var batch = 0
			var continuous = true // is task done consecutively
			var t syncTask

			e.mu.Lock()
			for index = e.doneIndex; index < len(e.tasks); index = e.doneIndex + skip {
				t = e.tasks[index]

				// update downIndex if done tasks are continuous
				if t.st == reqDone && continuous {
					e.doneIndex++
				} else if t.st == reqPending {
					continuous = false
					skip++
					batch++
				} else if t.st == reqDone {
					continuous = false
					skip++
				} else if batch < e.batch {
					continuous = false
					skip++
					batch++
					e.run(t)
				} else {
					break
				}
			}

			var last bool
			if e.doneIndex > 0 && e.doneIndex == len(e.tasks)-1 {
				last = true
				t = e.tasks[len(e.tasks)-1]
			}
			e.mu.Unlock()

			if last {
				go e.notifyAllDone(t)
			}
		}
	}
}

func (e *executor) runTo(to uint64) {
	var skip = 0 // db / cancel / done but not continue
	var index = 0
	var continuous = true // is task done consecutively
	var t syncTask

	e.mu.Lock()
	for index = e.doneIndex + skip; index < len(e.tasks); index = e.doneIndex + skip {
		t = e.tasks[index]

		if t.st == reqDone && continuous {
			e.doneIndex++
		} else if t.st == reqPending || t.st == reqDone {
			continuous = false
			skip++
		} else {
			continuous = false
			skip++
			if from, _ := t.bound(); from <= to {
				e.run(t)
			} else {
				e.mu.Unlock()
				return
			}
		}
	}

	var last bool
	if e.doneIndex > 0 && e.doneIndex == len(e.tasks)-1 {
		last = true
		t = e.tasks[len(e.tasks)-1]
	}
	e.mu.Unlock()

	if last {
		go e.notifyAllDone(t)
	}
}

func (e *executor) run(t syncTask) {
	t.st = reqPending
	go e.do(t)
}

func (e *executor) exec(t syncTask) {
	_, to := t.bound()

	e.mu.Lock()
	if len(e.tasks) == 0 {
		e.run(t)
	} else {
		from, _ := e.tasks[0].bound()
		if from > to {
			e.run(t)
		}
	}
	e.mu.Unlock()
}

func (e *executor) do(t syncTask) {
	t.ctx, t.cancel = context.WithCancel(e.ctx)

	err := t.do()

	if err != nil {
		t.st = reqError
	} else {
		t.st = reqDone
	}

	go e.notify(t, err)
}

func (e *executor) notify(t syncTask, err error) {
	for _, listener := range e.listeners {
		listener.taskDone(t, err)
	}
}

func (e *executor) notifyAllDone(last syncTask) {
	for _, listener := range e.listeners {
		listener.allTaskDone(last)
	}
}

func (e *executor) last() (t syncTask, ok bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.tasks) == 0 {
		ok = false
		return
	}

	return e.tasks[len(e.tasks)-1], true
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

type syncBatchDownloader struct {
	from, to   uint64
	batchEnd   uint64
	executor   syncTaskExecutor
	downloader chunkDownloader
	observer   func(err error)
	syncing    int32
}

func (s *syncBatchDownloader) allTaskDone(last syncTask) {
	s.executor.reset()

	if s.batchEnd >= s.to {
		go s.observer(nil)
		return
	} else {
		s.allocateTasks()
	}
}

func (s *syncBatchDownloader) taskDone(t syncTask, err error) {
	if err != nil {
		_, to := t.bound()
		if to < atomic.LoadUint64(&s.to) {
			s.observer(err)
		}
	}
}

func newBatchDownloader(peers *peerSet, fact syncConnectionFactory) *syncBatchDownloader {
	pool := newPool(peers)

	var s = &syncBatchDownloader{
		executor:   newExecutor(batchSyncTask, 4),
		downloader: newDownloader(pool, fact, nil),
	}

	s.executor.addListener(s)

	return s
}

func (s *syncBatchDownloader) stop() {
	if atomic.CompareAndSwapInt32(&s.syncing, 1, 0) {
		s.executor.stop()
		s.downloader.stop()
	}
}

func (s *syncBatchDownloader) sync(from, to uint64) {
	if atomic.CompareAndSwapInt32(&s.syncing, 0, 1) {
		s.from, s.to = from, to
		s.batchEnd = s.from - 1

		s.allocateTasks()
		s.executor.start()
	}
}
func (s *syncBatchDownloader) download(from, to uint64) {
	s.executor.exec(syncTask{
		task: &chunkTask{
			from:       from,
			to:         to,
			downloader: s.downloader,
		},
	})
}

func (s *syncBatchDownloader) allocateTasks() {
	from := s.batchEnd + 1
	s.batchEnd = from + batchSyncTask*syncTaskSize

	to := atomic.LoadUint64(&s.to)
	if s.batchEnd > to {
		s.batchEnd = to
	}

	cs := splitChunk(from, s.batchEnd, syncTaskSize)
	for _, c := range cs {
		s.executor.add(syncTask{
			task: &chunkTask{
				from:       c[0],
				to:         c[1],
				downloader: s.downloader,
			},
		})
	}
}

func (s *syncBatchDownloader) setTo(to uint64) {
	atomic.StoreUint64(&s.to, to)

	if s.batchEnd > s.to {
		// remove taller tasks
		s.executor.deleteFrom(s.to)
	}
}

func (s *syncBatchDownloader) subscribe(done func(err error)) {
	s.observer = done
}
