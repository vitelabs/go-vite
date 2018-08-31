package protocols

import (
	"sync"
	"time"
)

const reqCountCacheLimit = 3

type Equal interface {
	Equal(interface{}) bool
}

type reqStatus int

const (
	reqPending reqStatus = iota
	reqDoing
	reqDone
)

var reqStatusText = [...]string{
	reqPending: "pending",
	reqDoing:   "doing",
	reqDone:    "done",
}

func (s reqStatus) String() string {
	return reqStatusText[s]
}

type req struct {
	id uint64

	// Count the number of times the req was requested
	cacheTimes int

	params Equal

	status reqStatus

	time time.Time
}

func (r *req) Equal(r2 interface{}) bool {
	r3, ok := r2.(*req)
	if !ok {
		return false
	}

	if r.id == r3.id {
		return true
	}

	return r.params.Equal(r3.params)
}

// send the req to Peer immediately
func (r *req) Emit() {
	r.cacheTimes = 0
	r.status = reqDoing
	// todo
}

const MAX_ID uint64 = ^(uint64(0))

type reqPool struct {
	wait      []*req
	pending   []*req
	done      []*req
	mu        sync.RWMutex
	currentID uint64
}

func (l *reqPool) Add(r *req) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, req := range l.reqs {
		if req.Equal(r) {
			if r.cacheTimes >= reqCountCacheLimit {
				r.Emit()
				return
			}
		}
	}

	if l.currentID == MAX_ID {
		l.currentID = 0
	}
	l.currentID++

	r.id = l.currentID
	l.reqs = append(l.reqs, r)
}
