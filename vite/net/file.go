package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"net"
	"strconv"
	"sync"
	"time"
)

type FileServer struct {
	ln     net.Listener
	dialConns map[string]net.Conn	// all connections
	record map[uint64]string	// use to record nonce bind peerId
	term   chan struct{}
	log    log15.Logger
	wg     sync.WaitGroup
}

func newFileServer(port uint16) (*FileServer, error) {
	ln, err := net.Listen("tcp", "0.0.0.0:"+strconv.FormatUint(uint64(port), 10))

	if err != nil {
		return nil, err
	}

	return &FileServer{
		ln:     ln,
		record: make(map[uint64]string),
		term:   make(chan struct{}),
		log:    log15.New("module", "net/fileServer"),
	}, nil
}

func (s *FileServer) start() {
	defer s.ln.Close()

	for {
		select {
		case <-s.term:
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				continue
			}

			s.wg.Add(1)
			go s.handleConn(conn)
		}
	}
}

func (s *FileServer) stop() {
	select {
	case <-s.term:
	default:
		close(s.term)
	}

	s.wg.Wait()
}

func (s *FileServer) handleConn(conn net.Conn) {
	defer conn.Close()
	defer s.wg.Done()

	for {
		select {
		case <-s.term:
			return
		default:
			msg, err := p2p.ReadMsg(conn, true)
			if err != nil {
				return
			}

			if cmd(msg.Cmd) == GetFileCode {
				req := new(message.RequestFile)
				err = req.Deserialize(msg.Payload)
				if err != nil {
					s.log.Error(fmt.Sprintf("can`t parse message %s", GetFileCode), "error", err)
				}

				// todo send file content
			}
		}
	}
}

func (c *FileServer) Request(r *FileRequest) (err error) {
	select {
	case <- c.term:
		return errors.New("fileClient ternimated")
	default:
	}

	addr := r.peer.FileAddress().String()
	var conn net.Conn
	if conn, ok := c.dialConns[addr]; !ok {
		conn, err = net.Dial("tcp", addr)
		if err != nil {
			return
		}
		c.dialConns[addr] = conn
	}

	req := message.RequestFile{
		Name:  r.file.Name,
		Nonce: r.nonce,
	}

	data, _ := req.Serialize()

	n, err := conn.Write(data)
	if err != nil {
		return
	}
	if n != len(data) {
		return errors.New("send incompele fileRequest")
	}

	go c.ReadBlocks(conn)

	return nil
}

func (c *FileServer) ReadBlocks(conn net.Conn) {
	defer conn.Close()
	defer c.wg.Done()

	expired := time.NewTimer(10 * time.Second)
	defer expired.Stop()

	select {
	case <- c.term:
		return
	case <- expired.C:
		return
	default:
		// todo read blocks
	}
}

// @section fileRequest
type FileRequest struct {
	file *message.File
	nonce uint64
	peer *Peer
}
