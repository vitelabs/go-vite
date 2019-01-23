package net

import (
	"context"
	"sync"

	"github.com/vitelabs/go-vite/ledger"
)

type syncTaskType int

const (
	syncFileTask syncTaskType = iota
	syncChunkTask
)

type syncTask interface {
	bound() (from, to uint64)
	state() reqState
	setState(st reqState)
	do(ctx context.Context) error
	taskType() syncTaskType
	info() string
	cancel()
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
	st         reqState
	file       File
	downloader fileDownloader
	ctx        context.Context
}

func (f *fileTask) cancel() {
}

func (f *fileTask) info() string {
	panic("implement me")
}

func (f *fileTask) state() reqState {
	return f.st
}

func (f *fileTask) setState(st reqState) {
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
	st         reqState
	downloader chunkDownloader
}

func (c *chunkTask) info() string {
	panic("implement me")
}

func (c *chunkTask) state() reqState {
	return c.st
}

func (c *chunkTask) setState(st reqState) {
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
	cancel(t syncTask)
	runTo(to uint64)
	last() syncTask
	terminate()
}

type syncTaskListener interface {
	done(t syncTask)
	catch(t syncTask, err error)
	nonTask(last syncTask)
}

type executor struct {
	mu    sync.Mutex
	tasks []syncTask

	doneIndex int
	listener  syncTaskListener

	ctx       context.Context
	ctxCancel func()
}

func (e *executor) cancel(t syncTask) {
	panic("implement me")
}

func newExecutor(listener syncTaskListener) syncTaskExecutor {
	ctx, cancel := context.WithCancel(context.Background())

	return &executor{
		listener:  listener,
		ctx:       ctx,
		ctxCancel: cancel,
	}
}

func (e *executor) add(t syncTask) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tasks = append(e.tasks, t)
}

func (e *executor) runTo(to uint64) {
	e.mu.Lock()

	var skip = 0 // pending / cancel / done but not continue
	var index = 0
	var continuous = true // is task done continuously

	for index = e.doneIndex + skip; index < len(e.tasks); index = e.doneIndex + skip {
		t := e.tasks[index]
		st := t.state()

		if st == reqDone && continuous {
			e.doneIndex++
		} else if st == reqPending || st == reqDone {
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
	e.listener.nonTask(last)
}

func (e *executor) run(t syncTask) {
	t.setState(reqPending)
	go e.do(t)
}

func (e *executor) do(t syncTask) {
	if err := t.do(e.ctx); err != nil {
		t.setState(reqError)
		e.listener.catch(t, err)
	} else {
		t.setState(reqDone)
		e.listener.done(t)
	}
}

func (e *executor) last() syncTask {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.tasks[len(e.tasks)-1]
}

func (e *executor) terminate() {
	panic("implement me")
}
