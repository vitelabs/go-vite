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

package net

import (
	"errors"
	"fmt"
	"io"
	net2 "net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
)

const fileTimeout = 5 * time.Minute

var errSyncServerIsRunning = errors.New("sync server is running")
var errSyncServerNotRunning = errors.New("sync server is not running")

type FileServerStatus struct {
	Connections []SyncConnectionStatus `json:"connections"`
}

type syncServer struct {
	addr     string
	ln       net2.Listener
	mu       sync.Mutex
	sconnMap map[peerId]*syncConn // key is addr
	chain    ledgerReader
	factory  syncConnReceiver
	running  int32
	wg       sync.WaitGroup
	log      log15.Logger
}

func newSyncServer(addr string, chain ledgerReader, factory syncConnReceiver) *syncServer {
	return &syncServer{
		addr:     addr,
		sconnMap: make(map[peerId]*syncConn),
		chain:    chain,
		factory:  factory,
		log:      log15.New("module", "server"),
	}
}

func (s *syncServer) status() FileServerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	connStates := make([]SyncConnectionStatus, len(s.sconnMap))

	i := 0
	for _, c := range s.sconnMap {
		connStates[i] = c.status()
		i++
	}

	return FileServerStatus{
		Connections: connStates,
	}
}

func (s *syncServer) start() error {
	if atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		if ln, err := net2.Listen("tcp", s.addr); err != nil {
			return err
		} else {
			s.ln = ln
		}

		s.wg.Add(1)
		go s.listenLoop()

		return nil
	}

	return errSyncServerIsRunning
}

func (s *syncServer) stop() (err error) {
	if atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		if s.ln != nil {
			err = s.ln.Close()
		}

		for id, c := range s.sconnMap {
			_ = c.close()
			delete(s.sconnMap, id)
		}

		s.wg.Wait()
	}

	return errSyncServerNotRunning
}

func (s *syncServer) listenLoop() {
	defer s.wg.Done()

	for {
		c, err := s.ln.Accept()
		if err != nil {
			if nerr, ok := err.(net2.Error); ok {
				if nerr.Temporary() {
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}
			return
		}

		s.wg.Add(1)
		go s.handleConn(c)
	}
}

func (s *syncServer) deleteConn(c *syncConn) {
	_ = c.close()

	// is running
	if atomic.LoadInt32(&s.running) == 1 {
		s.mu.Lock()
		delete(s.sconnMap, c.ID())
		s.mu.Unlock()
	}
}

func (s *syncServer) addConn(c *syncConn) {
	if atomic.LoadInt32(&s.running) == 1 {
		s.mu.Lock()
		s.sconnMap[c.ID()] = c
		s.mu.Unlock()
	}
}

func (s *syncServer) handleConn(conn net2.Conn) {
	defer s.wg.Done()

	sconn, err := s.factory.receive(conn)
	if err != nil {
		_ = conn.Close()
		return
	}

	s.addConn(sconn)
	defer s.deleteConn(sconn)

	var msg p2p.Msg
	for {
		msg, err = sconn.c.ReadMsg()
		if err != nil {
			s.log.Error(fmt.Sprintf("failed read message from sync connection %s: %v", conn.RemoteAddr(), err))
			return
		}

		if msg.Code == p2p.CodeDisconnect {
			s.log.Warn(fmt.Sprintf("sync connection %s quit", conn.RemoteAddr()))
			return
		}

		if msg.Code != p2p.CodeSyncRequest {
			continue
		}
		request := &syncRequest{}
		err = request.deserialize(msg.Payload)
		if err != nil {
			return
		}

		var reader interfaces.LedgerReader
		reader, err = s.chain.GetLedgerReaderByHeight(request.from, request.to)
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to read chunk<%d-%d> from %s error: %v", request.from, request.to, conn.RemoteAddr(), err))

			_ = sconn.c.WriteMsg(p2p.Msg{
				Code:    p2p.CodeException,
				Id:      msg.Id,
				Payload: []byte{byte(p2p.ExpServerError)},
			})

			continue
		}

		segment := reader.Seg()
		if segment.PrevHash != request.prevHash || segment.Hash != request.endHash {
			s.log.Warn(fmt.Sprintf("different chunk<%d-%d> %s/%s %s/%s from %s", request.from, request.to, request.prevHash, request.endHash, segment.PrevHash, segment.PrevHash, conn.RemoteAddr()))

			_ = sconn.c.WriteMsg(p2p.Msg{
				Code:    p2p.CodeException,
				Id:      msg.Id,
				Payload: []byte{byte(p2p.ExpChunkNotMatch)},
			})

			continue
		}

		ready := &syncResponse{
			from:     segment.Bound[0],
			to:       segment.Bound[1],
			size:     uint64(reader.Size()),
			prevHash: segment.PrevHash,
			endHash:  segment.Hash,
		}
		var data []byte
		data, err = ready.Serialize()
		if err != nil {
			_ = sconn.c.WriteMsg(p2p.Msg{
				Code:    p2p.CodeException,
				Id:      msg.Id,
				Payload: []byte{byte(p2p.ExpOther)},
			})
			continue
		}

		err = sconn.c.WriteMsg(p2p.Msg{
			Code:    p2p.CodeSyncReady,
			Id:      msg.Id,
			Payload: data,
		})

		if err != nil {
			s.log.Error(fmt.Sprintf("failed to send chunk response <%d-%d> to %s: %v", segment.Bound[0], segment.Bound[1], conn.RemoteAddr(), err))
			return
		}

		var wn int64
		_ = conn.SetWriteDeadline(time.Now().Add(fileTimeout))
		wn, err = io.Copy(conn, reader)
		_ = reader.Close()

		if wn != int64(reader.Size()) {
			err = fmt.Errorf("write %d/%d bytes", wn, reader.Size())
		}

		if err != nil {
			s.log.Error(fmt.Sprintf("failed to send chunk<%d-%d> %d bytes to %s: %v", segment.Bound[0], segment.Bound[1], wn, conn.RemoteAddr(), err))
			return
		} else {
			s.log.Info(fmt.Sprintf("send chunk<%d-%d> %d bytes to %s done", segment.Bound[0], segment.Bound[1], wn, conn.RemoteAddr()))
		}
	}
}
