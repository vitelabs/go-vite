package net

import (
	"fmt"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/list"
	"github.com/vitelabs/go-vite/vite/net/message"
	"math/rand"
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
	if s > reqCancel {
		return "unknown request state"
	}
	return reqStatus[s]
}

type context interface {
	Add(r Request)
	Retry(id uint64, err error)
	FC() *fileClient
	Get(id uint64) Request
	Del(id uint64)
}

type Request interface {
	Handle(ctx context, msg *p2p.Msg, peer Peer)
	ID() uint64
	SetID(id uint64)
	Run(ctx context)
	Done(ctx context)
	Catch(err error)
	Expired() bool
	State() reqState
	Req() Request
	Band() (from, to uint64)
	SetBand(from, to uint64)
	SetPeer(peer Peer)
	Peer() Peer
}

type piece interface {
	band() (from, to uint64)
	setBand(from, to uint64)
}

type blockReceiver interface {
	receiveSnapshotBlock(block *ledger.SnapshotBlock)
	receiveAccountBlock(block *ledger.AccountBlock)
	catch(piece)
}

const maxBlocks = 300 // max blocks in one message(snapshotblocks + accountblocks)
const file2Chunk = 600
const minSubLedger = 1000

func splitChunk(from, to uint64) (chunks [][2]uint64) {
	// chunks may be only one block, then from == to
	if from > to || to == 0 {
		return
	}

	total := (to-from)/maxBlocks + 1
	chunks = make([][2]uint64, total)

	var cTo uint64
	var i int
	for from <= to {
		if cTo = from + maxBlocks - 1; cTo > to {
			cTo = to
		}

		chunks[i] = [2]uint64{from, cTo}

		from = cTo + 1
		i++
	}

	return chunks[:i]
}

var chunkTimeout = 3 * time.Minute

// @request for chunk
type chunkRequest struct {
	id       uint64
	from, to uint64
	peer     Peer
	state    reqState
	deadline time.Time
	msg      *message.GetChunk
}

func (c *chunkRequest) setBand(from, to uint64) {
	c.from, c.to = from, to
}

func (c *chunkRequest) band() (from, to uint64) {
	return c.from, c.to
}

type chunkPool struct {
	lock      sync.RWMutex
	peers     *peerSet
	gid       MsgIder
	queue     *list.List
	chunks    map[uint64]*chunkRequest
	handler   blockReceiver
	doneChan  chan uint64
	retryChan chan uint64
	slots     chan struct{}
	term      chan struct{}
	wg        sync.WaitGroup
}

func newChunkPool(peers *peerSet, gid MsgIder, handler blockReceiver) *chunkPool {
	return &chunkPool{
		peers:     peers,
		gid:       gid,
		queue:     list.New(),
		chunks:    make(map[uint64]*chunkRequest),
		handler:   handler,
		doneChan:  make(chan uint64),
		retryChan: make(chan uint64, 1),
		slots:     make(chan struct{}, 5),
	}
}

func (p *chunkPool) ID() string {
	return "chunk pool"
}

func (p *chunkPool) Cmds() []ViteCmd {
	return []ViteCmd{SubLedgerCode}
}

func (p *chunkPool) Handle(msg *p2p.Msg, sender Peer) error {
	if ViteCmd(msg.Cmd) == SubLedgerCode {
		res := new(message.SubLedger)

		if err := res.Deserialize(msg.Payload); err != nil {
			netLog.Error(fmt.Sprintf("descerialize %s from %s error: %v", res, sender.RemoteAddr(), err))
			p._retry(msg.Id)
			return err
		}

		netLog.Info(fmt.Sprintf("receive %s from %s", res, sender.RemoteAddr()))

		// receive account blocks first
		for _, block := range res.ABlocks {
			p.handler.receiveAccountBlock(block)
		}

		for _, block := range res.SBlocks {
			p.handler.receiveSnapshotBlock(block)
		}

		p._done(msg.Id)
	} else {
		p._retry(msg.Id)
	}

	return nil
}

func (p *chunkPool) start() {
	p.term = make(chan struct{})

	p.wg.Add(1)
	common.Go(p.loop)
}

func (p *chunkPool) stop() {
	if p.term == nil {
		return
	}

	select {
	case <-p.term:
	default:
		close(p.term)
		p.wg.Wait()
	}
}

func (p *chunkPool) loop() {
	defer p.wg.Done()

	ticker := time.NewTicker(chunkTimeout)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-p.term:
			break loop

		case id := <-p.retryChan:
			p.retry(id)

		case id := <-p.doneChan:
			delete(p.chunks, id)

			if p.slots != nil {
				<-p.slots
			}

		case now := <-ticker.C:
			var state reqState
			for id, c := range p.chunks {
				state = c.state
				if state == reqPending && now.After(c.deadline) {
					p._retry(id)
				}

				if state == reqError {
					delete(p.chunks, id)
				}
			}

		case p.slots <- struct{}{}:
			if ele := p.queue.Shift(); ele != nil {
				c := ele.(*chunkRequest)
				p.chunks[c.id] = c
				p.request(c)
			} else {
				<-p.slots
				time.Sleep(200 * time.Millisecond)
			}
		}
	}

	// clear
	for id := range p.chunks {
		delete(p.chunks, id)
	}

	p.slots = nil
}

func (p *chunkPool) add(c *chunkRequest) {
	c.id = p.gid.MsgID()
	p.queue.Append(c)
	c.msg = &message.GetChunk{
		Start: c.from,
		End:   c.to,
	}
}

func (p *chunkPool) _retry(id uint64) {
	select {
	case <-p.term:
	case p.retryChan <- id:
	}
}

func (p *chunkPool) _done(id uint64) {
	select {
	case <-p.term:
	case p.doneChan <- id:
	}
}

func (p *chunkPool) request(c *chunkRequest) {
	if c.peer == nil {
		peers := p.peers.Pick(c.to)
		if len(peers) == 0 {
			p.catch(c)
			return
		}
		c.peer = peers[rand.Intn(len(peers))]
	}

	p.do(c)
}

func (p *chunkPool) retry(id uint64) {
	if c, ok := p.chunks[id]; ok {
		old := c.peer
		c.peer = nil

		peers := p.peers.Pick(c.to)
		if len(peers) > 0 {
			for _, peer := range peers {
				if peer != old {
					c.peer = peer
					break
				}
			}
		}

		if c.peer == nil {
			p.catch(c)
		} else {
			p.do(c)
		}
	}
}

func (p *chunkPool) catch(c *chunkRequest) {
	c.state = reqError
	p.handler.catch(c)
	<-p.slots
}

func (p *chunkPool) do(c *chunkRequest) {
	c.deadline = time.Now().Add(chunkTimeout)
	c.state = reqPending
	c.peer.Send(GetChunkCode, c.id, c.msg)
}

// helper
type files []*ledger.CompressedFileMeta

func (f files) Len() int {
	return len(f)
}

func (f files) Less(i, j int) bool {
	return f[i].StartHeight < f[j].StartHeight
}

func (f files) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// must sort first
func (f files) unique() files {
	j := 0
	for i, file := range f {
		if file.Filename != f[j].Filename {
			f[j] = f[i]
			j++
		}
	}

	return f[:j]
}

// helper
func u64ToDuration(n uint64) time.Duration {
	return time.Duration(int64(n/1000) * int64(time.Second))
}
