package net

import (
	"errors"
	"fmt"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	retry *retryPolicy
}

type retryPolicy struct {
	peers  *peerSet
	record map[uint64]int
	lock   sync.RWMutex
}

func (rp *retryPolicy) retry(id, to uint64) (p *Peer) {
	if rp.record[id] >= maxRetry {
		return
	}

	peers := rp.peers.Pick(to)
	n := len(peers)
	if n > 0 {
		rp.record[id]++
		return peers[rand.Intn(n)]
	}

	return
}

type Request interface {
	Handle(ctx *context, msg *p2p.Msg, peer *Peer)
	ID() uint64
	Run(ctx *context)
	Done(err error)
	Expired() bool
	State() reqState
	Req() Request
	To() uint64
	SetTo(to uint64)
}

type receiveBlocks func(sblocks []*ledger.SnapshotBlock, ablocks []*ledger.AccountBlock)
type doneCallback func(r Request, err error)

var errMissingPeer = errors.New("request missing peer")
var errUnExpectedRes = errors.New("unexpected response")

const minBlocks uint64 = 3600  // minimal snapshot blocks per subLedger request
const maxBlocks uint64 = 10800 // maximal snapshot blocks per subLedger request
const maxRetry = 3

type subLedgerPiece struct {
	from, to uint64
	peer     *Peer
}

// split large subledger request to many small pieces
func splitSubLedger(from, to uint64, peers Peers) (cs []*subLedgerPiece) {
	// sort peers from low to high
	sort.Sort(peers)

	total := to - from + 1
	if total < minBlocks {
		cs = append(cs, &subLedgerPiece{
			from: from,
			to:   to,
			peer: peers[len(peers)-1], // choose the tallest peer
		})
		return
	}

	var pCount, pTo uint64 // piece length
	for _, peer := range peers {
		if peer.height > from+minBlocks {
			pTo = peer.height - minBlocks
			pCount = pTo - from + 1

			// peer not high enough
			if pCount < minBlocks {
				continue
			}

			// piece too large
			if pCount > maxBlocks {
				pCount = maxBlocks
			}

			// piece to
			pTo = from + pCount - 1

			// piece to exceed target height
			if pTo > to {
				pTo = to
				pCount = to - from + 1
			}
			// reset piece is too small, then collapse to one piece
			if to < pTo+minBlocks {
				pCount += to - pTo
			}

			cs = append(cs, &subLedgerPiece{
				from: from,
				to:   pTo,
				peer: peer,
			})

			from = pTo + 1
			if from > to {
				break
			}
		}
	}

	// reset piece, alloc to best peer
	if from < to {
		// if peer is not too much, has rest chunk, then collapse the rest chunk with last chunk
		lastChunk := cs[len(cs)-1]
		if lastChunk.peer == peers[len(peers)-1] {
			lastChunk.to = to
		} else {
			cs = append(cs, &subLedgerPiece{
				from: from,
				to:   to,
				peer: peers[len(peers)-1],
			})
		}
	}

	return
}

func splitChunk(from, to uint64) (chunks [][2]uint64) {
	// chunks may be only one block, then from == to
	if from > to || to == 0 {
		return
	}

	total := (to-from)/minBlocks + 1
	chunks = make([][2]uint64, total)

	var cTo uint64
	var i int
	for from <= to {
		if to > from+minBlocks {
			cTo = from + minBlocks - 1
		} else {
			cTo = to
		}

		chunks[i] = [2]uint64{from, cTo}

		from = cTo + 1
		i++
	}

	return chunks
}

// @request for subLedger, will get FileList and Chunk
type subLedgerRequest struct {
	id         uint64 // id & child_id
	from, to   uint64
	_retry     int
	peer       *Peer
	state      reqState
	expiration time.Time
	done       doneCallback
}

func (s *subLedgerRequest) To() uint64 {
	return s.to
}

func (s *subLedgerRequest) SetTo(to uint64) {
	s.to = to
}

func (s *subLedgerRequest) State() reqState {
	return s.state
}

