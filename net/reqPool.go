package net

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"math/big"
	"sync"
	"time"
)

const reqCountCacheLimit = 3
const maxBlocksOneTrip = 100
const maxBlocksOneFile = 1000
const expireCheckInterval = 30 * time.Second

var errHasNoSuitablePeer = errors.New("has no suitable peer")

type Params interface {
	Equal(interface{}) bool
	Ceil() uint64
}

type reqStatus int

const (
	reqWaiting reqStatus = iota
	reqPending
	reqDone
)

var reqStatusText = [...]string{
	reqWaiting: "waiting",
	reqPending: "pending",
	reqDone:    "done",
}

func (s reqStatus) String() string {
	return reqStatusText[s]
}

type req struct {
	id      uint64
	peer    *Peer
	count   int // Count the number of times the req was requested
	params  Params
	status  reqStatus
	time    time.Time
	timeout time.Duration
	lock    sync.Mutex
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
func (r *req) Execute() {
	r.lock.Lock()

	if r.status == reqPending {
		r.lock.Unlock()
		return
	}
	r.count = 0
	r.status = reqPending
	r.time = time.Now()

	r.lock.Unlock()
}

func (r *req) Cancel() {

}

const MAX_ID uint64 = ^(uint64(0))
const INIT_ID uint64 = 1

type reqPool struct {
	waiting   map[uint64]*req
	pending   map[string]*req
	done      map[uint64]*req
	mu        sync.RWMutex
	currentID uint64
	peers     *peerSet
	errChan   chan *req
	doneChan  chan *req
	addChan   chan *req
	stop      chan struct{}
	log       log15.Logger
}

func NewReqPool(peers *peerSet) *reqPool {
	pool := &reqPool{
		waiting:   make(map[uint64]*req, 100),
		pending:   make(map[string]*req, 20),
		done:      make(map[uint64]*req, 100),
		currentID: INIT_ID,
		peers:     peers,
		errChan:   make(chan *req, 1),
		doneChan:  make(chan *req, 1),
		addChan:   make(chan *req, 1),
		stop:      make(chan struct{}),
		log:       log15.New("module", "net/reqpool"),
	}

	pool.loop()

	return pool
}

func (p *reqPool) loop() {
	ticker := time.NewTicker(expireCheckInterval)
	defer ticker.Stop()

	now := time.Now()
loop:
	for {
		select {
		case <-p.stop:
			break loop
		case req := <-p.doneChan:
			peerId := req.peer.ID
			delete(p.pending, peerId)
			p.done[req.id] = req
		case req := <-p.errChan:
			peerId := req.peer.ID
			delete(p.pending, peerId)
			p.Add(req)
		case <-ticker.C:
			now = time.Now()
			for _, req := range p.pending {
				if now.Sub(req.time) > req.timeout {
					p.log.Error("req timeout")
					p.errChan <- req
				}
			}
		}
	}

	for k, r := range p.pending {
		delete(p.pending, k)
		r.Cancel()
	}

	p.waiting = nil
	p.done = nil
}

func (p *reqPool) Mark(r *req) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, req := range p.waiting {
		if req.Equal(r) {
			r.count++
			return
		}
	}

	if p.currentID == MAX_ID {
		p.currentID = INIT_ID
	} else {
		p.currentID++
	}

	r.id = p.currentID
	p.waiting[r.id] = r
}

func (p *reqPool) Add(r *req) {
	p.Mark(r)

	if r.count >= reqCountCacheLimit {
		p.Execute(r)
	}
}

func (p *reqPool) Execute(r *req) {
	delete(p.waiting, r.id)

	height := new(big.Int)
	height.SetUint64(r.params.Ceil())
	peers := p.peers.Pick(height)

	for _, peer := range peers {
		if _, ok := p.pending[peer.ID]; !ok {
			r.peer = peer
			break
		}
	}

	if r.peer != nil {
		r.Execute()
	}
}

func (p *reqPool) Receive(peerId string, code Cmd) {

}
