package net

import (
	"errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sort"
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

const safeHeight uint64 = 3600 // peer`s height max taller than from + 3600
const minBlocks uint64 = 3600  // minimal snapshot blocks per subLedger request
const maxBlocks uint64 = 10800 // maximal snapshot blocks per subLedger request

type subLedgerPiece struct {
	from  uint64
	count uint64
	peer  *Peer
}

// split large subledger request to many small pieces
func splitSubLedger(from, to uint64, peers Peers) (cs []*subLedgerPiece) {
	// sort peers from low to high
	sort.Sort(peers)

	// choose the tallest peer
	if to-from+1 < minBlocks {
		cs = append(cs, &subLedgerPiece{
			from:  from,
			count: to - from + 1,
			peer:  peers[len(peers)-1],
		})
		return
	}

	var p, t uint64 // piece length
	for _, peer := range peers {
		if peer.height > from+safeHeight {
			p = peer.height - from - safeHeight

			// peer not high enough
			if p < minBlocks {
				continue
			}

			// piece too large
			if p > maxBlocks {
				p = maxBlocks
			}

			// piece end
			t = from + p

			// piece end exceed target height
			if t > to {
				p = to - from + 1
			}
			// reset piece is too small, then collapse to one piece
			if to < t+minBlocks {
				p += to - t
			}

			cs = append(cs, &subLedgerPiece{
				from:  from,
				count: p,
				peer:  peer,
			})

			from = p + 1
			if from > to {
				break
			}
		}
	}

	// reset piece, alloc to best peer
	if from < to {
		cs = append(cs, &subLedgerPiece{
			from:  from,
			count: to - from + 1,
			peer:  peers[len(peers)-1],
		})
	}

	return
}

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

func (s *subLedgerRequest) Handle(msg *p2p.Msg, peer *Peer) {

}

// @request for chunk
type chunkRequest struct {
	id         uint64
	start, end uint64
	peer       *Peer
	rec        receiveBlocks
	done       doneCallback
	expiration time.Time
}

func (c *chunkRequest) Done(err error) {
	panic("implement me")
}

func (c *chunkRequest) Expired() bool {
	panic("implement me")
}

func (c *chunkRequest) Handle(msg *p2p.Msg, peer *Peer) {
	panic("implement me")
}

func (c *chunkRequest) ID() uint64 {
	panic("implement me")
}

func (c *chunkRequest) Run() {
	panic("implement me")
}
