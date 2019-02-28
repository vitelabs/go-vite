package net

import (
	"context"
	"strconv"
	"testing"

	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite/net/message"
)

func Test_Syncer_receiveFileList(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	exec := &executor{

		tasks:     make([]*syncTask, 0, SYNC_TASK_BATCH),
		ctx:       ctx,
		ctxCancel: cancel,
	}

	var s = &syncer{
		fileMap: make(map[filename]*fileRecord),
		exec:    exec,
	}

	var count int
	var m message.FileList

	count = (943201 - 1) / 3600
	for i := uint64(1); i < 943201; i++ {
		j := i + 3599
		m.Files = append(m.Files, &ledger.CompressedFileMeta{
			StartHeight: i,
			EndHeight:   j,
			Filename:    "subgraph_" + strconv.FormatUint(i, 10) + "_" + strconv.FormatUint(j, 10),
		})
		i = j + 1
	}
	m.Chunks = append(m.Chunks, [2]uint64{943201, 8157244})

	s.receiveFileList(&m)
	if len(exec.tasks) != count {
		t.Fatalf("task count is not %d", count)
	}

	count = (8157601 - 1) / 3600
	for i := uint64(1); i < 8157601; i++ {
		j := i + 3599
		m.Files = append(m.Files, &ledger.CompressedFileMeta{
			StartHeight: i,
			EndHeight:   j,
			Filename:    "subgraph_" + strconv.FormatUint(i, 10) + "_" + strconv.FormatUint(j, 10),
		})
		i = j + 1
	}
	m.Chunks = nil

	s.receiveFileList(&m)

	if len(exec.tasks) != count {
		t.Fatalf("task count is not %d", count)
	}
}
