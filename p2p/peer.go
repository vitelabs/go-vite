package p2p

import (
	"time"
	"sync"
	"fmt"
	"errors"
	"io"
)

var pingInterval = 15 * time.Second

const (
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
)

// @section peer error
const (
	errInvalidMsgCode = iota
	errInvalidMsg
)

var errorToString = map[int]string{
	errInvalidMsgCode: "invalid message code",
	errInvalidMsg:     "invalid message",
}

type peerError struct {
	code    int
	message string
}

func newPeerError(code int, format string, v ...interface{}) *peerError {
	desc, ok := errorToString[code]
	if !ok {
		panic("invalid error code")
	}
	err := &peerError{code, desc}
	if format != "" {
		err.message += ": " + fmt.Sprintf(format, v...)
	}
	return err
}

func (pe *peerError) Error() string {
	return pe.message
}

var errProtocolReturned = errors.New("protocol returned")

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

func discReasonForError(err error) DiscReason {
	if reason, ok := err.(DiscReason); ok {
		return reason
	}
	if err == errProtocolReturned {
		return DiscQuitting
	}
	peerError, ok := err.(*peerError)
	if ok {
		switch peerError.code {
		case errInvalidMsgCode, errInvalidMsg:
			return DiscProtocolError
		default:
			return DiscSubprotocolError
		}
	}
	return DiscSubprotocolError
}

// @section Peer
type Peer struct {
	ts		*TSConn
	created	time.Time
	wg      sync.WaitGroup
	errch 	chan error
	closed  chan struct{}
	disc    chan DiscReason
	readMsg chan Msg
}

func NewPeer(ts *TSConn) *Peer {
	return &Peer{
		ts: 		ts,
		errch: 		make(chan error),
		closed:		make(chan struct{}),
		disc: 		make(chan DiscReason),
		readMsg:	make(chan Msg),
		created: 	time.Now(),
	}
}

func (p *Peer) run() (err error) {
	var (
		writeStart = make(chan struct{}, 1)
		writeErr   = make(chan error, 1)
		readErr    = make(chan error, 1)
		reason     DiscReason // sent to the peer
	)
	p.wg.Add(2)
	go p.readLoop(readErr)
	go p.pingLoop()

	// Start all protocol handlers.
	writeStart <- struct{}{}

	// Wait for an error or disconnect.
loop:
	for {
		select {
		case err = <-writeErr:
			if err != nil {
				reason = DiscNetworkError
				break loop
			}
			writeStart <- struct{}{}
		case err = <-readErr:
			if r, ok := err.(DiscReason); ok {
				reason = r
			} else {
				reason = DiscNetworkError
			}
			break loop
		case err = <-p.errch:
			reason = discReasonForError(err)
			break loop
		case err = <-p.disc:
			reason = discReasonForError(err)
			break loop
		}
	}

	close(p.closed)
	p.ts.Close(reason)
	p.wg.Wait()
	return err
}

func (p *Peer) ID() NodeID {
	return p.ts.id
}

func (p *Peer) readLoop(cherr chan<- error) {
	defer p.wg.Done()

	for {
		msg, err := p.ts.ReadMsg()
		if err != nil {
			cherr <- err
			return
		}
		err = p.handleMsg(msg)
		if err != nil {
			cherr <- err
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
		// todo extract discReason from msg
	default:
		// higher protocol
		select {
		case p.readMsg <- msg:
			return nil
		case <- p.closed:
			return io.EOF
		}
	}
	return nil
}

func (p *Peer) Disconnect(reason DiscReason) {
	select {
	case p.disc <- reason:
	case <-p.closed:
	}
}
