package net

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"
)

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

func TestExecutor_Add(t *testing.T) {
	const from uint64 = 1
	const chunk uint64 = 20
	const max = 100
	const batch = 3
	const to = chunk*max + from - 1

	exec := newExecutor(max, batch)
	exec.start()

	cs := splitChunk(from, to, chunk)
	if len(cs) != max {
		t.Errorf("split chunks error: %d", len(cs))
	}

	mdownloader := &mockDownloader{}
	for i := 0; i < len(cs); i++ {
		success := exec.add(&syncTask{
			task: &chunkTask{
				from:       cs[i][0],
				to:         cs[i][1],
				downloader: mdownloader,
			},
		})

		if !success {
			t.Errorf("should add success")
		}
	}
	success := exec.add(&syncTask{})
	if success {
		t.Errorf("should add fail, because task queue is full")
	}
}

func TestExecutor_DeleteFrom(t *testing.T) {
	const from uint64 = 1
	const chunk uint64 = 20
	const max = 100
	const batch = 3
	const to = chunk*max + from - 1

	exec := newExecutor(max, batch)
	exec.start()

	cs := splitChunk(from, to, chunk)

	mdownloader := &mockDownloader{}

	for i := 0; i < len(cs); i++ {
		exec.add(&syncTask{
			task: &chunkTask{
				from:       cs[i][0],
				to:         cs[i][1],
				downloader: mdownloader,
			},
		})
	}

	target := to >> 1
	start := exec.deleteFrom(target)

	var shouldStart uint64
	for i := 0; i < len(cs); i++ {
		if cs[i][0] >= target {
			shouldStart = cs[i][0]
			break
		}
	}

	if start != shouldStart {
		t.Fatalf("target %d, task next start should be %d, but get %d", target, shouldStart, start)
	}
}

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
}

func (m *mockListener) taskDone(t syncTask, err error) {
	fmt.Println("task", t.String())
}

func (m *mockListener) allTaskDone(last syncTask) {
	fmt.Println("all task done, last", last.String())
}

func TestExecutor_Exec(t *testing.T) {
	const from uint64 = 1
	const chunk uint64 = 20
	const max = 10
	const batch = 3
	const to = chunk*max + from - 1

	exec := newExecutor(max, batch)
	exec.addListener(&mockListener{})
	exec.start()

	mdownloader := &mockDownloader{}

	exec.exec(&syncTask{
		task: &chunkTask{
			from:       10,
			to:         20,
			downloader: mdownloader,
		},
	})

	cs := splitChunk(from, to, chunk)

	for i := 0; i < len(cs); i++ {
		exec.add(&syncTask{
			task: &chunkTask{
				from:       cs[i][0],
				to:         cs[i][1],
				downloader: mdownloader,
			},
		})
	}

	time.Sleep(time.Hour)
}
