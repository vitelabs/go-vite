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
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/p2p/protos"

	"github.com/vitelabs/go-vite/p2p/vnode"

	"github.com/vitelabs/go-vite/log15"
)

var errPeerAlreadyRunning = errors.New("peer is already running")
var errPeerNotRunning = errors.New("peer is not running")
var errPeerWriteBusy = errors.New("peer is busy")
var errPeerCannotWrite = errors.New("peer is not writable")

type peerProtocol struct {
	Protocol
	state interface{}
}

type protoPeer struct {
	*peerMux
	peerProtocol
}

func (p *protoPeer) WriteMsg(msg Msg) (err error) {
	msg.pid = p.Protocol.ID()
	return p.peerMux.WriteMsg(msg)
}

// WriteMsg will put msg into queue, then write asynchronously
func (p *peerMux) WriteMsg(msg Msg) (err error) {
	p.write()
	defer p.writeDone()

	if atomic.LoadInt32(&p.writable) == 0 {
		return errPeerCannotWrite
	}

	select {
	case p.writeQueue <- msg:
		return nil
	default:
		return errPeerWriteBusy
	}
}

func (p *peerMux) write() {
	atomic.AddInt32(&p.writing, 1)
}

func (p *peerMux) writeDone() {
	atomic.AddInt32(&p.writing, -1)
}

func (p *protoPeer) State() interface{} {
	return p.state
}

func (p *protoPeer) SetState(state interface{}) {
	p.state = state
}

type protocolState struct {
	Name  string      `json:"name"`
	State interface{} `json:"state"`
}

type protocolStateMap = map[ProtocolID]protocolState

type PeerInfo struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Version    uint32           `json:"version"`
	Protocols  []string         `json:"protocols"`
	Address    string           `json:"address"`
	Level      Level            `json:"level"`
	CreateAt   string           `json:"createAt"`
	State      protocolStateMap `json:"state"`
	ReadQueue  int              `json:"readQueue"`
	WriteQueue int              `json:"writeQueue"`
}

const peerReadMsgBufferSize = 10
const peerWriteMsgBufferSize = 100

type levelManager interface {
	changeLevel(p PeerMux, old Level) error
}

type peerMux struct {
	codec      Codec
	id         vnode.NodeID
	name       string
	version    uint32
	level      Level
	pm         levelManager
	createAt   time.Time
	protoMap   map[ProtocolID]*protoPeer
	running    int32
	writable   int32 // set to 0 when write error in writeLoop, or close actively
	writing    int32
	readQueue  chan Msg // will be closed when read error in readLoop
	writeQueue chan Msg // will be closed in method Close
	errChan    chan error
	wg         sync.WaitGroup
	log        log15.Logger
}

// Level return the peer`s level
func (p *peerMux) Level() Level {
	return p.level
}

// SetLevel change the peer`s level, return error is not nil if peer is not running, or change failed
func (p *peerMux) SetLevel(level Level) error {
	if atomic.LoadInt32(&p.running) == 0 {
		return errPeerNotRunning
	}

	old := p.level
	p.level = level

	err := p.pm.changeLevel(p, old)
	if err != nil {
		p.log.Warn(fmt.Sprintf("failed to change peer %s from level %d to level %d", p.Address(), old, level))
	}

	return err
}

// String return `id@address`
func (p *peerMux) String() string {
	return p.id.Brief() + "@" + p.codec.Address().String()
}

// Address return the remote net address
func (p *peerMux) Address() net.Addr {
	return p.codec.Address()
}

func NewPeer(id vnode.NodeID, name string, version uint32, c Codec, level Level, m map[ProtocolID]peerProtocol) PeerMux {
	pm := &peerMux{
		codec:      c,
		id:         id,
		name:       name,
		version:    version,
		level:      level,
		createAt:   time.Now(),
		readQueue:  make(chan Msg, peerReadMsgBufferSize),
		writeQueue: make(chan Msg, peerWriteMsgBufferSize),
		running:    0,
		writable:   1,
		errChan:    make(chan error, 3),
		log:        p2pLog.New("peer", id.Brief()),
	}

	pm.protoMap = make(map[ProtocolID]*protoPeer, len(m))
	for pid, ppt := range m {
		pm.protoMap[pid] = &protoPeer{
			peerMux:      pm,
			peerProtocol: ppt,
		}
	}

	return pm
}

func (p *peerMux) ID() vnode.NodeID {
	return p.id
}

// setManager will be invoked before run by module p2p
func (p *peerMux) setManager(pm levelManager) {
	p.pm = pm
}

