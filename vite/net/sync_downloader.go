package net

import (
	"context"
	"fmt"
	net2 "net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"
)

type reqState byte

const (
	reqWaiting reqState = iota
	reqPending
	reqDone
	reqError
	reqCancel
)

var reqStatus = map[reqState]string{
	reqWaiting: "waiting",
	reqPending: "pending",
	reqDone:    "done",
	reqError:   "error",
	reqCancel:  "canceled",
}

func (s reqState) String() string {
	if str, ok := reqStatus[s]; ok {
		return str
	}

	return "unknown request state"
}

type syncTask struct {
	from, to  uint64
	st        reqState
	doneAt    time.Time
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (t *syncTask) String() string {
	return strconv.FormatUint(t.from, 10) + "-" + strconv.FormatUint(t.to, 10) + " " + t.st.String()
}

func (t *syncTask) cancel() {
	if t.ctxCancel != nil {
		t.ctxCancel()
	}
}

type syncTasks []*syncTask

func (s syncTasks) Len() int {
	return len(s)
}

func (s syncTasks) Less(i, j int) bool {
	return s[i].from < s[j].from
}

func (s syncTasks) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// chunks should be continuous [from, to]
func missingChunks(chunks [][2]uint64, from, to uint64) (mis [][2]uint64) {
	for _, chunk := range chunks {
		// useless
		if chunk[1] < from {
			continue
		}

		// missing front piece
		if chunk[0] > from {
			mis = append(mis, [2]uint64{
				from,
				chunk[0] - 1,
			})
		}

		// next response
		from = chunk[1] + 1
	}

	// from should equal (cr.to + 1)
	if from-1 < to {
		mis = append(mis, [2]uint64{
			from,
			to,
		})
	}

	return
}

func missingTasks(tasks syncTasks, from, to uint64) (mis syncTasks) {
	for _, t := range tasks {
		// useless
		if t.to < from {
			continue
		}

		// missing front piece
		if t.from > from {
			mis = append(mis, &syncTask{
				from: from,
				to:   t.from - 1,
			})
		}

		// next response
		from = t.to + 1
	}

	// from should equal (cr.to + 1)
	if from-1 < to {
		mis = append(mis, &syncTask{
			from: from,
			to:   to,
		})
	}

	return
}

type syncDownloader interface {
	start()
	stop()
	// will be block, if cannot download (eg. no peers) or task queue is full
	// must will download the task regardless of task repeat
	download(from, to uint64, must bool) bool
	// cancel tasks between from and to
	cancel(from, to uint64)
	addListener(listener taskListener)
}

//type syncExecutor interface {
//	start()
//	stop()
//
//	// add a task into queue, task will be sorted from small to large.
//	// if task is overlapped, then the task will be split to small tasks.
//	add(t *syncTask) bool
//
//	// execute a task directly, NOT put into queue.
//	// if a previous same from-to task has done, the gap time must longer than 1 min, or the task will not exec.
//	exec(t *syncTask) bool
//
//	// runTo will execute those from is small than to tasks
//	// runTo(to uint64)
//
//	// deleteFrom will delete those from is large than start tasks.
//	// return (the last task`s to + 1)
//	deleteFrom(start uint64) (nextStart uint64)
//
//	// last return the last task in queue
//	// ok is false if task queue is empty
//	// last() (t *syncTask, ok bool)
//
//	status() ExecutorStatus
//
//	// clear all tasks
//	reset()
//
//	size() int
//
//	addListener(listener taskListener)
//}

type taskListener = func(from, to uint64, err error)

type executor struct {
	mu         sync.Mutex
	tasks      syncTasks
	cond       *sync.Cond
	max, batch int

	pool    connPool
	factory syncConnInitiator
	dialing map[string]struct{}
	dialer  *net2.Dialer

	listeners []taskListener
	ctx       context.Context
	ctxCancel context.CancelFunc
	running   int32
	wg        sync.WaitGroup

	log log15.Logger
}

type ExecutorStatus struct {
	Tasks []string
}

func newExecutor(max, batch int, pool connPool, factory syncConnInitiator) *executor {
	e := &executor{
		max:     max,
		batch:   batch,
		tasks:   make(syncTasks, 0, max),
		pool:    pool,
		factory: factory,
		dialing: make(map[string]struct{}),
		dialer: &net2.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		},
		log: netLog.New("module", "downloader"),
	}

	e.cond = sync.NewCond(&e.mu)
	return e
}

func (e *executor) size() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return len(e.tasks)
}

