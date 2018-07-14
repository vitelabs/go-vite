package p2p

import (
	"time"
	"sync"
	"fmt"
)

var pingInterval = 15 * time.Second

const (
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
)

// @section peer error
type DiscReason uint

const (
	DiscRequested DiscReason = iota
	DiscNetworkError
	DiscProtocolError
	DiscUselessPeer
	DiscTooManyPeers
	DiscAlreadyConnected
	DiscIncompatibleVersion
	DiscInvalidIdentity
	DiscQuitting
	DiscUnexpectedIdentity
	DiscSelf
	DiscReadTimeout
	DiscSubprotocolError = 0x10
)

var discReasonToString = [...]string{
	DiscRequested:           "disconnect requested",
	DiscNetworkError:        "network error",
	DiscProtocolError:       "breach of protocol",
	DiscUselessPeer:         "useless peer",
	DiscTooManyPeers:        "too many peers",
	DiscAlreadyConnected:    "already connected",
	DiscIncompatibleVersion: "incompatible p2p protocol version",
	DiscInvalidIdentity:     "invalid node identity",
	DiscQuitting:            "client quitting",
	DiscUnexpectedIdentity:  "unexpected identity",
	DiscSelf:                "connected to self",
	DiscReadTimeout:         "read timeout",
	DiscSubprotocolError:    "subprotocol error",
}

func (d DiscReason) String() string {
	if len(discReasonToString) < int(d) {
		return fmt.Sprintf("unknown disconnect reason %d", d)
	}
	return discReasonToString[d]
}

func (d DiscReason) Error() string {
	return d.String()
}


// @section Peer
type Peer struct {
	ts		*TSConn
	created	time.Time
	wg      sync.WaitGroup
	errch 	chan error
	closed  chan struct{}
	disc    chan DiscReason
	protoMsg chan Msg
}

func NewPeer(ts *TSConn) *Peer {
	return &Peer{
		ts: 		ts,
		errch: 		make(chan error),
		closed:		make(chan struct{}),
		disc: 		make(chan DiscReason),
		protoMsg:	make(chan Msg),
		created: 	time.Now(),
	}
}

func (p *Peer) run() (err error) {
	p.wg.Add(2)
	go p.readLoop()
	go p.pingLoop()

	// Wait for an error or disconnect.
loop:
	for {
		select {
		case err = <-p.errch:
			break loop
		case err = <-p.disc:
			break loop
		}
	}

	close(p.closed)
	p.ts.Close(err)
	p.wg.Wait()
	return err
}

func (p *Peer) ID() NodeID {
	return p.ts.id
}

func (p *Peer) readLoop() {
	defer p.wg.Done()

	for {
		msg, err := p.ts.ReadMsg()
		if err != nil {
			p.errch <- err
			return
		}
		err = p.handleMsg(msg)
		if err != nil {
			p.errch <- err
			return
		}
	}
}

func (p *Peer) pingLoop() {
	defer p.wg.Done()

	timer := time.NewTimer(pingInterval)
	defer timer.Stop()

	for {
		select {
		case <- timer.C:
			if err := Send(p.ts, &Msg{Code: pingMsg}); err != nil {
				p.errch <- err
				return
			}
			timer.Reset(pingInterval)
		case <- p.closed:
			return
		}
	}
}

func (p *Peer) handleMsg(msg Msg) error {
	switch {
	case msg.Code == pingMsg:
		go Send(p.ts, &Msg{Code: pongMsg})
	case msg.Code == pongMsg:
		// ignore
	case msg.Code == discMsg:
		// todo: extract discReason from msg
		var discReason DiscReason
		return discReason
	default:
		// higher protocol
		// todo: use goroutine handle higher message
		// error must write to p.errch
		//p.protoMsg <- msg
	}

	return nil
}

func (p *Peer) Disconnect(reason DiscReason) {
	select {
	case p.disc <- reason:
	case <-p.closed:
	}
}
