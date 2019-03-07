/*
 * Copyright 2019 The go-vite Authors
 * This file is part of the go-vite library.
 *
 * The go-vite library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The go-vite library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with the go-vite library. If not, see <http://www.gnu.org/licenses/>.
 */

package p2p

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"
)

var errPeerAlreadyRunning = errors.New("peer is already running")
var errPeerNotRunning = errors.New("peer is not running")
var errPeerWriteBusy = errors.New("peer is busy")

type connFlag byte

const (
	outbound  connFlag = 0
	inbound   connFlag = 1
	outstatic connFlag = 2
	instatic  connFlag = 3
	prior     connFlag = 4 // will not be limit by maxPeers & maxInbound
)

var flagStr = [...]string{
	outbound:  "outbound",
	inbound:   "inbound",
	outstatic: "outstatic",
	instatic:  "instatic",
	prior:     "prior",
}

func (f connFlag) is(f2 connFlag) bool {
	return f&f2 > 0
}

func (f connFlag) String() (s string) {
	if int(f) < len(flagStr) {
		return flagStr[f]
	}

	return "unknown"
}

type PeerInfo struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Version   int           `json:"version"`
	Protocols []string      `json:"protocols"`
	Address   string        `json:"address"`
	Flag      string        `json:"flag"`
	CreateAt  string        `json:"createAt"`
	State     []interface{} `json:"state"`
}

const peerReadMsgBufferSize = 10
const peerWriteMsgBufferSize = 10

type peerMux struct {
	codec      Codec
	id         string
	name       string
	version    int
	flag       connFlag
	createAt   time.Time
	protoMap   map[ProtocolID]Protocol
	running    int32
	wg         sync.WaitGroup
	readQueue  chan Msg
	writeQueue chan Msg
	errChan    chan error
	log        log15.Logger
}

// WriteMsg will put msg into queue, then write asynchronously
func (p *peerMux) WriteMsg(msg Msg) (err error) {
	if atomic.LoadInt32(&p.running) == 0 {
		return errPeerNotRunning
	}

	select {
	case p.writeQueue <- msg:
		return nil
	default:
		return errPeerWriteBusy
	}
}

// String return `id@address`
func (p *peerMux) String() string {
	return p.id + "@" + p.codec.Address()
}

func NewPeer(id string, name string, version int, c Codec, flag connFlag, m ProtocolMap, log log15.Logger) Peer {
	pm := &peerMux{
		codec:      c,
		id:         id,
		name:       name,
		version:    version,
		flag:       flag,
		createAt:   time.Now(),
		protoMap:   m,
		readQueue:  make(chan Msg, peerReadMsgBufferSize),
		writeQueue: make(chan Msg, peerWriteMsgBufferSize),
		running:    0,
		errChan:    make(chan error, 1),
		log:        log,
	}

	return pm
}

func (p *peerMux) ID() string {
	return p.id
}

func (p *peerMux) is(flag connFlag) bool {
	return p.flag.is(flag)
}

func (p *peerMux) run() (err error) {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		defer atomic.StoreInt32(&p.running, 0)

		err = p.onAdded()
		if err != nil {
			return
		}

		defer p.onRemoved()

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()

			select {
			case p.errChan <- p.readLoop():
			default:

			}
		}()

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()

			select {
			case p.errChan <- p.writeLoop():
			default:

			}
		}()

		p.wg.Add(1)
		go func() {
			defer p.wg.Done()

			select {
			case p.errChan <- p.handleLoop():
			default:

			}
		}()

		return <-p.errChan
	}

	return errPeerAlreadyRunning
}

func (p *peerMux) readLoop() (err error) {
	defer close(p.readQueue)

	var msg Msg

	for {
		msg, err = p.codec.ReadMsg()
		if err != nil {
			return
		}

		msg.ReceivedAt = time.Now()
		msg.Sender = p

		switch msg.Pid {
		case baseProtocolID:
			switch msg.Code {
			case baseDisconnect:
				return PeerQuitting
			case baseTooManyMsg:
				// todo
			default:
				// nothing
			}
		default:
			if _, ok := p.protoMap[msg.Pid]; ok {
				p.readQueue <- msg
			} else {
				return PeerUnknownProtocol
			}
		}
	}
}

func (p *peerMux) writeLoop() (err error) {
	var msg Msg
	for msg = range p.writeQueue {
		if err = p.codec.WriteMsg(msg); err != nil {
			return
		}
	}

	return nil
}

func (p *peerMux) handleLoop() (err error) {
	var msg Msg
	for msg = range p.readQueue {
		err = p.protoMap[msg.Pid].Handle(msg)
		if err != nil {
			return
		}
	}

	return nil
}

func (p *peerMux) Close(err PeerError) error {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {

		_ = p.WriteMsg(Msg{
			Pid:     baseProtocolID,
			Code:    baseDisconnect,
			Payload: err.Serialize(),
		})

		p.wg.Wait()
	}

	return errPeerNotRunning
}

func (p *peerMux) onAdded() (err error) {
	for _, pt := range p.protoMap {
		err = pt.OnPeerAdded(p)
		if err != nil {
			return
		}
	}

	return nil
}

func (p *peerMux) onRemoved() {
	var err error
	for _, pt := range p.protoMap {
		err = pt.OnPeerRemoved(p)
		if err != nil {
			p.log.Error(fmt.Sprintf("failed to remove peer %s of protocol %s: %v", p, pt.Name(), err))
		}
	}

	return
}

func (p *peerMux) Info() PeerInfo {
	ps := make([]string, len(p.protoMap))

	i := 0
	for _, pt := range p.protoMap {
		ps[i] = pt.Name()
	}

	return PeerInfo{
		ID:        p.id,
		Name:      p.name,
		Version:   p.version,
		Protocols: ps,
		Address:   p.codec.Address(),
		Flag:      p.flag.String(),
		CreateAt:  p.createAt.Format("2006-01-02 15:04:05"),
	}
}
