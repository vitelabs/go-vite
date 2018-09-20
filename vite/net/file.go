package net

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/snappy"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/p2p"
	"io"
	"net"
	"strconv"
	"sync"
)

type FileServer struct {
	ln     net.Listener
	record map[uint64]string
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
			msg, err := ReadMsg(conn)
			if err != nil {
				return
			}

			if cmd(msg.Cmd) == GetFileCode {
				data := make([]byte, msg.Size)
				n, err := io.ReadFull(msg.Payload, data)
				if err != nil {
					s.log.Error(fmt.Sprintf("read message %s payload error", GetFileCode), "error", err)
					return
				}

				req := new(FileRequest)
				err = req.Deserialize(data[:n])
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

// helper functions

func ReadMsg(fd io.Reader) (msg p2p.Msg, err error) {
	head := make([]byte, 32)
	if _, err = io.ReadFull(fd, head); err != nil {
		return
	}

	msg.CmdSetID = binary.BigEndian.Uint64(head[:8])
	msg.Cmd = binary.BigEndian.Uint64(head[8:16])
	msg.Id = binary.BigEndian.Uint64(head[16:24])
	msg.Size = binary.BigEndian.Uint64(head[24:32])

	payload := make([]byte, msg.Size)
	n, err := io.ReadFull(fd, payload)
	if err != nil {
		return
	}
	if uint64(n) != msg.Size {
		err = fmt.Errorf("read incomplete message %d/%d\n", n, msg.Size)
		return
	}

	var fullSize int
	fullSize, err = snappy.DecodedLen(payload)
	if err != nil {
		return
	}

	payload, err = snappy.Decode(nil, payload)
	if err != nil {
		return
	}
	msg.Size = uint64(fullSize)

	msg.Payload = p2p.BytesToReader(payload)

	return
}
