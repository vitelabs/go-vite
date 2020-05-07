package syncer

import (
	"sync"
	"time"

	"strconv"

	"github.com/vitelabs/go-vite/interval/common"
	"github.com/vitelabs/go-vite/interval/common/face"
	"github.com/vitelabs/go-vite/interval/p2p"
)

type retryPolicy interface {
	retry(hash string) bool
	done(hash string)
}

type RetryStatus struct {
	cnt   int
	done  bool
	ftime time.Time // first time
	dtime time.Time // done time
}

func (rs *RetryStatus) reset() {
	rs.cnt = 1
	rs.done = false
	rs.ftime = time.Now()
}

func (rs *RetryStatus) finish() {
	rs.done = true
	rs.cnt = 0
	rs.dtime = time.Now()
}
func (rs *RetryStatus) inc() {
	rs.cnt = rs.cnt + 1
}

type defaultRetryPolicy struct {
	fetchedHashs map[string]*RetryStatus
	mu           sync.Mutex
}

func (rp *defaultRetryPolicy) done(hash string) {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	status, ok := rp.fetchedHashs[hash]
	if ok {
		status.finish()
	} else {
		tmp := rp.newRetryStatus()
		tmp.finish()
		rp.fetchedHashs[hash] = tmp
	}
}

func (rp *defaultRetryPolicy) retry(hash string) bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	status, ok := rp.fetchedHashs[hash]
	now := time.Now()
	if ok {
		status.inc()
		if status.done {
			// cnt>5 && now - dtime > 10s
			if status.cnt > 5 && now.After(status.dtime.Add(time.Second*3)) {
				status.reset()
				return true
			}
		} else {
			// cnt>5 && now - ftime > 5s
			if status.cnt > 10 && now.After(status.ftime.Add(time.Second*2)) {
				status.reset()
				return true
			}
		}
	} else {
		rp.fetchedHashs[hash] = rp.newRetryStatus()
		return true
	}
	return false
}

func (rp *defaultRetryPolicy) newRetryStatus() *RetryStatus {
	return &RetryStatus{done: false, cnt: 1, ftime: time.Now()}
}

type addressRetryPolicy struct {
	fetchedAddr sync.Map
}

func (arp *addressRetryPolicy) done(key string) {
}

func (arp *addressRetryPolicy) retry(key string) bool {
	value, ok := arp.fetchedAddr.Load(key)
	if ok && value != nil {
		var status = value.(*RetryStatus)
		now := time.Now()
		status.inc()
		if now.After(status.dtime.Add(time.Second)) {
			return true
		}
		// cnt>5 && now - dtime > 10s
		if status.cnt > 3 {
			status.reset()
			return true
		}
	} else {
		arp.fetchedAddr.Store(key, arp.newRetryStatus())
		return true
	}
	return false
}

func (arp *addressRetryPolicy) newRetryStatus() *RetryStatus {
	return &RetryStatus{done: false, cnt: 1, ftime: time.Now()}
}

type fetcher struct {
	sender *sender

	retryPolicy  retryPolicy
	addressRetry retryPolicy
}

func (f *fetcher) fetchSnapshotBlockFromPeer(hash common.HashHeight, peer p2p.Peer) {
	if f.retryPolicy.retry(hash.Hash) {
		f.sender.requestSnapshotBlockByPeer(hash, peer)
	}
}

func (f *fetcher) FetchAccount(address string, hash common.HashHeight, prevCnt uint64) {
	if prevCnt <= 0 {
		return
	}
	if f.retryPolicy.retry(hash.Hash + strconv.FormatUint(hash.Height, 10)) {
		f.sender.RequestAccountHash(address, hash, prevCnt)
	}
}
func (f *fetcher) Fetch(request face.FetchRequest) {
	if request.PrevCnt <= 0 {
		return
	}
	if f.retryPolicy.retry(request.Hash + strconv.FormatUint(request.Height, 10)) {
		hashH := common.HashHeight{Hash: request.Hash, Height: request.Height}
		if request.Chain == "" {
			f.sender.RequestSnapshotHash(hashH, request.PrevCnt)
		} else {
			f.sender.RequestAccountHash(request.Chain, hashH, request.PrevCnt)
		}
	}
}
func (f *fetcher) FetchSnapshot(hash common.HashHeight, prevCnt uint64) {
	if prevCnt <= 0 {
		return
	}
	if f.retryPolicy.retry(hash.Hash + strconv.FormatUint(hash.Height, 10)) {
		f.sender.RequestSnapshotHash(hash, prevCnt)
	}
}

func (f *fetcher) fetchSnapshotBlockByHash(tasks []common.HashHeight) {
	var target []common.HashHeight
	for _, task := range tasks {
		if f.retryPolicy.retry(task.Hash) {
			target = append(target, task)
		}
	}
	if len(target) > 0 {
		f.sender.RequestSnapshotBlocks(target)
	}
}

func (f *fetcher) fetchAccountBlockByHash(address string, tasks []common.HashHeight) {
	var target []common.HashHeight
	for _, task := range tasks {
		if f.retryPolicy.retry(task.Hash) {
			target = append(target, task)
		}
	}
	if len(target) > 0 {
		f.sender.RequestAccountBlocks(address, target)
	}
}

func (f *fetcher) done(block string, height uint64) {
	f.retryPolicy.done(block)
}
