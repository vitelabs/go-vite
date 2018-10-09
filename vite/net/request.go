package net

import (
	"errors"
	"fmt"
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
	reqRespond
	reqDone
	reqError
	reqCancel
)

var reqStatus = [...]string{
	reqWaiting: "waiting",
	reqPending: "pending",
	reqRespond: "respond",
	reqDone:    "done",
	reqError:   "error",
	reqCancel:  "canceled",
}

func (s reqState) String() string {
	return reqStatus[s]
}

type context struct {
	*syncer
	peers *peerSet
	pool  RequestPool
	fc    *fileClient
}

type Request interface {
	Handle(ctx *context, msg *p2p.Msg, peer *Peer)
	ID() uint64
	Run()
	Done(err error)
	Expired() bool
}

type receiveBlocks func(sblocks []*ledger.SnapshotBlock, mblocks map[types.Address][]*ledger.AccountBlock)
type doneCallback func(id uint64, err error)

var errMissingPeer = errors.New("request missing peer")
var errUnknownResErr = errors.New("unknown response exception")
var errUnExpectedRes = errors.New("unexpected response")
var errMaxRetry = errors.New("max Retry")

const safeHeight uint64 = 3600 // peer`s height max taller than from + 3600
const minBlocks uint64 = 3600  // minimal snapshot blocks per subLedger request
const maxBlocks uint64 = 10800 // maximal snapshot blocks per subLedger request
const maxRetry = 3

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

// @request for subLedger, will get FileList and Chunk
type subLedgerRequest struct {
	id, cid    uint64 // id & child_id
	from, to   uint64
	_retry     int
	peer       *Peer
	state      reqState
	expiration time.Time
	done       doneCallback
}

func (s *subLedgerRequest) Retry(ctx *context, peer *Peer) {
	if s._retry >= maxRetry {
		s.Done(errMaxRetry)
		return
	}

	peers := ctx.peers.Pick(peer.height)
	if len(peers) != 0 {
		s.peer = peers[0]
		ctx.pool.Retry(s.id)
		s._retry++
	} else {
		s.Done(errMissingPeer)
	}
}

func (s *subLedgerRequest) Handle(ctx *context, pkt *p2p.Msg, peer *Peer) {
	if cmd(pkt.Cmd) == FileListCode {
		msg := new(message.FileList)
		err := msg.Deserialize(pkt.Payload)
		if err != nil {
			fmt.Println("subLedgerRequest handle error: ", err)
			s.Retry(ctx, peer)
			return
		}

		// request files
		s.cid++
		ctx.fc.request(&fileReq{
			id:    s.cid,
			files: msg.Files,
			nonce: msg.Nonce,
			peer:  peer,
			rec:   ctx.syncer.receiveBlocks,
			done:  s.childDone,
		})

		// request chunks
		for _, chunk := range msg.Chunks {
			if chunk[1]-chunk[0] > 0 {
				msgId := ctx.pool.MsgID()

				c := &chunkRequest{
					id:         msgId,
					start:      chunk[0],
					end:        chunk[1],
					peer:       peer,
					expiration: time.Now().Add(30 * time.Second),
					done:       s.childDone,
				}

				ctx.pool.Add(c)
			}
		}
	} else {
		s.Retry(ctx, peer)
	}
}

func (s *subLedgerRequest) ID() uint64 {
	return s.id
}

func (s *subLedgerRequest) Run() {
	err := s.peer.Send(SubLedgerCode, s.id, &message.GetSubLedger{
		From:    &ledger.HashHeight{Height: s.from},
		Count:   s.to - s.from + 1,
		Forward: true,
	})

	if err != nil {
		s.Done(err)
	} else {
		s.state = reqPending
	}
}

func (s *subLedgerRequest) childDone(id uint64, err error) {

}

func (s *subLedgerRequest) Done(err error) {
	if err != nil {
		s.state = reqError
	} else {
		s.state = reqDone
	}

	s.done(s.id, err)
}

func (s *subLedgerRequest) Expired() bool {
	return time.Now().After(s.expiration)
}

// @request for chunk
type chunkRequest struct {
	id         uint64
	start, end uint64
	_retry     int
	peer       *Peer
	state      reqState
	expiration time.Time
	done       doneCallback
}

func (c *chunkRequest) Retry(ctx *context, peer *Peer) {
	if c._retry >= maxRetry {
		c.Done(errMaxRetry)
		return
	}

	peers := ctx.peers.Pick(peer.height)
	if len(peers) != 0 {
		c.peer = peers[0]
		ctx.pool.Retry(c.id)
		c._retry++
	} else {
		c.Done(errMissingPeer)
	}
}

func (c *chunkRequest) Handle(ctx *context, pkt *p2p.Msg, peer *Peer) {
	if cmd(pkt.Cmd) == SubLedgerCode {
		msg := new(message.SubLedger)
		err := msg.Deserialize(pkt.Payload)
		if err != nil {
			fmt.Println("chunkRequest handle error: ", err)
			c.Retry(ctx, peer)
			return
		}

		ctx.syncer.receiveBlocks(msg.SBlocks, msg.ABlocks)
		c.Done(nil)
	} else {
		c.Retry(ctx, peer)
	}
}

func (c *chunkRequest) ID() uint64 {
	return c.id
}

func (c *chunkRequest) Run() {
	err := c.peer.Send(SubLedgerCode, c.id, &message.GetChunk{
		Start: c.start,
		End:   c.end,
	})

	if err != nil {
		c.state = reqError
	} else {
		c.state = reqPending
	}
}

func (c *chunkRequest) Done(err error) {
	if err != nil {
		c.state = reqError
	} else {
		c.state = reqDone
	}

	c.done(c.id, err)
}

func (c *chunkRequest) Expired() bool {
	return time.Now().After(c.expiration)
}
