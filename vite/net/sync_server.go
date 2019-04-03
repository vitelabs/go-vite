package net

import (
	"fmt"
	"io"
	net2 "net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/vite/net/message"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/log15"
)

const fReadTimeout = 20 * time.Second
const fWriteTimeout = 20 * time.Second
const fileTimeout = 5 * time.Minute

var errFileServerIsRunning = errors.New("file server is running")
var errFileServerNotRunning = errors.New("file server is not running")

type FileServerStatus struct {
	Conns []string
}

type fileServer struct {
	addr    string
	ln      net2.Listener
	mu      sync.Mutex
	conns   map[string]syncConnection // key is addr
	chain   ledgerReader
	fac     func(net2.Conn) syncConnection
	running int32
	wg      sync.WaitGroup
	log     log15.Logger
}

func newFileServer(addr string, chain ledgerReader, fac func(net2.Conn) syncConnection) *fileServer {
	return &fileServer{
		addr:  addr,
		conns: make(map[string]syncConnection),
		chain: chain,
		fac:   fac,
		log:   log15.New("module", "net/fileServer"),
	}
}

func (s *fileServer) status() FileServerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	conns := make([]string, len(s.conns))

	i := 0
	for addr := range s.conns {
		conns[i] = addr
		i++
	}

	return FileServerStatus{
		Conns: conns,
	}
}

func (s *fileServer) start() error {
	if atomic.CompareAndSwapInt32(&s.running, 0, 1) {
		if ln, err := net2.Listen("tcp", s.addr); err != nil {
			return err
		} else {
			s.ln = ln
		}

		s.wg.Add(1)
		common.Go(s.listenLoop)

		return nil
	}

	return errFileServerIsRunning
}

func (s *fileServer) stop() (err error) {
	if atomic.CompareAndSwapInt32(&s.running, 1, 0) {
		if s.ln != nil {
			err = s.ln.Close()
		}

		for id, c := range s.conns {
			_ = c.Close()
			delete(s.conns, id)
		}

		s.wg.Wait()
	}

	return errFileServerNotRunning
}

func (s *fileServer) listenLoop() {
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

func (s *fileServer) deleteConn(c syncConnection) {
	c.Close()

	// is running
	if atomic.LoadInt32(&s.running) == 1 {
		s.mu.Lock()
		delete(s.conns, c.RemoteAddr().String())
		s.mu.Unlock()
	}
}

func (s *fileServer) addConn(c syncConnection) {
	if atomic.LoadInt32(&s.running) == 1 {
		s.mu.Lock()
		s.conns[c.RemoteAddr().String()] = c
		s.mu.Unlock()
	}
}

func (s *fileServer) handleConn(conn net2.Conn) {
	defer s.wg.Done()

	sconn := s.fac(conn)
	err := sconn.handshake()
	if err != nil {
		return
	}

	s.addConn(sconn)
	defer s.deleteConn(sconn)

	var msg syncMsg
	for {
		msg, err = sconn.read()
		if err != nil {
			return
		}

		if msg.code == syncQuit {
			return
		}

		if msg.code == syncRequest {
			chunk := msg.payload.(message.GetChunk)
			reader, err := s.chain.GetLedgerReaderByHeight(chunk.Start, chunk.End)
			if err != nil {
				s.log.Error(fmt.Sprintf("read chunk %d-%d error: %v", chunk.Start, chunk.End, err))

				_ = sconn.write(syncMsg{
					code: syncError,
				})

				return
			}

			err = sconn.write(syncMsg{
				code: syncReady,
			})

			if err != nil {
				return
			}

			_, err = io.Copy(conn, reader)
			_ = reader.Close()

			if err != nil {
				s.log.Error(fmt.Sprintf("send chunk<%d-%d> to %s error: %v", chunk.Start, chunk.End, conn.RemoteAddr(), err))
				return
			} else {
				s.log.Info(fmt.Sprintf("send chunk<%d-%d> to %s done", chunk.Start, chunk.End, conn.RemoteAddr()))
			}
		} else {
			return
		}
	}
}