func (p *peerMux) run() (err error) {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		err = p.onAdded()
		if err != nil {
			return
		}

		defer p.onRemoved()

		p.goLoop(p.readLoop, p.errChan)
		p.goLoop(p.writeLoop, p.errChan)
		p.goLoop(p.handleLoop, p.errChan)

		err = <-p.errChan
		return
	}

	return errPeerAlreadyRunning
}

func (p *peerMux) goLoop(fn func() error, ch chan<- error) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := fn()
		ch <- err
	}()
}

func (p *peerMux) readLoop() (err error) {
	defer close(p.readQueue)

	var msg Msg

	for {
		p.log.Debug(fmt.Sprintf("begin read message"))
		msg, err = p.codec.ReadMsg()
		p.log.Debug(fmt.Sprintf("read message %d/%d %d bytes done", msg.pid, msg.Code, len(msg.Payload)))
		if err != nil {
			return
		}

		msg.ReceivedAt = time.Now()

		switch msg.pid {
		case baseProtocolID:
			switch msg.Code {
			case baseDisconnect:
				return PeerQuitting
			case baseTooManyMsg:
			// todo
			case baseHeartBeat:
				var heartBeat = &protos.HeartBeat{}
				err = proto.Unmarshal(msg.Payload, heartBeat)
				if err != nil {
					return PeerUnmarshalError
				}

				for id, data := range heartBeat.State {
					if pp, ok := p.protoMap[ProtocolID(id)]; ok {
						go pp.Protocol.SetState(data, pp)
					} else {
						return PeerUnknownProtocol
					}
				}

			default:
				// nothing
			}
		default:
			if pt, ok := p.protoMap[msg.pid]; ok {
				msg.Sender = pt
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
		t1 := time.Now()
		p.log.Debug(fmt.Sprintf("begin write msg %d/%d %d bytes", msg.pid, msg.Code, len(msg.Payload)))
		if err = p.codec.WriteMsg(msg); err != nil {
			p.log.Debug(fmt.Sprintf("write msg %d/%d %d bytes error: %v", msg.pid, msg.Code, len(msg.Payload), err))
			atomic.StoreInt32(&p.writable, 0)
			return
		}
		p.log.Debug(fmt.Sprintf("write msg %d/%d %d bytes done[%d][%s]", msg.pid, msg.Code, len(msg.Payload), len(p.writeQueue), time.Now().Sub(t1)))
	}

	return nil
}

func (p *peerMux) handleLoop() (err error) {
	var msg Msg
	for msg = range p.readQueue {
		t1 := time.Now()
		p.log.Debug(fmt.Sprintf("begin handle msg %d/%d", msg.pid, msg.Code))
		err = p.protoMap[msg.pid].Handle(msg)
		p.log.Debug(fmt.Sprintf("handle msg %d/%d done[%d][%s]", msg.pid, msg.Code, len(p.readQueue), time.Now().Sub(t1)))
		if err != nil {
			return
		}
	}

	return nil
}

func (p *peerMux) Close(err PeerError) (err2 error) {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {

		_ = Disconnect(p, err)

		atomic.StoreInt32(&p.writable, 0)

		// ensure nobody is writing
		for {
			if atomic.LoadInt32(&p.writing) == 0 {
				break
			}

			time.Sleep(100 * time.Millisecond)
		}

		close(p.writeQueue)

		if err3 := p.codec.Close(); err3 != nil {
			err2 = err3
		}

		p.wg.Wait()
	}

	return errPeerNotRunning
}

func (p *peerMux) onAdded() (err error) {
	for _, pt := range p.protoMap {
		err = pt.OnPeerAdded(pt)
		if err != nil {
			p.log.Error(fmt.Sprintf("failed to add peer %s of protocol %s: %v", p, pt.Name(), err))
			return
		}
	}

	return nil
}

func (p *peerMux) onRemoved() {
	var err error
	for _, pt := range p.protoMap {
		err = pt.OnPeerRemoved(pt)
		if err != nil {
			p.log.Error(fmt.Sprintf("failed to remove peer %s of protocol %s: %v", p, pt.Name(), err))
		}
	}

	return
}

func (p *peerMux) Info() PeerInfo {
	state := make(protocolStateMap, len(p.protoMap))
	pts := make([]string, 0, len(p.protoMap))

	for pid, pt := range p.protoMap {
		state[pid] = protocolState{
			Name:  pt.Name(),
			State: pt.state,
		}
		pts = append(pts, pt.Name())
	}

	return PeerInfo{
		ID:         p.id.String(),
		Name:       p.name,
		Version:    p.version,
		Protocols:  pts,
		Address:    p.codec.Address().String(),
		Level:      p.level,
		CreateAt:   p.createAt.Format("2006-01-02 15:04:05"),
		State:      state,
		ReadQueue:  len(p.readQueue),
		WriteQueue: len(p.writeQueue),
	}
}
