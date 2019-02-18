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

	if exec.end() != to {
		t.Fatalf("task end should be %d, but get %d", to, exec.end())
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
