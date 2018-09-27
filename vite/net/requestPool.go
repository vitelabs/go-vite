package net

import (
	"fmt"
	"github.com/pkg/errors"
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

type reqState byte

const (
	reqWaiting reqState = iota
	reqPending
	reqDone
	reqError
)

var reqStatus = [...]string{
	reqWaiting: "waiting",
	reqPending: "pending",
	reqDone:    "done",
	reqError: "error",
}

func (s reqState) String() string {
	return reqStatus[s]
}

// @section requestPool
const MAX_ID = ^(uint64(0)) - 1
const INIT_ID uint64 = 1

type requestPool struct {
	queue *list.List
	pending       map[uint64]Request // has executed, wait for response
	busyPeers     map[string]uint64 // key: string, value: pending request id
	currentID     uint64              // unique id
	peers         *peerSet
	add chan Request
	retryChan         chan uint64
	doneChan      chan uint64
	term          chan struct{}
	log           log15.Logger
	peerEvent chan *peerEvent
}

func newRequestPool(peers *peerSet) *requestPool {
	pool := &requestPool{
		queue: list.New(),
		pending:       make(map[uint64]Request, 20),
		busyPeers: make(map[string]uint64),
		currentID:     INIT_ID,
		peers:         peers,
		add: make(chan Request, 100),
		retryChan:         make(chan uint64, 100),
		doneChan:      make(chan uint64, 100),
		term:          make(chan struct{}),
		log:           log15.New("module", "net/reqpool"),
		peerEvent: make(chan *peerEvent),
	}

	peers.Sub(pool.peerEvent)

	go pool.loop()

	return pool
}

func (p *requestPool) stop() {
	select {
	case <-p.term:
	default:
		close(p.term)
		p.peers.Unsub(p.peerEvent)
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
		case r := <- p.add:
			if p.currentID == MAX_ID {
				p.currentID = INIT_ID
			} else {
				p.currentID++
			}

			r.setId(p.currentID)
			p.queue.Append(r)
		case e := <- p.peerEvent:
			if e.code == delPeer {
				peerId := e.peer.ID
				if id, ok := p.busyPeers[peerId]; ok {
					p.retryChan <- id
				}
			}
		case id := <-p.doneChan:
			r := p.pending[id]
			if r != nil {
				peerId := r.peer().ID
				delete(p.busyPeers, peerId)
				delete(p.pending, r.id())

				if pid := r.pid(); pid != 0 {
					parent := p.pending[pid]
					parent.childDone(r)
				}
			}
		case id := <-p.retryChan:
			r := p.pending[id]
			if r != nil {
				peerId := r.peer().ID
				delete(p.busyPeers, peerId)
				delete(p.pending, r.id())

				if !r.run(p.peers, true) {
					p.Add(r)
				}
			}
		case <-ticker.C:
			for _, r := range p.pending {
				if r.expired() {
					select {
					case <-p.term:
					default:
					}

					p.log.Error(fmt.Sprintf("request %d wait from %s timeout", r.id, r.peer().ID))
					p.Retry(r)
				}
			}
		default:
			r := p.queue.Shift()
			if r != nil {
				req, _ := r.(Request)
				p.do(req);
			}
		}
	}
}

func (p *requestPool) Add(r Request) {
	if p.currentID == MAX_ID {
		p.currentID = INIT_ID
	} else {
		p.currentID++
	}

	r.setId(p.currentID)
	p.queue.Append(r)
}

func (p *requestPool) Retry(r Request) {
	p.retryChan <- r.id()
}

func (p *requestPool) Done(r Request) {
	p.doneChan <- r.id()
}

func (p *requestPool) receive(cmd cmd, id uint64, data p2p.Serializable, peer *Peer) {
	if r, ok := p.pending[id]; ok {
		r.handle(cmd, data, peer)
	}
}

func (p *requestPool) do(r Request) {
	if r.run(p.peers, false) {
		p.busyPeers[r.peer().ID] = r.id()
		p.pending[r.id()] = r
	}
}
