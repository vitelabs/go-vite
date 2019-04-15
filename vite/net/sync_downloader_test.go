package net

import (
	"context"
	"fmt"
	"strconv"
	"testing"

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
		{10, 20},
		{30, 40},
		{35, 45},
		{40, 50},
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
	var chunks = syncTasks{
		&syncTask{from: 10, to: 20},
		&syncTask{from: 30, to: 40},
		&syncTask{from: 35, to: 45},
		&syncTask{from: 40, to: 50},
	}

	mis := missingTasks(chunks, 2, 60)
	// mis should be [2, 9] [21, 29] [51, 60]
	if len(mis) != 3 ||
		mis[0].from != 2 || mis[0].to != 9 ||
		mis[1].from != 21 || mis[1].to != 29 ||
		mis[2].from != 51 || mis[2].to != 60 {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingTasks(chunks, 30, 60)
	// mis should be [51, 60]
	if len(mis) != 1 || mis[0].from != 51 || mis[0].to != 60 {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingTasks(chunks, 0, 10)
	if len(mis) != 1 || mis[0].from != 0 || mis[0].to != 9 {
		t.Errorf("wrong mis: %v", mis)
	}

	mis = missingTasks(chunks, 60, 70)
	if len(mis) != 1 || mis[0].from != 60 || mis[0].to != 70 {
		t.Errorf("wrong mis: %v", mis)
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
