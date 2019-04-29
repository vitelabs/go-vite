package net

import (
	"fmt"
	"io"
	net2 "net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/interfaces"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
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
	sconnMap map[peerId]syncConnection // key is addr
	chain    ledgerReader
	factory  syncConnReceiver
	running  int32
	wg       sync.WaitGroup
	log      log15.Logger
}

func newSyncServer(addr string, chain ledgerReader, factory syncConnReceiver) *syncServer {
	return &syncServer{
		addr:     addr,
		sconnMap: make(map[peerId]syncConnection),
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
			_ = c.Close()
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

func (s *syncServer) deleteConn(c syncConnection) {
	_ = c.Close()

	// is running
	if atomic.LoadInt32(&s.running) == 1 {
		s.mu.Lock()
		delete(s.sconnMap, c.ID())
		s.mu.Unlock()
	}
}

func (s *syncServer) addConn(c syncConnection) {
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

	var msg syncMsg
	for {
		msg, err = sconn.read()
		if err != nil {
			s.log.Error(fmt.Sprintf("failed read message from sync connection %s: %v", sconn.RemoteAddr(), err))
			return
		}

		if msg.code() == syncQuit {
			s.log.Warn(fmt.Sprintf("sync connection %s quit", sconn.RemoteAddr()))
			return
		}

		if msg.code() == syncRequest {
			request := msg.(*syncRequestMsg)

			var reader interfaces.LedgerReader
			reader, err = s.chain.GetLedgerReaderByHeight(request.from, request.to)
			if err != nil {
				s.log.Error(fmt.Sprintf("failed to read chunk %d-%d error: %v", request.from, request.to, err))

				_ = sconn.write(syncServerError)

				return
			}

			segment := reader.Seg()
			from, to := segment.Bound[0], segment.Bound[1]
			err = sconn.write(&syncReadyMsg{
				from:     from,
				to:       to,
				size:     uint64(reader.Size()),
				prevHash: segment.PrevHash,
				endHash:  segment.Hash,
			})

			if err != nil {
				s.log.Error(fmt.Sprintf("failed to send ready message %d-%d to %s: %v", from, to, sconn.RemoteAddr(), err))
				return
			}

			_ = conn.SetWriteDeadline(time.Now().Add(fileTimeout))
			_, err = io.Copy(conn, reader)
			_ = reader.Close()

			if err != nil {
				s.log.Error(fmt.Sprintf("failed to send chunk<%d-%d> to %s: %v", from, to, conn.RemoteAddr(), err))
				return
			} else {
				s.log.Info(fmt.Sprintf("send chunk<%d-%d> to %s done", from, to, conn.RemoteAddr()))
			}
		}
	}
}