func (e *executor) start() {
	if atomic.CompareAndSwapInt32(&e.running, 0, 1) {
		e.ctx, e.ctxCancel = context.WithCancel(context.Background())

		e.tasks = e.tasks[:0]

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

func (e *executor) addListener(listener taskListener) {
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

func (e *executor) download(from, to uint64, must bool) bool {
	if atomic.LoadInt32(&e.running) == 0 {
		return false
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for {
		if len(e.tasks) == cap(e.tasks) {
			e.cond.Wait()
		} else {
			break
		}
	}

	if must {
		e.tasks = append(e.tasks, &syncTask{
			from: from,
			to:   to,
		})
	} else {
		tasks := missingTasks(e.tasks, from, to)
		for _, t := range tasks {
			e.tasks = append(e.tasks, t)
		}
	}

	sort.Sort(e.tasks)

	return true
}

func (e *executor) cancel(from, to uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := len(e.tasks); i > -1; i-- {
		t := e.tasks[i]
		if t.from >= from {
			t.cancel()
		} else if t.to > from {
			t.to = from - 1
		}
	}
}

func (e *executor) reset() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tasks = e.tasks[:0]
}

func (e *executor) loop() {
	defer e.wg.Done()

	for {
		if atomic.LoadInt32(&e.running) == 0 {
			break
		}

		var done = 0
		var index = 0
		var batch = 0
		var t *syncTask

		e.mu.Lock()
		for index = done; index < len(e.tasks); index = done + batch {
			t = e.tasks[index]

			// update downIndex if done tasks are continuous
			if t.st == reqDone {
				done++
			} else if t.st == reqPending {
				batch++
			} else if batch < e.batch {
				batch++
				e.run(t)
			} else {
				break
			}
		}

		// todo clean done tasks
		// more than a half tasks have done
		if done > cap(e.tasks)/2 {
			for index = 0; index < done; index++ {

			}
		}
		e.mu.Unlock()

		time.Sleep(100 * time.Millisecond)
	}
}

func (e *executor) run(t *syncTask) {
	t.st = reqPending

	go e.do(t)
}

func (e *executor) doJob(c syncConnection, from, to uint64) error {
	start := time.Now()

	e.log.Info(fmt.Sprintf("download chunk %d-%d from %s", from, to, c.RemoteAddr()))

	if err := c.download(from, to); err != nil {
		if c.catch(err) {
			e.pool.delConn(c)
		}
		e.log.Error(fmt.Sprintf("download chunk %d-%d from %s error: %v", from, to, c.RemoteAddr(), err))

		return err
	}

	e.log.Info(fmt.Sprintf("download chunk %d-%d from %s elapse %s", from, to, c.RemoteAddr(), time.Now().Sub(start)))

	return nil
}

func (e *executor) createConn(p downloadPeer) (c syncConnection, err error) {
	addr := p.fileAddress()

	e.mu.Lock()
	if _, ok := e.dialing[addr]; ok {
		e.mu.Unlock()
		return nil, errPeerDialing
	}
	e.dialing[addr] = struct{}{}
	e.mu.Unlock()

	tcp, err := e.dialer.Dial("tcp", addr)

	e.mu.Lock()
	delete(e.dialing, addr)
	e.mu.Unlock()

	// dial error
	if err != nil {
		return nil, err
	}

	// handshake error
	c, err = e.factory.initiate(tcp, p)
	if err != nil {
		return
	}

	// add error
	if err = e.pool.addConn(c); err != nil {
		e.pool.delConn(c)
	}

	return
}

func (e *executor) do(t *syncTask) {
	if t.ctx == nil {
		t.ctx, t.ctxCancel = context.WithCancel(e.ctx)
	}

	var p downloadPeer
	var c syncConnection
	var err error

	if p, c, err = e.pool.chooseSource(t.to); err != nil {
		// no peers
		t.st = reqError
	} else if c != nil {
		if err = e.doJob(c, t.from, t.to); err == nil {
			// downloaded
			t.st = reqDone
		} else {
			t.st = reqError
		}
	} else if p != nil {
		if c, err = e.createConn(p); err == nil {
			if err = e.doJob(c, t.from, t.to); err == nil {
				// downloaded
				t.st = reqDone
			} else {
				t.st = reqError
			}
		} else {
			t.st = reqError
		}
	} else {
		t.st = reqWaiting
	}

	if t.st == reqDone {
		e.notify(t, err)
	}
}

func (e *executor) notify(t *syncTask, err error) {
	for _, listener := range e.listeners {
		listener(t.from, t.to, err)
	}
}
