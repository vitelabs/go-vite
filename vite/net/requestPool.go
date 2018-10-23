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
	pending *sync.Map // has executed, wait for response
	id      uint64    // atomic, unique request id, identically message id
	term    chan struct{}
	log     log15.Logger
	wg      sync.WaitGroup
	peers   *peerSet
	fc      *fileClient
	running chan struct{} // restrict concurrency
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
		pending: new(sync.Map),
		id:      INIT_ID,
		log:     log15.New("module", "net/reqpool"),
		peers:   peers,
		fc:      fc,
		running: make(chan struct{}, 100),
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

		case <-ticker.C:
			p.pending.Range(func(key, value interface{}) bool {
				id, r := key.(uint64), value.(Request)
				state := r.State()

				if state == reqDone || state == reqError {
					p.pending.Delete(key)
					<-p.running
				} else if r.Expired() && state == reqPending {
					p.log.Error(fmt.Sprintf("retry request %d, error: %v", id, errRequestTimeout))
					p.retry(r)
				}

				return true
			})
		}
	}

	// clean job
	p.pending.Range(func(key, value interface{}) bool {
		r := value.(Request)
		r.Catch(errPoolStopped)
		p.pending.Delete(key)

		return true
	})
}

func (p *requestPool) Add(r Request) {
	select {
	case <-p.term:
		r.Catch(errPoolStopped)
		return
	case p.running <- struct{}{}:
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
			p.pending.Store(r.ID(), r)
		}
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
	if v, ok := p.pending.Load(id); ok {
		if r, ok := v.(Request); ok {
			return r
		}
		return nil
	}

	return nil
}

func (p *requestPool) Del(id uint64) {
	if _, ok := p.pending.Load(id); ok {
		p.pending.Delete(id)
		<-p.running
	}
}
