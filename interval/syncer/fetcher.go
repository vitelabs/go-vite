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

func (self *RetryStatus) reset() {
	self.cnt = 1
	self.done = false
	self.ftime = time.Now()
}

func (self *RetryStatus) finish() {
	self.done = true
	self.cnt = 0
	self.dtime = time.Now()
}
func (self *RetryStatus) inc() {
	self.cnt = self.cnt + 1
}

type defaultRetryPolicy struct {
	fetchedHashs map[string]*RetryStatus
	mu           sync.Mutex
}

func (self *defaultRetryPolicy) done(hash string) {
	self.mu.Lock()
	defer self.mu.Unlock()
	status, ok := self.fetchedHashs[hash]
	if ok {
		status.finish()
	} else {
		tmp := self.newRetryStatus()
		tmp.finish()
		self.fetchedHashs[hash] = tmp
	}
}

func (self *defaultRetryPolicy) retry(hash string) bool {
	self.mu.Lock()
	defer self.mu.Unlock()
	status, ok := self.fetchedHashs[hash]
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
		self.fetchedHashs[hash] = self.newRetryStatus()
		return true
	}
	return false
}

func (self *defaultRetryPolicy) newRetryStatus() *RetryStatus {
	return &RetryStatus{done: false, cnt: 1, ftime: time.Now()}
}

type addressRetryPolicy struct {
	fetchedAddr sync.Map
}

func (self *addressRetryPolicy) done(key string) {
}

func (self *addressRetryPolicy) retry(key string) bool {
	value, ok := self.fetchedAddr.Load(key)
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
		self.fetchedAddr.Store(key, self.newRetryStatus())
		return true
	}
	return false
}

func (self *addressRetryPolicy) newRetryStatus() *RetryStatus {
	return &RetryStatus{done: false, cnt: 1, ftime: time.Now()}
}

type fetcher struct {
	sender *sender

	retryPolicy  retryPolicy
	addressRetry retryPolicy
}

func (self *fetcher) fetchSnapshotBlockFromPeer(hash common.HashHeight, peer p2p.Peer) {
	if self.retryPolicy.retry(hash.Hash) {
		self.sender.requestSnapshotBlockByPeer(hash, peer)
	}
}

func (self *fetcher) FetchAccount(address string, hash common.HashHeight, prevCnt uint64) {
	if prevCnt <= 0 {
		return
	}
	if self.retryPolicy.retry(hash.Hash + strconv.FormatUint(hash.Height, 10)) {
		self.sender.RequestAccountHash(address, hash, prevCnt)
	}
}
func (self *fetcher) Fetch(request face.FetchRequest) {
	if request.PrevCnt <= 0 {
		return
	}
	if self.retryPolicy.retry(request.Hash + strconv.FormatUint(request.Height, 10)) {
		hashH := common.HashHeight{Hash: request.Hash, Height: request.Height}
		if request.Chain == "" {
			self.sender.RequestSnapshotHash(hashH, request.PrevCnt)
		} else {
			self.sender.RequestAccountHash(request.Chain, hashH, request.PrevCnt)
		}
	}
}
func (self *fetcher) FetchSnapshot(hash common.HashHeight, prevCnt uint64) {
	if prevCnt <= 0 {
		return
	}
	if self.retryPolicy.retry(hash.Hash + strconv.FormatUint(hash.Height, 10)) {
		self.sender.RequestSnapshotHash(hash, prevCnt)
	}
}

func (self *fetcher) fetchSnapshotBlockByHash(tasks []common.HashHeight) {
	var target []common.HashHeight
	for _, task := range tasks {
		if self.retryPolicy.retry(task.Hash) {
			target = append(target, task)
		}
	}
	if len(target) > 0 {
		self.sender.RequestSnapshotBlocks(target)
	}
}

func (self *fetcher) fetchAccountBlockByHash(address string, tasks []common.HashHeight) {
	var target []common.HashHeight
	for _, task := range tasks {
		if self.retryPolicy.retry(task.Hash) {
			target = append(target, task)
		}
	}
	if len(target) > 0 {
		self.sender.RequestAccountBlocks(address, target)
	}
}

func (self *fetcher) done(block string, height uint64) {
	self.retryPolicy.done(block)
}