func (s *subLedgerRequest) Retry(ctx *context, err error) {
	select {
	case s.peer.errch <- err:
	default:
	}

	if peer := ctx.retry.retry(s.id, s.to); peer != nil {
		s.peer = peer
		s.Run(ctx)
	} else {
		s.Done(errMissingPeer)
	}
}

func (s *subLedgerRequest) Handle(ctx *context, pkt *p2p.Msg, peer *Peer) {
	if cmd(pkt.Cmd) == FileListCode {
		s.state = reqRespond

		msg := new(message.FileList)
		err := msg.Deserialize(pkt.Payload)
		if err != nil {
			s.Retry(ctx, err)
			return
		}

		if len(msg.Files) != 0 {
			// sort as StartHeight
			sort.Sort(files(msg.Files))
			msgId := ctx.pool.MsgID()
			from := msg.Files[0].StartHeight
			to := msg.Files[len(msg.Files)-1].EndHeight
			if from < s.from {
				from = s.from
			}
			if to > s.to {
				to = s.to
			}

			var blockNumber uint64
			for _, file := range msg.Files {
				blockNumber += file.BlockNumbers
			}

			// request files
			ctx.fc.request(&fileRequest{
				id:         msgId,
				from:       from,
				to:         to,
				files:      msg.Files,
				nonce:      msg.Nonce,
				peer:       peer,
				rec:        ctx.syncer.receiveBlocks,
				expiration: time.Now().Add(u64ToDuration(blockNumber)),
				current:    from,
				done:       s.done,
			})
		}

		// request chunks
		for _, chunk := range msg.Chunks {
			// maybe chunk is too large
			cs := splitChunk(chunk[0], chunk[1])
			for _, c := range cs {
				msgId := ctx.pool.MsgID()

				c := &chunkRequest{
					id:         msgId,
					from:       c[0],
					to:         c[1],
					peer:       peer,
					expiration: time.Now().Add(u64ToDuration(c[1] - c[0])),
					done:       s.done,
				}

				ctx.pool.Add(c)
			}
		}

		s.Done(nil)
	} else {
		s.Retry(ctx, errUnExpectedRes)
	}
}

func (s *subLedgerRequest) ID() uint64 {
	return s.id
}

func (s *subLedgerRequest) Run(*context) {
	err := s.peer.Send(GetSubLedgerCode, s.id, &message.GetSubLedger{
		From:    ledger.HashHeight{Height: s.from},
		Count:   s.to - s.from + 1,
		Forward: true,
	})

	if err != nil {
		s.peer.log.Error(fmt.Sprintf("send GetSubLedgerMsg<%d-%d> to %s error: %v", s.from, s.to, s.peer, err))
		s.Done(err)
	} else {
		s.state = reqPending
		s.peer.log.Info(fmt.Sprintf("send GetSubLedgerMsg<%d-%d> to %s done", s.from, s.to, s.peer))
	}
}

func (s *subLedgerRequest) Done(err error) {
	if err != nil {
		s.state = reqError
	} else {
		s.state = reqDone
	}

	s.done(s, err)
}

func (s *subLedgerRequest) Expired() bool {
	return time.Now().After(s.expiration)
}

func (s *subLedgerRequest) Req() Request {
	return s
}

// @request file
type fileRequest struct {
	id         uint64
	from, to   uint64
	state      reqState
	files      []*ledger.CompressedFileMeta
	nonce      uint64
	peer       *Peer
	rec        receiveBlocks
	expiration time.Time
	current    uint64 // the tallest snapshotBlock have received, as the breakpoint resume
	done       doneCallback
}

func (r *fileRequest) To() uint64 {
	return r.to
}

func (r *fileRequest) SetTo(to uint64) {
	r.to = to
}

func (r *fileRequest) Handle(ctx *context, msg *p2p.Msg, peer *Peer) {
	// nothing
}

func (r *fileRequest) ID() uint64 {
	return r.id
}

func (r *fileRequest) Expired() bool {
	return time.Now().After(r.expiration)
}

