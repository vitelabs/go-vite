package net

import (
	"io"
	net2 "net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/log15"
)

const fReadTimeout = 20 * time.Second
const fWriteTimeout = 20 * time.Second
const fileTimeout = 5 * time.Minute

var errFileServerIsRunning = errors.New("file server is running")
var errFileServerNotRunning = errors.New("file server is not running")

type fileReader interface {
	FileReader(filename string) (io.ReadCloser, error)
}

type FileServerStatus struct {
	Conns []string
}

type fileServer struct {
	addr    string
	ln      net2.Listener
	mu      sync.Mutex
	conns   map[string]net2.Conn // key is addr
	chain   Chain
	running int32
	wg      sync.WaitGroup
	log     log15.Logger
}

func newFileServer(addr string, chain Chain) *fileServer {
	return &fileServer{
		addr:  addr,
		conns: make(map[string]net2.Conn),
		chain: chain,
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

func (s *fileServer) deleteConn(c net2.Conn) {
	c.Close()

	// is running
	if atomic.LoadInt32(&s.running) == 1 {
		s.mu.Lock()
		delete(s.conns, c.RemoteAddr().String())
		s.mu.Unlock()
	}
}

func (s *fileServer) addConn(c net2.Conn) {
	if atomic.LoadInt32(&s.running) == 1 {
		s.mu.Lock()
		s.conns[c.RemoteAddr().String()] = c
		s.mu.Unlock()
	}
}

func (s *fileServer) handleConn(conn net2.Conn) {
	defer s.wg.Done()

	s.addConn(conn)
	defer s.deleteConn(conn)

	// todo
}
