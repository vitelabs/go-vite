package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

type RequestPool interface {
	Add(r Request) bool // can add request or not
	Retry(r uint64) bool
	Del(r uint64) bool
	MsgID() uint64
}

var errRequestTimeout = errors.New("request timeout")
var errPoolStopped = errors.New("pool stopped")

// @section requestPool
const INIT_ID uint64 = 1

type requestPool struct {
	pending map[uint64]Request // has executed, wait for response
	id      uint64             // atomic, unique request id, identically message id
	add     chan Request
	retry   chan *reqEvent
	del     chan uint64
	term    chan struct{}
	log     log15.Logger
	wg      sync.WaitGroup
	peers   *peerSet
	fc      *fileClient
}

// as message handler
func (p *requestPool) ID() string {
	return "request pool"
}

func (p *requestPool) Cmds() []cmd {
	return []cmd{FileListCode, SubLedgerCode, ExceptionCode}
}

func (p *requestPool) Handle(msg *p2p.Msg, sender *Peer) error {
	for id, r := range p.pending {
		if id == msg.Id {
			go r.Handle(p, msg, sender)
			return nil
		}
	}

	return nil
}

func newRequestPool(peers *peerSet, fc *fileClient) *requestPool {
	pool := &requestPool{
		pending: make(map[uint64]Request, 20),
		id:      INIT_ID,
		add:     make(chan Request, 10),
		retry:   make(chan *reqEvent, 1),
		del:     make(chan uint64, 1),
		term:    make(chan struct{}),
		log:     log15.New("module", "net/reqpool"),
		peers:   peers,
		fc:      fc,
	}

	pool.wg.Add(1)
	go pool.loop()

	return pool
}

func (p *requestPool) stop() {
	select {
	case <-p.term:
	default:
		close(p.term)
		p.wg.Wait()
	}
}

type reqEvent struct {
	id  uint64
	err error
}

func (p *requestPool) pickPeer(height uint64) (peer *Peer) {
	peers := p.peers.Pick(height)
	n := len(peers)
	if n > 0 {
		peer = peers[rand.Intn(n)]
	}

	return peer
}

func (p *requestPool) loop() {
	defer p.wg.Done()

	expireCheckInterval := 30 * time.Second
	ticker := time.NewTicker(expireCheckInterval)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-p.term:
			break loop

		case r := <-p.add:
			if r.ID() == 0 {
				r.SetID(p.MsgID())
			}

			if r.Peer() == nil {
				_, to := r.Band()
				if peer := p.pickPeer(to); peer != nil {
					r.SetPeer(peer)
				}
			}

			if r.Peer() == nil {
				r.Done(errMissingPeer)
			} else {
				r.Run(p)
				p.pending[r.ID()] = r
			}

		case e := <-p.retry:
			if r, ok := p.pending[e.id]; ok {
				select {
				case r.Peer().errch <- e.err:
				default:
				}

				_, to := r.Band()
				if peer := p.pickPeer(to); peer != nil {
					r.SetPeer(peer)
					r.Run(p)
				} else {
					r.Done(errMissingPeer)
				}
			}

		case <-ticker.C:
			for _, r := range p.pending {
				state := r.State()

				if state == reqDone || state == reqError {
					delete(p.pending, r.ID())
				} else if r.Expired() && state == reqPending {
					r.Done(errRequestTimeout)
				}
			}
		}
	}

	// clean job
	for i := 0; i < len(p.add); i++ {
		r := <-p.add
		r.Done(errPoolStopped)
	}

	for i := 0; i < len(p.retry); i++ {
		e := <-p.retry
		if r, ok := p.pending[e.id]; ok {
			r.Done(errPoolStopped)
		}
	}

	for i := 0; i < len(p.retry); i++ {
		id := <-p.del
		delete(p.pending, id)
	}

	for _, r := range p.pending {
		r.Done(errPoolStopped)
	}
}

func (p *requestPool) Add(r Request) {
	select {
	case <-p.term:
		return
	case p.add <- r:
	}
}

func (p *requestPool) MsgID() uint64 {
	return atomic.AddUint64(&p.id, 1)
}

func (p *requestPool) Retry(id uint64, err error) {
	p.log.Error(fmt.Sprintf("retry request %d, error: %v", id, err))
	p.retry <- &reqEvent{id, err}
}

func (p *requestPool) FC() *fileClient {
	return p.fc
}
