package p2p

import (
	"time"
	"sync"
	"fmt"
	"encoding/binary"
	"log"
)

var pingInterval = 15 * time.Second

const (
	baseProtocolBand = 16
	discMsg      = 1
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
	DiscTooManyPassivePeers
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
	DiscTooManyPassivePeers: "too many passive peers",
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
	TS		*TSConn
	created	time.Time
	wg      sync.WaitGroup
	Errch 	chan error
	Closed  chan struct{}
	disc    chan DiscReason
	ProtoMsg chan Msg
	protoHandler peerHandler
}

func NewPeer(ts *TSConn) *Peer {
	return &Peer{
		TS: 		ts,
		Errch: 		make(chan error),
		Closed:		make(chan struct{}),
		disc: 		make(chan DiscReason),
		ProtoMsg:	make(chan Msg),
		created: 	time.Now(),
	}
}

func (p *Peer) run(protoHandler peerHandler) (err error) {
	log.Printf("peer %s run\n", p.ID())

	p.wg.Add(1)
	go p.readLoop()

	// higher protocol
	if protoHandler != nil {
		p.protoHandler = protoHandler
		p.wg.Add(1)
		go func() {
			log.Printf("proto handle peer %s\n", p.ID())
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
	p.TS.Close(err)
	p.wg.Wait()
	return err
}

func (p *Peer) ID() NodeID {
	return p.TS.id
}

func (p *Peer) readLoop() {
	defer p.wg.Done()

	for {
		select {
		case <- p.Closed:
			return
		default:
			msg, err := p.TS.ReadMsg()
			if err != nil {
				log.Printf("peer %s read error: %v\n", p.ID(), err)
				p.Errch <- err
				return
			}

			p.handleMsg(msg)
		}
	}
}

func (p *Peer) handleMsg(msg Msg) {
	log.Printf("tcp receive msg %d from %s\n", msg.Code, p.ID())

	switch {
	case msg.Code == discMsg:
		discReason := binary.BigEndian.Uint64(msg.Payload)
		log.Printf("disconnect with peer %s: %s\n", p.ID(), DiscReason(discReason))
		p.Errch <- DiscReason(discReason)
	case msg.Code < baseProtocolBand:
		// ignore
	default:
		// higher protocol
		if p.protoHandler != nil {
			p.ProtoMsg <- msg
		} else {
			p.Errch <- fmt.Errorf("cannot handle msg %d from %s, missing protoHandler\n", p.ID(), msg.Code)
		}
	}
}

func (p *Peer) Disconnect(reason DiscReason) {
	log.Printf("disconnect with peer %s\n", p.ID())

	select {
	case p.disc <- reason:
	case <-p.Closed:
	}
}
