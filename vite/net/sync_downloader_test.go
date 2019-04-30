package net

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/interfaces"
)

func TestMissingChunks(t *testing.T) {
	var chunks = [][2]uint64{
		{10, 20},
		{30, 40},
		{35, 45},
		{40, 50},
	}

	mis := missingChunks(chunks, 2, 60)
	// mis should be [2, 9] [21, 29] [51, 60]
	if len(mis) != 3 || mis[0] != [2]uint64{2, 9} || mis[1] != [2]uint64{21, 29} || mis[2] != [2]uint64{51, 60} {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingChunks(chunks, 30, 60)
	// mis should be [51, 60]
	if len(mis) != 1 || mis[0] != [2]uint64{51, 60} {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingChunks(chunks, 0, 10)
	if len(mis) != 1 || mis[0] != [2]uint64{0, 9} {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingChunks(chunks, 60, 70)
	if len(mis) != 1 || mis[0] != [2]uint64{60, 70} {
		t.Errorf("wrong mis: %v", mis)
	}
}

func TestMissingSegments(t *testing.T) {
	var chunks = interfaces.SegmentList{
		{Bound: [2]uint64{10, 20}},
		{Bound: [2]uint64{30, 40}},
		{Bound: [2]uint64{35, 45}},
		{Bound: [2]uint64{40, 50}},
	}

	mis := missingSegments(chunks, 2, 60)
	// mis should be [2, 9] [21, 29] [51, 60]
	if len(mis) != 3 || mis[0] != [2]uint64{2, 9} || mis[1] != [2]uint64{21, 29} || mis[2] != [2]uint64{51, 60} {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingSegments(chunks, 30, 60)
	// mis should be [51, 60]
	if len(mis) != 1 || mis[0] != [2]uint64{51, 60} {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingSegments(chunks, 0, 10)
	if len(mis) != 1 || mis[0] != [2]uint64{0, 9} {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingSegments(chunks, 60, 70)
	if len(mis) != 1 || mis[0] != [2]uint64{60, 70} {
		t.Errorf("wrong mis: %v", mis)
	}
}

func TestMissingTasks(t *testing.T) {
	var tasks syncTasks
	var mis syncTasks

	mis = missingTasks(tasks, 2, 60)
	if len(mis) != 1 || mis[0].from != 2 || mis[0].to != 60 {
		t.Errorf("wrong mis: %v", mis)
	}

	tasks = syncTasks{
		&syncTask{from: 10, to: 20},
		&syncTask{from: 30, to: 40},
		&syncTask{from: 35, to: 45},
		&syncTask{from: 40, to: 50},
	}

	mis = missingTasks(tasks, 2, 60)
	// mis should be [2, 9] [21, 29] [51, 60]
	if len(mis) != 3 ||
		mis[0].from != 2 || mis[0].to != 9 ||
		mis[1].from != 21 || mis[1].to != 29 ||
		mis[2].from != 51 || mis[2].to != 60 {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingTasks(tasks, 30, 60)
	// mis should be [51, 60]
	if len(mis) != 1 || mis[0].from != 51 || mis[0].to != 60 {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingTasks(tasks, 0, 10)
	if len(mis) != 1 || mis[0].from != 0 || mis[0].to != 9 {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingTasks(tasks, 60, 70)
	if len(mis) != 1 || mis[0].from != 60 || mis[0].to != 70 {
		t.Errorf("wrong mis: %v", mis)
	}
}

func TestChunksOverlap(t *testing.T) {
	var cs = make(chunks, 0, 1)
	var ok bool
	var chunk [2]uint64

	if chunk, ok = cs.overlap(1, 9); !ok {
		t.Errorf("should not overlap")
	}

	cs = [][2]uint64{
		{10, 20},
		{22, 40},
		{41, 50},
	}

	if chunk, ok = cs.overlap(1, 9); !ok {
		t.Errorf("should not overlap")
	}

	if chunk, ok = cs.overlap(51, 60); !ok {
		t.Errorf("should not overlap")
	}

	if chunk, ok = cs.overlap(20, 21); ok || chunk != [2]uint64{10, 20} {
		t.Errorf("should overlap")
	}

	if chunk, ok = cs.overlap(50, 61); ok || chunk != [2]uint64{41, 50} {
		t.Errorf("should overlap")
	}

	if chunk, ok = cs.overlap(19, 42); ok || chunk != [2]uint64{10, 20} {
		t.Errorf("should overlap")
	}
}

func TestExecutor_cancel(t *testing.T) {
	exec := newExecutor(100, 3, nil, nil)
	exec.start()

	exec.download(1, 10, false)
	exec.download(11, 20, false)
	exec.download(21, 30, false)
	exec.download(31, 40, false)

	exec.cancel(13)

	if len(exec.tasks) != 2 {
		t.Errorf("wrong tasks length: %d", len(exec.tasks))
	}
	if exec.tasks[1].from != 11 || exec.tasks[1].to != 12 {
		t.Errorf("wrong task")
	}
}

func TestCancelTasks(t *testing.T) {
	var tasks = syncTasks{
		{from: 1, to: 10},
		{from: 11, to: 20},
		{from: 21, to: 30},
		{from: 31, to: 40},
		{from: 41, to: 50},
		{from: 51, to: 60},
	}

	tasks, end := cancelTasks(tasks, 15)
	var cs = [][2]uint64{{1, 10}}
	if len(tasks) != len(cs) {
		t.Errorf("wrong tasks: %d", len(tasks))
	} else {
		if end != cs[len(cs)-1][1] {
			t.Errorf("wrong end: %d", end)
		}
		for i, t2 := range tasks {
			if t2.from != cs[i][0] || t2.to != cs[i][1] {
				t.Errorf("wrong task: %v", t2)
			}
		}
	}
}

func TestRunTasks(t *testing.T) {
	var tasks = syncTasks{
		{
			from:   3,
			to:     10,
			st:     reqDone,
			doneAt: time.Unix(time.Now().Unix()-10, 0), // will be clean
		},
		{
			from: 11,
			to:   15,
		},
		{
			from:   16,
			to:     20,
			st:     reqDone,
			doneAt: time.Now(),
		},
		{
			from: 21,
			to:   25,
			st:   reqError,
		},
		{
			from: 26,
			to:   30,
			st:   reqCancel,
		},
		{
			from: 31,
			to:   35,
		},
	}

	var wg sync.WaitGroup
	var run = func(t *syncTask) {
		if t.st == reqDone || t.st == reqPending {
			panic(fmt.Sprintf("run task %d-%d repeatedly", t.from, t.to))
		}

		fmt.Printf("run task %d-%d\n", t.from, t.to)
		t.st = reqPending

		wg.Add(1)
		go func() {
			defer wg.Done()

			time.Sleep(time.Second)
			t.st = reqDone
			t.doneAt = time.Now()
		}()
	}

	for {
		if before := len(tasks); before > 0 {
			tasks = runTasks(tasks, 3, run)
			fmt.Printf("before: %d, after: %d\n", before, len(tasks))
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

func TestAddTasks(t *testing.T) {
	var tasks syncTasks

	reset := func() {
		tasks = syncTasks{
			{
				from: 1,
				to:   10,
				st:   reqDone,
			},
			{
				from: 11,
				to:   20,
			},
			{
				from: 31,
				to:   40,
				st:   reqDone,
			},
			{
				from: 51,
				to:   60,
				st:   reqError,
			},
		}
	}

	type sample struct {
		from, to uint64
		must     bool
		cs       [][2]uint64
	}
	var samples = []sample{
		{1, 70, false, [][2]uint64{{1, 10}, {11, 20}, {21, 30}, {31, 40}, {41, 50}, {51, 60}, {61, 70}}},
		{1, 70, true, [][2]uint64{{1, 10}, {11, 20}, {21, 50}, {51, 60}, {61, 70}}},
	}

	for _, samp := range samples {
		reset()
		tasks = addTasks(tasks, samp.from, samp.to, samp.must)
		if len(tasks) != len(samp.cs) {
			t.Errorf("wrong tasks length: %d", len(tasks))
		} else {
			for i, c := range samp.cs {
				if tasks[i].from != c[0] || tasks[i].to != c[1] {
					t.Errorf("wrong task: %d - %d", tasks[i].from, tasks[i].to)
				}
			}
		}
	}
}

type mockDownloader struct {
}

func (m mockDownloader) download(ctx context.Context, from, to uint64) <-chan error {
	ch := make(chan error, 1)
	ch <- nil
	return ch
}

func (m mockDownloader) start() {
	return
}

func (m mockDownloader) stop() {
	return
}

//func TestExecutor_Add(t *testing.T) {
//	const from uint64 = 1
//	const chunk uint64 = 20
//	const max = 100
//	const batch = 3
//	const to = chunk*max + from - 1
//
//	exec := newExecutor(max, batch)
//	exec.start()
//
//	cs := splitChunk(from, to, chunk)
//	if len(cs) != max {
//		t.Errorf("split chunks error: %d", len(cs))
//	}
//
//	mdownloader := &mockDownloader{}
//	for i := 0; i < len(cs); i++ {
//		success := exec.add(&syncTask{
//			task: &chunkTask{
//				from:       cs[i][0],
//				to:         cs[i][1],
//				downloader: mdownloader,
//			},
//		})
//
//		if !success {
//			t.Errorf("should add success")
//		}
//	}
//	success := exec.add(&syncTask{})
//	if success {
//		t.Errorf("should add fail, because task queue is full")
//	}
//}
//
//func TestExecutor_DeleteFrom(t *testing.T) {
//	const from uint64 = 1
//	const chunk uint64 = 20
//	const max = 100
//	const batch = 3
//	const to = chunk*max + from - 1
//
//	exec := newExecutor(max, batch)
//	exec.start()
//
//	cs := splitChunk(from, to, chunk)
//
//	mdownloader := &mockDownloader{}
//
//	for i := 0; i < len(cs); i++ {
//		exec.add(&syncTask{
//			task: &chunkTask{
//				from:       cs[i][0],
//				to:         cs[i][1],
//				downloader: mdownloader,
//			},
//		})
//	}
//
//	target := to >> 1
//	start := exec.deleteFrom(target)
//
//	var shouldStart uint64
//	for i := 0; i < len(cs); i++ {
//		if cs[i][0] >= target {
//			shouldStart = cs[i][0]
//			break
//		}
//	}
//
//	if start != shouldStart {
//		t.Fatalf("target %d, task next start should be %d, but get %d", target, shouldStart, start)
//	}
//}

type mockTask struct {
	from, to uint64
}

func (m mockTask) bound() (from, to uint64) {
	return m.from, m.to
}

func (m mockTask) do(ctx context.Context) error {
	return nil
}

func (m mockTask) String() string {
	return strconv.FormatUint(m.from, 10) + "-" + strconv.FormatUint(m.to, 10)
}

type mockListener struct {
	done func(t *syncTask)
}

func (m *mockListener) taskDone(t *syncTask, err error) {
	fmt.Println("task", t.String())
}

func (m *mockListener) allTaskDone(last *syncTask) {
	fmt.Println("last", last.String())
	m.done(last)
}

//func TestExecutor_Exec(t *testing.T) {
//	const from uint64 = 1
//	const chunk uint64 = 20
//	const max = 10
//	const batch = 3
//	const to = chunk*max + from - 1
//
//	pending := make(chan struct{})
//
//	exec := newExecutor(max, batch)
//	exec.addListener(&mockListener{
//		done: func(last *syncTask) {
//			if _, to2 := last.bound(); to2 != to {
//				t.Errorf("wrong last task")
//			}
//			close(pending)
//		},
//	})
//	exec.start()
//
//	mdownloader := &mockDownloader{}
//
//	exec.exec(&syncTask{
//		task: &chunkTask{
//			from:       10,
//			to:         20,
//			downloader: mdownloader,
//		},
//	})
//
//	cs := splitChunk(from, to, chunk)
//
//	for i := 0; i < len(cs); i++ {
//		exec.add(&syncTask{
//			task: &chunkTask{
//				from:       cs[i][0],
//				to:         cs[i][1],
//				downloader: mdownloader,
//			},
//		})
//	}
//
//	<-pending
//}

type mockQueue struct {
	tasks   syncTasks
	max     int
	batch   int
	mu      sync.Mutex
	cond    *sync.Cond
	running bool
}

func newMockQueue(max, batch int) *mockQueue {
	q := &mockQueue{
		tasks:   make(syncTasks, 0, max),
		max:     max,
		batch:   batch,
		cond:    nil,
		running: false,
	}

	q.cond = sync.NewCond(&q.mu)

	return q
}

func (m *mockQueue) start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	var t *syncTask
	var batch int
	var total int
	var continuous bool
	var done int
	for {
		m.mu.Lock()
	Wait:
		for {
			total = len(m.tasks)
			if total == 0 && m.running {
				m.cond.Wait()
				continue
			}
			break
		}

		if false == m.running {
			m.mu.Unlock()
			return
		}

	Run:
		batch = 0
		continuous = true
		done = 0
		for i := 0; i < total; i++ {
			t = m.tasks[i]
			if t.st == reqDone {
				if continuous {
					done++
				}
				continue
			} else {
				continuous = false
				if done > m.max/2 {
					n := copy(m.tasks, m.tasks[done:])
					m.tasks = m.tasks[:n]
					fmt.Printf("clean %d tasks from %d, rest %d\n", done, total, n)
					m.cond.Broadcast()
					goto Run
				}
			}

			if t.st == reqPending {
				batch++
				continue
			}

			if batch >= m.batch {
				m.cond.Wait()
				goto Wait
			} else {
				batch++
				m.run(t)
			}
		}

		m.mu.Unlock()
	}
}

func (m *mockQueue) stop() {
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()

	m.cond.Broadcast()
}

func (m *mockQueue) run(t *syncTask) {
	t.st = reqPending

	go func() {
		n := time.Duration(rand.Intn(500))
		time.Sleep(n * time.Millisecond)
		fmt.Printf("task %d-%d done\n", t.from, t.to)
		t.st = reqDone
		t.doneAt = time.Now()
		m.cond.Signal()
	}()
}

func (m *mockQueue) add(from, to uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for {
		if len(m.tasks) >= m.max && m.running {
			m.cond.Wait()
		} else {
			break
		}
	}

	if false == m.running {
		return false
	}

	m.tasks = append(m.tasks, &syncTask{
		from: from,
		to:   to,
	})

	sort.Sort(m.tasks)
	m.cond.Signal()
	//m.cond.Broadcast()
	return true
}

func TestMockQueue(t *testing.T) {
	queue := newMockQueue(20, 3)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		queue.start()
	}()

	time.Sleep(500 * time.Millisecond)

	wg.Add(1)
	go func() {
		defer wg.Done()

		taskChan := make(chan [2]uint64, 10)

		const max = 2
		for i := 0; i < max; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for chunk := range taskChan {
					if false == queue.add(chunk[0], chunk[1]) {
						fmt.Printf("failed to add task %d-%d\n", chunk[0], chunk[1])
						return
					}
				}
			}()
		}

		const start = 1
		const end = 10000
		var from, to uint64
		from = start

		for from = start; to < end; from = to + 1 {
			to = from + 100 - 1
			if to > end {
				to = end
			}

			taskChan <- [2]uint64{from, to}
		}

		close(taskChan)
	}()

	go func() {
		err := http.ListenAndServe("127.0.0.1:8080", nil)
		if err != nil {
			panic(err)
		}
	}()

	wg.Wait()
}
