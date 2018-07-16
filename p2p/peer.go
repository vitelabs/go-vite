package p2p

import (
	"time"
	"sync"
	"fmt"
	"encoding/binary"
)

var pingInterval = 15 * time.Second

const (
	baseProtocolBand = 16
	discMsg      = 1
	pingMsg      = 2
	pongMsg      = 3
	handshakeMsg = 4
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

func errTodiscReason(err error) DiscReason {
	if reason, ok := err.(DiscReason); ok {
		return reason
	}
	return DiscSubprotocolError
}


// @section Peer
type Peer struct {
	ts		*TSConn
	created	time.Time
	wg      sync.WaitGroup
	Errch 	chan error
	Closed  chan struct{}
	disc    chan DiscReason
	ProtoMsg chan Msg
}

func NewPeer(ts *TSConn) *Peer {
	return &Peer{
		ts: 		ts,
		Errch: 		make(chan error),
		Closed:		make(chan struct{}),
		disc: 		make(chan DiscReason),
		ProtoMsg:	make(chan Msg),
		created: 	time.Now(),
	}
}

func (p *Peer) run(protoHandler func(*Peer)) (err error) {
	p.wg.Add(2)
	go p.readLoop()
	go p.pingLoop()

	// higher protocol
	if protoHandler != nil {
		p.wg.Add(1)
		go func() {
			protoHandler(p)
			p.wg.Done()
		}()
	}

	// Wait for an error or disconnect.
loop:
	for {
		select {
		case err = <-p.Errch:
			break loop
		case err = <-p.disc:
			break loop
		}
	}

	close(p.Closed)
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
			p.Errch <- err
			return
		}
		go p.handleMsg(msg)
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
				p.Errch <- err
				return
			}
			timer.Reset(pingInterval)
		case <- p.Closed:
			return
		}
	}
}

func (p *Peer) handleMsg(msg Msg) {
	switch {
	case msg.Code == pingMsg:
		go Send(p.ts, &Msg{Code: pongMsg})
	case msg.Code == discMsg:
		discReason := binary.BigEndian.Uint64(msg.Payload)
		p.Errch <- DiscReason(discReason)
	case msg.Code < baseProtocolBand:
		// ignore
	default:
		// higher protocol
		p.ProtoMsg <- msg
	}
}

func (p *Peer) Disconnect(reason DiscReason) {
	select {
	case p.disc <- reason:
	case <-p.Closed:
	}
}