func (r *fileRequest) State() reqState {
	return r.state
}

func (r *fileRequest) Req() Request {
	return &subLedgerRequest{
		id:   r.id,
		from: r.from,
		to:   r.to,
		peer: r.peer,
		done: r.done,
	}
}

// file1/0-3600 file2/3601-7200
func (r *fileRequest) String() string {
	files := make([]string, len(r.files))
	for i, file := range r.files {
		files[i] = file.Filename + "/" +
			strconv.FormatUint(file.StartHeight, 10) +
			"-" +
			strconv.FormatUint(file.EndHeight, 10)
	}

	return strings.Join(files, " ")
}

func (r *fileRequest) Done(err error) {
	if err != nil {
		r.state = reqError
	} else {
		r.state = reqDone
	}

	r.done(r, err)
}

func (r *fileRequest) Addr() string {
	return r.peer.FileAddress().String()
}

func (r *fileRequest) Retry(ctx *context, err error) {
	select {
	case r.peer.errch <- err:
	default:
	}

	if peer := ctx.retry.retry(r.id, r.to); peer != nil {
		r.peer = peer
		r.Run(ctx)
	} else {
		r.Done(errMissingPeer)
	}
}

func (r *fileRequest) Run(ctx *context) {
	ctx.fc.request(r)
}

// @request for chunk
type chunkRequest struct {
	id         uint64
	from, to   uint64
	_retry     int
	peer       *Peer
	state      reqState
	expiration time.Time
	done       doneCallback
}

func (c *chunkRequest) To() uint64 {
	return c.to
}

func (c *chunkRequest) SetTo(to uint64) {
	c.to = to
}

func (c *chunkRequest) Req() Request {
	return c
}

func (c *chunkRequest) State() reqState {
	return c.state
}

func (c *chunkRequest) Retry(ctx *context, err error) {
	select {
	case c.peer.errch <- err:
	default:
	}

	if peer := ctx.retry.retry(c.id, c.to); peer != nil {
		c.peer = peer
		c.Run(ctx)
	} else {
		c.Done(errMissingPeer)
	}
}

func (c *chunkRequest) Handle(ctx *context, pkt *p2p.Msg, peer *Peer) {
	if cmd(pkt.Cmd) == SubLedgerCode {
		c.state = reqRespond

		msg := new(message.SubLedger)
		err := msg.Deserialize(pkt.Payload)
		if err != nil {
			fmt.Println("chunkRequest handle error: ", err)
			c.Retry(ctx, err)
			return
		}

		ctx.syncer.receiveBlocks(msg.SBlocks, msg.ABlocks)
		c.Done(nil)
	} else {
		c.Retry(ctx, errUnExpectedRes)
	}
}

func (c *chunkRequest) ID() uint64 {
	return c.id
}

func (c *chunkRequest) Run(*context) {
	err := c.peer.Send(GetChunkCode, c.id, &message.GetChunk{
		Start: c.from,
		End:   c.to,
	})

	if err != nil {
		c.state = reqError
		c.peer.log.Error(fmt.Sprintf("send GetChunkMsg<%d/%d> to %s error: %v", c.from, c.to, c.peer, err))
	} else {
		c.state = reqPending
		c.peer.log.Info(fmt.Sprintf("send GetChunkMsg<%d/%d> to %s done", c.from, c.to, c.peer))
	}
}

func (c *chunkRequest) Done(err error) {
	if err != nil {
		c.state = reqError
	} else {
		c.state = reqDone
	}

	c.done(c, err)
}

func (c *chunkRequest) Expired() bool {
	return time.Now().After(c.expiration)
}

// helper
type files []*ledger.CompressedFileMeta

func (a files) Len() int {
	return len(a)
}

func (a files) Less(i, j int) bool {
	return a[i].StartHeight < a[j].StartHeight
}

func (a files) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// helper
func u64ToDuration(n uint64) time.Duration {
	return time.Duration(int64(n/100) * int64(time.Second))
}
