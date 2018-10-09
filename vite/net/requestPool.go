package net

import (
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
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

var errNoPeer = errors.New("no peer")
var errHasNoSuitablePeer = errors.New("has no suitable peer")
var errNoRequestedPeer = errors.New("request has no matched peer")
var errRequestTimeout = errors.New("request timeout")
var errPoolStopped = errors.New("pool stopped")

// @section requestPool
const INIT_ID uint64 = 1

type requestPool struct {
	pending map[uint64]Request // has executed, wait for response
	id      uint64             // atomic, unique request id, identically message id
	add     chan Request
	retry   chan uint64
	del     chan uint64
	term    chan struct{}
	log     log15.Logger
	wg      sync.WaitGroup
	ctx     *context
}

// as message handler
func (p *requestPool) ID() string {
	return "request pool"
}

func (p *requestPool) Cmds() []cmd {
	return []cmd{FileListCode, SubLedgerCode, SnapshotBlocksCode, AccountBlocksCode, ExceptionCode}
}

func (p *requestPool) Handle(msg *p2p.Msg, sender *Peer) error {
	for id, r := range p.pending {
		if id == msg.Id {
			// todo goroutine
			r.Handle(p.ctx, msg, sender)
		}
	}

	return nil
}

func newRequestPool() *requestPool {
	pool := &requestPool{
		pending: make(map[uint64]Request, 20),
		id:      INIT_ID,
		add:     make(chan Request, 10),
		retry:   make(chan uint64, 5),
		del:     make(chan uint64, 10),
		term:    make(chan struct{}),
		log:     log15.New("module", "net/reqpool"),
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

func (p *requestPool) loop() {
	defer p.wg.Done()

	expireCheckInterval := 10 * time.Second
	ticker := time.NewTicker(expireCheckInterval)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-p.term:
			break loop

		case r := <-p.add:
			// todo goroutine
			r.Run()
			p.pending[r.ID()] = r

		case id := <-p.retry:
			if r, ok := p.pending[id]; ok {
				// todo goroutine
				r.Run()
			}

		case <-ticker.C:
			for _, r := range p.pending {
				if r.Expired() {
					r.Done(errRequestTimeout)
				}
			}

		case id := <-p.del:
			delete(p.pending, id)
		}
	}

	// clean job
	for i := 0; i < len(p.add); i++ {
		r := <-p.add
		r.Done(errPoolStopped)
	}

	for i := 0; i < len(p.retry); i++ {
		id := <-p.retry
		if r, ok := p.pending[id]; ok {
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

func (p *requestPool) Add(r Request) bool {
	select {
	case p.add <- r:
		return true
	default:
		p.log.Error("can`t add request")
		return false
	}
}

func (p *requestPool) Del(id uint64) bool {
	select {
	case p.del <- id:
		return true
	default:
		p.log.Error("can`t del request")
		return false
	}
}

func (p *requestPool) MsgID() uint64 {
	return atomic.AddUint64(&p.id, 1)
}

func (p *requestPool) Retry(id uint64) bool {
	select {
	case p.retry <- id:
		return true
	default:
		p.log.Error("can`t Retry request")
		return false
	}
}
