package p2p

import (
	"encoding/binary"
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p/discovery"
	"sync"
	"time"
)

const (
	baseProtocolBand = 16
	discMsg          = 1
	handshakeMsg     = 4
)

// @section Peer
type Peer struct {
	TS           *TSConn
	created      time.Time
	wg           sync.WaitGroup
	Errch        chan error
	Closed       chan struct{}
	disc         chan DiscReason
	ProtoMsg     chan Msg
	protoHandler peerHandler
	log          log15.Logger
}

func NewPeer(ts *TSConn) *Peer {
	return &Peer{
		TS:       ts,
		Errch:    make(chan error, 1),
		Closed:   make(chan struct{}),
		disc:     make(chan DiscReason),
		ProtoMsg: make(chan Msg),
		created:  time.Now(),
		log:      log15.New("module", "p2p/peer"),
	}
}

func (p *Peer) run(protoHandler peerHandler) (err error) {
	p.log.Info("run peer", "ID", p.ID().Brief())

	p.wg.Add(1)
	go p.readLoop()

	// higher protocol
	if protoHandler != nil {
		p.protoHandler = protoHandler
		p.wg.Add(1)
		go func() {
			p.log.Info("proto handle peer", "ID", p.ID().Brief())
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

func (p *Peer) ID() discovery.NodeID {
	return p.TS.id
}

func (p *Peer) readLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.Closed:
			return
		default:
			msg, err := p.TS.ReadMsg()
			if err != nil {
				p.log.Error("peer read error", "ID", p.ID().String(), "error", err)
				p.Errch <- err
				return
			}

			p.handleMsg(msg)
		}
	}
}

func (p *Peer) handleMsg(msg Msg) {
	p.log.Info("peer handle msg", "code", msg.Code, "from", p.ID().String())

	switch {
	case msg.Code == discMsg:
		discReason := binary.BigEndian.Uint64(msg.Payload)
		p.log.Info("disconnect with peer", "ID", p.ID().String(), "reason", DiscReason(discReason))
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
	p.log.Info("disconnect with peer", "ID", p.ID().String())

	select {
	case p.disc <- reason:
	case <-p.Closed:
	}
}
