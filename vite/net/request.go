package net

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"time"
)

type reqState byte

const (
	reqWaiting reqState = iota
	reqPending
	reqDone
	reqError
	reqCancel
)

var reqStatus = [...]string{
	reqWaiting: "waiting",
	reqPending: "pending",
	reqDone:    "done",
	reqError:   "error",
	reqCancel:  "canceled",
}

func (s reqState) String() string {
	return reqStatus[s]
}

type RequestRunCtx struct {
	peers *peerSet
	pool  RequestPool
	fc    *fileClient
}

type Request interface {
	Handle(msg *p2p.Msg, peer *Peer)
	ID() uint64
	Run()
	Done(err error)
	Expired() bool
}

type receiveBlocks func(sblocks []*ledger.SnapshotBlock, ablocks map[types.Address][]*ledger.AccountBlock)
type doneCallback func(id uint64, err error)
type msgReceive func(cmd cmd, data []byte, peer *Peer)

var errMissingPeer = errors.New("request missing peer")
var errUnknownResErr = errors.New("unknown response exception")
var errUnExpectedRes = errors.New("unexpected response")

type subLedgerRequest struct {
	id         uint64 // unique id
	peer       *Peer
	msg        *message.GetSubLedger
	state      reqState
	act        msgReceive
	done       doneCallback
	expiration time.Time
}

func (s *subLedgerRequest) Done(err error) {

}

func (s *subLedgerRequest) Run() {
	err := s.peer.Send(GetSubLedgerCode, s.id, s.msg)

	if err != nil {
		s.done(s.id, err)
	}
}

func (s *subLedgerRequest) ID() uint64 {
	return s.id
}

func (s *subLedgerRequest) Expired() bool {
	return time.Now().After(s.expiration)
}

func (s *subLedgerRequest) Handle(cmd cmd, data []byte, peer *Peer) {
	s.act(cmd, data, peer)

	switch cmd {
	case FileListCode:
		msg := new(message.FileList)
		err := msg.Deserialize(data)
		if err != nil {
			s.done(s.id, fmt.Errorf("deserialize message %s error: %v", cmd, err))
			return
		}

		for _, file := range msg.Files {
			r := &fileRequest{
				_id:        getMsgId(),
				_pid:       s.id,
				nonce:      msg.Nonce,
				file:       file,
				_peer:      peer,
				expiration: time.Now().Add(2 * time.Minute),
				rec:        s.rec,
				done:       s.childDone,
			}
		}

		for _, chunk := range msg.Chunk {
			if chunk[1]-chunk[0] != 0 {
				c := &chunkRequest{
					_id:        getMsgId(),
					_pid:       s.id,
					start:      chunk[0],
					end:        chunk[1],
					_peer:      peer,
					expiration: time.Now().Add(30 * time.Second),
					rec:        s.rec,
					done:       s.childDone,
				}
			}
		}
	case ExceptionCode:
		exp, err := message.DeserializeException(data)
		if err == nil {
			s.done(s.id, exp)
		} else {
			s.done(s.id, errUnknownResErr)
		}
	default:
		s.done(s.id, fmt.Errorf("unexpected response %s", cmd))
	}
}

// @request for file
type fileRequest struct {
	_id, _pid uint64
	nonce     uint64
	file      *message.File
	_peer     *Peer

	fc *fileClient

	rec  receiveBlocks
	done doneCallback

	expiration time.Time
}

// @request for chunk
type chunkRequest struct {
	_id, _pid  uint64
	start, end uint64
	_peer      *Peer

	rec  receiveBlocks
	done doneCallback

	expiration time.Time
}
