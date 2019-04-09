package net

import (
	"fmt"
	"testing"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
)

type mock_executor struct {
}

func (mock_executor) end() (i uint64, s syncTaskType) {
	return
}

func (mock_executor) add(t *syncTask) {
}

func (mock_executor) exec(t *syncTask) {
}

func (mock_executor) runTo(to uint64) {
}

func (mock_executor) deleteFrom(start uint64) (nextStart uint64) {
	return 0
}

func (mock_executor) last() *syncTask {
	return nil
}

func (mock_executor) terminate() {
	return
}

func (mock_executor) status() (e ExecutorStatus) {
	return
}

func TestSyncer_Timeout(t *testing.T) {
	log := log15.New("module", "syncer_test")

	var ps = newPeerSet()
	_ = ps.Add(NewMockPeer("peer1", 7000, types.Hash{}))

	const timeout = 5 * time.Second
	var s = syncer{
		syncTimeout: timeout,
		peers:       ps,
		fileMap:     make(map[filename]*fileRecord),
		exec:        &mock_executor{},
		subs:        make(map[int]SyncStateCallback),
		chain:       &mock_chain{},
		log:         log,
	}

	go s.Start()

	time.Sleep(20*time.Second + timeout)

	if s.state != Syncerr {
		fmt.Println(s.state, s.from, s.to, s.current)
		t.Fail()
	}
}
