package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var errRequestTimeout = errors.New("request timeout")
var errPoolStopped = errors.New("pool stopped")

// @section requestPool
const INIT_ID uint64 = 1

type requestPool struct {
	pending map[uint64]Request // has executed, wait for response
	id      uint64             // atomic, unique request id, identically message id
	add     chan Request
	term    chan struct{}
	log     log15.Logger
	wg      sync.WaitGroup
	peers   *peerSet
	fc      *fileClient
	get     chan uint64
	out     chan Request
}

// as message handler
func (p *requestPool) ID() string {
	return "request pool"
}

func (p *requestPool) Cmds() []cmd {
	return []cmd{FileListCode, SubLedgerCode, ExceptionCode}
}

func (p *requestPool) Handle(msg *p2p.Msg, sender *Peer) error {
	if r := p.Get(msg.Id); r != nil {
		r.Handle(p, msg, sender)
	}

	return nil
}

func newRequestPool(peers *peerSet, fc *fileClient) *requestPool {
	pool := &requestPool{
		pending: make(map[uint64]Request, 20),
		id:      INIT_ID,
		add:     make(chan Request, 10),
		log:     log15.New("module", "net/reqpool"),
		peers:   peers,
		fc:      fc,
		get:     make(chan uint64),
		out:     make(chan Request),
	}

	return pool
}

func (p *requestPool) start() {
	p.term = make(chan struct{})

	p.wg.Add(1)
	common.Go(p.loop)
}

func (p *requestPool) stop() {
	if p.term == nil {
		return
	}

	select {
	case <-p.term:
	default:
		close(p.term)
		p.wg.Wait()
	}
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

	ticker := time.NewTicker(5 * time.Second)
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
				r.Catch(errMissingPeer)
			} else {
				r.Run(p)
				p.pending[r.ID()] = r
			}

		case id := <-p.get:
			if r, ok := p.pending[id]; ok {
				p.out <- r
			} else {
				p.out <- nil
			}

		case <-ticker.C:
			for id, r := range p.pending {
				state := r.State()

				if state == reqDone || state == reqError {
					delete(p.pending, r.ID())
				} else if r.Expired() && state == reqPending {
					p.log.Error(fmt.Sprintf("retry request %d, error: %v", id, errRequestTimeout))
					p.retry(r)
				}
			}
		}
	}

	// clean job
	for i := 0; i < len(p.add); i++ {
		r := <-p.add
		r.Catch(errPoolStopped)
	}

	for id, r := range p.pending {
		r.Catch(errPoolStopped)
		delete(p.pending, id)
	}
}

func (p *requestPool) Add(r Request) {
	select {
	case <-p.term:
		r.Catch(errPoolStopped)
		return
	case p.add <- r:
	}
}

func (p *requestPool) MsgID() uint64 {
	return atomic.AddUint64(&p.id, 1)
}

func (p *requestPool) Retry(id uint64, err error) {
	p.log.Error(fmt.Sprintf("retry request %d, error: %v", id, err))

	if r := p.Get(id); r != nil && r.State() == reqPending {
		p.retry(r)
	}
}

func (p *requestPool) retry(r Request) {
	if r == nil {
		return
	}

	_, to := r.Band()
	if peer := p.pickPeer(to); peer != nil {
		r.SetPeer(peer)
		r.Run(p)
	} else {
		r.Catch(errMissingPeer)
	}
}

func (p *requestPool) FC() *fileClient {
	return p.fc
}

func (p *requestPool) Get(id uint64) Request {
	p.get <- id
	return <-p.out
}
