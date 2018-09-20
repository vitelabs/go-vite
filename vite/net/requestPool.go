package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"time"
)

const reqCountCacheLimit = 3
const maxBlocksOneTrip = 100
const maxBlocksOneFile = 1000
const expireCheckInterval = 30 * time.Second

var errHasNoSuitablePeer = errors.New("has no suitable peer")
var errNoRequestedPeer = errors.New("request has no matched peer")

type requestParam interface {
	p2p.Serializable
	Equal(interface{}) bool
	Ceil() uint64 // peer who`s height taller than this value can handle it
	Handle(*response) error
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

// request
type response struct {
	cmd
	msgId  uint64
	sender *Peer
	msg    p2p.Serializable
}

type request struct {
	cmd        // message type
	id         uint64
	peer       *Peer
	count      int // Count the number of times the req was requested
	param      requestParam
	timeout    time.Duration
	expiration time.Time
	next       *request // linked list
	done       chan bool
}

func newRequest(cmd cmd, param requestParam, timeout time.Duration) *request {
	return &request{
		cmd:     cmd,
		param:   param,
		timeout: timeout,
	}
}

func (r *request) equal(r2 *request) bool {
	if r.id == r2.id {
		return true
	}

	return r.cmd == r2.cmd && r.param.Equal(r2.param)
}

// done means request has fulfilled, can be removed from pending queue,
// error means request catch error, should be retry
func (r *request) handle(res *response) (done bool, err error) {
	//if r.cmd
}

func (r *request) cancel() {
	if r.peer != nil {
		r.peer.delRequest(r)
	}
}

// request queue
type requestQueue struct {
	head *request // the head(null) item
	tail *request // the last item
	size int
}

func newReqQueue() *requestQueue {
	req := new(request)

	return &requestQueue{
		head: req,
		tail: req,
		size: 0,
	}
}

// append to tail
func (q *requestQueue) append(r *request) {
	q.tail.next = r
	q.tail = r
	q.size++
}

// append r after target, if can`t find target, then append to tail
func (q *requestQueue) add(r *request, target *request) {
	for item := q.head; item.next != nil; item = item.next {
		if item.equal(target) {
			r.next = target.next
			target.next = r
			q.size++
			return
		}
	}

	q.append(r)
}

// cont indicate whether continue traverse
func (q *requestQueue) traverse(fn func(prev, current *request) (cont bool)) {
	for prev, current := q.head, q.head.next; current != nil; prev, current = current, current.next {
		if !fn(prev, current) {
			break
		}
	}
}

// pick the first item
func (q *requestQueue) shift() (item *request) {
	item = q.head.next
	if item != nil {
		q.head.next = item.next
		q.size--
	}

	// if item is the last item, then redirect tail to head
	if item == q.tail {
		q.tail = q.head
	}

	return
}

// @section requestPool
const MAX_ID = ^(uint64(0)) - 1
const INIT_ID uint64 = 1

//GetSubLedgerCode
//GetSnapshotBlockHeadersCode
//GetSnapshotBlockBodiesCode
//GetSnapshotBlocksCode
//GetSnapshotBlocksByHashCode
//GetAccountBlocksCode
//GetAccountBlocksByHashCode

func isSnapshotRequest(cmd cmd) bool {
	switch cmd {
	case GetSubLedgerCode:
	case GetSnapshotBlocksCode:
	case GetSnapshotBlocksByHashCode:
	default:
		return false

	}
	return true
}

type requestPool struct {
	snapshotQueue *requestQueue
	accountQueue  map[types.Address]*requestQueue
	pending       map[uint64]*request // has executed, wait for response
	done          map[uint64]*request
	busyPeers     map[string]struct{} // mark peers whether has request for handle
	currentID     uint64              // unique id
	peers         *peerSet
	errChan       chan *request
	doneChan      chan *request
	term          chan struct{}
	log           log15.Logger
}

func newRequestPool(peers *peerSet) *requestPool {
	pool := &requestPool{
		snapshotQueue: new(requestQueue),
		accountQueue:  make(map[types.Address]*requestQueue, 100),
		pending:       make(map[uint64]*request, 20),
		done:          make(map[uint64]*request, 100),
		currentID:     INIT_ID,
		peers:         peers,
		errChan:       make(chan *request, 1),
		doneChan:      make(chan *request, 1),
		term:          make(chan struct{}),
		log:           log15.New("module", "net/reqpool"),
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

	for {
		select {
		case <-p.term:
			goto END
		case req := <-p.doneChan:
			p.fulfill(req)
		case req := <-p.errChan:
			p.retry(req)
		case now := <-ticker.C:
			for _, req := range p.pending {
				if now.After(req.expiration) {
					select {
					case <-p.term:
					case req.peer.reqErr <- p2p.DiscResponseTimeout:
					default:
					}

					p.log.Error(fmt.Sprintf("request %s/%d wait from %s timeout", req.cmd, req.id, req.peer.ID))
					p.retry(req)
				}
			}
		default:
			r := p.snapshotQueue.shift()
			if err := p.execute(r); err != nil {
				p.retry(r)
			}
		}
	}

END:
	for k, r := range p.pending {
		delete(p.pending, k)
		r.cancel()
	}
}

func (p *requestPool) retry(r *request) {
	peerId := r.peer.ID
	delete(p.busyPeers, peerId)
	delete(p.pending, r.id)

	p.execute(r)
}

func (p *requestPool) fulfill(r *request) {
	peerId := r.peer.ID
	delete(p.busyPeers, peerId)
	delete(p.pending, r.id)
	p.done[r.id] = r
}

// will have two param for now
// Segment for subLedger
// AccountSegment for single accountChain
func (p *requestPool) add(r *request) {
	if p.currentID == MAX_ID {
		p.currentID = INIT_ID
	} else {
		p.currentID++
	}

	r.id = p.currentID
	r.count = 1

	// todo split long chain request to small chunk
	if isSnapshotRequest(r.cmd) {
		p.snapshotQueue.append(r)
	} else {
		as, _ := r.param.(*AccountSegment)

		queue, ok := p.accountQueue[as.Address]
		if !ok {
			q := newReqQueue()
			p.accountQueue[as.Address] = q
			queue = q
		}
		queue.append(r)
	}
}

func (p *requestPool) execute(r *request) error {
	p.pickPeer(r)

	if r.peer == nil {
		// todo peer maybe nil
	}

	// set status
	p.busyPeers[r.peer.ID] = struct{}{}
	p.pending[r.id] = r

	return r.peer.Send(r.cmd, r.param)
}

func (p *requestPool) pickPeer(r *request) {
	if r.peer != nil {
		return
	}

	peers := p.peers.Pick(r.param.Ceil())

	for _, peer := range peers {
		if _, ok := p.busyPeers[peer.ID]; !ok {
			r.peer = peer
			return
		}
	}
}

func (p *requestPool) handleResponse(res *response) {
	for id, req := range p.pending {
		if req.peer == res.sender && id == res.msgId {
			if err := req.param.Handle(res); err == nil {
				p.fulfill(req)
			}
		}
	}
}
