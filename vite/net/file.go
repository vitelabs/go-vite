package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"net"
	"strconv"
	"sync"
)

type FileServer struct {
	ln     net.Listener
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
	var nonce uint64

	defer conn.Close()
	defer s.wg.Done()
	defer s.delRecord(nonce)

	for {
		select {
		case <-s.term:
			// send disconnect msg

		default:
			msg, err := p2p.ReadMsg(conn, true)
			if err != nil {
				return
			}

			if cmd(msg.Cmd) == GetFileCode {
				req := new(FileRequestMsg)
				err = req.Deserialize(msg.Payload)
				if err != nil {
					s.log.Error(fmt.Sprintf("can`t parse message %s", GetFileCode), "error", err)
				}

				nonce = req.Nonce
				// todo get NodeID, verify whether nonce can be used by the NodeID
				// todo send File through conn
			}
		}
	}
}

// after send FileListMsg to peer, add record to fileServer
func (s *FileServer) addRecord(nonce uint64, peerId string) {
	s.record[nonce] = peerId
}

func (s *FileServer) delRecord(nonce uint64) {
	delete(s.record, nonce)
}

// @section fileRequest
type FileRequest struct {
	file *file
	nonce uint64
	peer *Peer
	done chan struct{}
}

func (f *FileRequest) Perform(p *Peer) {

}