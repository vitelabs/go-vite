package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/list"
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
//type Request interface {
//	Perform(p *Peer)
//	Cancel()
//	Handle(msg *p2p.Msg, p *Peer) bool
//	Expired() bool
//	Msg() *p2p.Msg
//}

type request struct {
	peer       *Peer
	count      int // Count the number of times the req was requested
	cmd        // message type
	id         uint64	// message id
	param      requestParam
	timeout    time.Duration
	expiration time.Time
}

func (r *request) equal(r2 *request) bool {
	if r.id == r2.id {
		return true
	}

	return r.cmd == r2.cmd && r.param.Equal(r2.param)
}

// means request has fulfilled, can be removed from pending queue
func (r *request) Handle(msg *p2p.Msg) bool {
	if msg.Id != r.id {
		return false
	}

}

func (r *request) Cancel() {
	if r.peer != nil {
		r.peer.delRequest(r)
	}
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
	snapshotQueue *list.List
	accountQueue  map[types.Address]*list.List
	pending       map[uint64]*request // has executed, wait for response
	done          map[uint64]*request
	busyPeers     map[string]struct{} // mark peers whether has request for handle
	currentID     uint64              // unique id
	peers         *peerSet
	retryChan         chan *request
	doneChan      chan *request
	term          chan struct{}
	log           log15.Logger
}

func newRequestPool(peers *peerSet) *requestPool {
	pool := &requestPool{
		snapshotQueue: list.New(),
		accountQueue:  make(map[types.Address]*list.List, 100),
		pending:       make(map[uint64]*request, 20),
		done:          make(map[uint64]*request, 100),
		currentID:     INIT_ID,
		peers:         peers,
		retryChan:         make(chan *request, 1),
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

	loop:
	for {
		select {
		case <-p.term:
			break loop
		case req := <-p.doneChan:
			p.fulfill(req)
		case req := <-p.retryChan:
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
			r := p.snapshotQueue.Shift()
			if r != nil {
				req, _ := r.(*request)
				p.do(req);
			}
		}
	}

	for k, r := range p.pending {
		delete(p.pending, k)
		r.Cancel()
	}
}

func (p *requestPool) retry(r *request) {
	peerId := r.peer.ID
	delete(p.busyPeers, peerId)
	delete(p.pending, r.id)

	p.do(r)
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
		p.snapshotQueue.Append(r)
	} else {
		as, _ := r.param.(*AccountSegment)

		queue, ok := p.accountQueue[as.Address]
		if !ok {
			queue = list.New()
			p.accountQueue[as.Address] = queue
		}
		queue.Append(r)
	}
}

func (p *requestPool) do(r *request) {
	p.pickPeer(r)

	if r.peer == nil {
		return
	}

	// set status
	p.busyPeers[r.peer.ID] = struct{}{}
	p.pending[r.id] = r
	r.peer.doRequest(r)
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
