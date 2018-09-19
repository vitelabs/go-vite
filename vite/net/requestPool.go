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
var errNoRequestedPeer = errors.New("request has no matched peer")

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
	cmd
	Params
}

type request struct {
	cmd
	id         uint64
	peer       *Peer
	count      int // Count the number of times the req was requested
	param      Params
	timeout    time.Duration
	expiration time.Time
	next       *request // linked list
}

func newReq(cmd cmd, params Params, timeout time.Duration) *request {
	return &request{
		param: &reqParam{
			cmd,
			params,
		},
		timeout: timeout,
	}
}

func (r *request) Equal(r2 *request) bool {
	if r.id == r2.id {
		return true
	}

	return r.param.Equal(r2.param)
}

func (r *request) Cancel() {
	if r.peer != nil {
		r.peer.delete(r)
	}
}

// @section requestPool
const MAX_ID = ^(uint64(0))
const INIT_ID uint64 = 1

type requestPool struct {
	queue     *request
	pending   map[uint64]*request // has executed, wait for response
	done      map[uint64]*request
	busyPeers map[string]struct{} // mark peers whether has request for handle
	currentID uint64              // unique id
	peers     *peerSet
	errChan   chan *request
	doneChan  chan *request
	term      chan struct{}
	log       log15.Logger
}

func newRequestPool(peers *peerSet) *requestPool {
	pool := &requestPool{
		queue:     &request{},
		pending:   make(map[uint64]*request, 20),
		done:      make(map[uint64]*request, 100),
		currentID: INIT_ID,
		peers:     peers,
		errChan:   make(chan *request, 1),
		doneChan:  make(chan *request, 1),
		term:      make(chan struct{}),
		log:       log15.New("module", "net/reqpool"),
	}

	go pool.loop()

	return pool
}

func (p *requestPool) stop() {
	select {
	case <-p.term:
	default:
		close(p.term)
	}
}

func (p *requestPool) loop() {
	ticker := time.NewTicker(expireCheckInterval)
	defer ticker.Stop()

	var expiration time.Time

	for {
		select {
		case <-p.term:
			goto END
		case req := <-p.doneChan:
			peerId := req.peer.ID
			delete(p.busyPeers, peerId)
			delete(p.pending, req.id)
			p.done[req.id] = req
		case req := <-p.errChan:
			p.Retry(req)
		case now := <-ticker.C:
			for _, req := range p.pending {
				if now.After(req.expiration) {
					p.log.Error("req timeout")
					p.errChan <- req
				}
			}
		}
	}
END:
	for k, r := range p.pending {
		delete(p.pending, k)
		r.Cancel()
	}
}

func (p *requestPool) Retry(req *req) {
	peerId := req.peer.ID
	delete(p.busyPeers, peerId)
	delete(p.pending, req.id)

	go p.Execute(req)
}

func (p *requestPool) Add(r *req) {
	p.mu.Lock()
	defer p.mu.Unlock()

	r.addedAt = time.Now()

	for _, queue := range []map[uint64]*req{p.waiting, p.pending, p.done} {
		for _, req := range queue {
			if req.Equal(r) {
				r.count++
				if r.count >= reqCountCacheLimit {
					go p.Execute(r)
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
	//
	go p.Execute(r)
}

func (p *requestPool) Execute(r *req) {
	delete(p.waiting, r.id)

	// if req has no given peer, then choose one
	if r.peer == nil {
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
	}

	done, err := r.Execute()
	if err != nil {
		p.errChan <- r
	} else if done {
		p.doneChan <- r
	}
}
