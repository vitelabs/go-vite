package net

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"sync"
	"time"
)

const reqCountCacheLimit = 3
const maxBlocksOneTrip = 100
const maxBlocksOneFile = 1000
const expireCheckInterval = 30 * time.Second

var errHasNoSuitablePeer = errors.New("has no suitable peer")

type Params interface {
	p2p.Serializable
	Equal(interface{}) bool
	Ceil() uint64 // the minimal height of snapshotchain
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

type reqParam struct {
	Cmd
	Params
}

type reqCallback func(Cmd, interface{}) (done bool, err error)
type req struct {
	id         uint64
	peer       *Peer
	count      int // Count the number of times the req was requested
	param      *reqParam
	status     reqStatus
	addedAt    time.Time
	timeout    time.Duration
	expiration time.Time
	lock       sync.RWMutex
	callback   reqCallback // when peer get response, call the callback to judge whether req is done or error
	errch      chan error  // if callback encounter error, then send the error to this channel
	done       chan bool   // if callback is executed, send the first result to this channel
}

func newReq(cmd Cmd, params Params, callback reqCallback, timeout time.Duration) *req {
	return &req{
		param: &reqParam{
			cmd,
			params,
		},
		timeout:  timeout,
		callback: callback,
		errch:    make(chan error, 1),
		done:     make(chan bool, 1),
	}
}

func (r *req) Equal(r2 interface{}) bool {
	r3, ok := r2.(*req)
	if !ok {
		return false
	}

	if r.id == r3.id {
		return true
	}

	return r.param.Equal(r3.param)
}

// send the req to Peer immediately
var errNoRequestedPeer = errors.New("request has no matched peer")

func (r *req) Execute() (done bool, err error) {
	r.lock.Lock()

	if r.status == reqPending {
		r.lock.Unlock()
		return
	}
	r.count = 0
	r.status = reqPending

	r.lock.Unlock()

	if r.peer != nil {
		r.expiration = time.Now().Add(r.timeout)
		r.peer.execute(r)
		return <-r.done, <-r.errch
	}

	return false, errNoRequestedPeer
}

func (r *req) Cancel() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.status = reqDone

	if r.peer != nil {
		r.peer.delete(r)
	}
}

const MAX_ID uint64 = ^(uint64(0))
const INIT_ID uint64 = 1

type reqPool struct {
	waiting   map[uint64]*req
	pending   map[uint64]*req
	done      map[uint64]*req
	busyPeers map[string]*req
	mu        sync.RWMutex
	currentID uint64
	peers     *peerSet
	errChan   chan *req
	doneChan  chan *req
	stop      chan struct{}
	log       log15.Logger
}

func NewReqPool(peers *peerSet) *reqPool {
	pool := &reqPool{
		waiting:   make(map[uint64]*req, 100),
		pending:   make(map[uint64]*req, 20),
		done:      make(map[uint64]*req, 100),
		currentID: INIT_ID,
		peers:     peers,
		errChan:   make(chan *req, 1),
		doneChan:  make(chan *req, 1),
		stop:      make(chan struct{}),
		log:       log15.New("module", "net/reqpool"),
	}

	pool.loop()

	return pool
}

func (p *reqPool) loop() {
	ticker := time.NewTicker(expireCheckInterval)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-p.stop:
			break loop
		case req := <-p.doneChan:
			req.status = reqDone
			peerId := req.peer.ID
			delete(p.busyPeers, peerId)
			delete(p.pending, req.id)
			p.done[req.id] = req
		case req := <-p.errChan:
			peerId := req.peer.ID
			delete(p.busyPeers, peerId)
			delete(p.pending, req.id)

			p.Add(req)
		case now := <-ticker.C:
			for _, req := range p.pending {
				if now.After(req.expiration) {
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

func (p *reqPool) Add(r *req) {
	p.mu.Lock()
	defer p.mu.Unlock()

	r.addedAt = time.Now()

	for _, queue := range []map[uint64]*req{p.waiting, p.pending, p.done} {
		for _, req := range queue {
			if req.Equal(r) {
				r.count++
				if r.count >= reqCountCacheLimit {
					p.Execute(r)
				}
				return
			}
		}
	}

	// request at the first time
	if p.currentID == MAX_ID {
		p.currentID = INIT_ID
	} else {
		p.currentID++
	}

	r.id = p.currentID
	r.count = 1
	p.Execute(r)
}

func (p *reqPool) Execute(r *req) {
	delete(p.waiting, r.id)

	peers := p.peers.Pick(r.param.Ceil())

	for _, peer := range peers {
		if _, ok := p.busyPeers[peer.ID]; !ok {
			r.peer = peer
			// set pending
			p.busyPeers[peer.ID] = r
			p.pending[r.id] = r
			break
		}
	}

	done, err := r.Execute()
	if err != nil {
		p.errChan <- r
	} else if done {
		p.doneChan <- r
	}
}
