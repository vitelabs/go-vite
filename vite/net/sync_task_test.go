package net

import (
	"math/rand"
	"testing"
)

func TestExecutor_DeleteTo(t *testing.T) {
	from := uint64(0)
	to := uint64(rand.Int31())

	exec := newExecutor(nil)

	cs := splitChunk(from, to, 20)
	for i := 0; i < len(cs); i++ {
		exec.add(&syncTask{
			task: &chunkTask{
				from: cs[i][0],
				to:   cs[i][1],
			},
			typ: syncChunkTask,
		})
	}

	if e, _ := exec.end(); e != to {
		t.Fatalf("task end should be %d, but get %d", to, e)
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

func TestExecutor_DeleteTo2(t *testing.T) {
	const chunkSize = 20
	const minTo = 1000
	const doneTasks = 1000 / chunkSize

	from := uint64(0)
	to := uint64(rand.Intn(100000)) + minTo

	exec := newExecutor(nil)

	cs := splitChunk(from, to, chunkSize)
	for i := 0; i < len(cs); i++ {
		st := &syncTask{
			task: &chunkTask{
				from: cs[i][0],
				to:   cs[i][1],
			},
			typ: syncChunkTask,
		}

		// first 50 tasks set done
		if i < doneTasks || i > doneTasks+1 {
			st.st = reqDone
		}

		exec.add(st)
	}

	start := exec.deleteFrom(0)
	if start != minTo {
		t.Fatalf("start should be 1001, but get %d", start)
	}
}

func TestExecutor_clean(t *testing.T) {
	const total = 1000

	e := &executor{
		tasks: make([]*syncTask, 0, total),
	}

	var st reqState
	var doneCount int
	var rd int
	var sts = make([]reqState, 0, total)
	for i := 0; i < total; i++ {
		rd = rand.Intn(11)
		if rd > 8 {
			st = reqDone
			doneCount++
		} else if rd > 6 {
			st = reqPending
			sts = append(sts, st)
		} else if rd > 4 {
			st = reqError
			sts = append(sts, st)
		} else {
			st = reqWaiting
			sts = append(sts, st)
		}

		e.tasks = append(e.tasks, &syncTask{
			st: st,
		})
	}

	e.clean()

	rest := len(e.tasks)
	if rest != total-doneCount {
		t.Fatalf("tasks count is not %d but %d", total-doneCount, len(e.tasks))
	}
	for i := 0; i < rest; i++ {
		st = e.tasks[i].st
		if st != sts[i] {
			t.Fatalf("tasks state is not right")
		}
	}
}
