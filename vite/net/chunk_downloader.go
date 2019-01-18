package net

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitelabs/go-vite/log15"

	"github.com/vitelabs/go-vite/vite/net/blockQueue"

	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/list"
	"github.com/vitelabs/go-vite/vite/net/message"
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

<<<<<<< HEAD:vite/net/request.go
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

type filePiece struct {
	from, to uint64
}

func (f *filePiece) band() (from, to uint64) {
	return f.from, f.to
}

func (f *filePiece) setBand(from, to uint64) {
	f.from, f.to = from, to
}

type blockReceiver interface {
	receiveSnapshotBlock(block *ledger.SnapshotBlock, sender Peer) (err error)
	receiveAccountBlock(block *ledger.AccountBlock, sender Peer) (err error)
	catch(piece)
	done(p piece)
}

const file2Chunk = 600
const minSubLedger = 1000

const chunk = 20
=======
const chunkSize = 50
>>>>>>> fileDownloader:vite/net/chunk_downloader.go
const maxBlocksOneTrip = 1000
const chunkTimeout = 20 * time.Second

func splitChunk(from, to uint64, chunk uint64) (chunks [][2]uint64) {
	// chunks may be only one block, then from == to
	if from > to || to == 0 {
		return
	}

	total := (to-from)/chunk + 1
	chunks = make([][2]uint64, total)

	var cTo uint64
	var i int
	for from <= to {
		if cTo = from + chunk - 1; cTo > to {
			cTo = to
		}

		chunks[i] = [2]uint64{from, cTo}

		from = cTo + 1
		i++
	}

	return chunks[:i]
}

type chunkRequest struct {
	id       uint64
	from, to uint64
	peer     Peer
	state    reqState
	deadline time.Time
	ret      chunkResponses
	mu       sync.Mutex
	ch       chan error
	busy     bool // is receiving response
}

type chunkResponse struct {
	from, to uint64
}
type chunkResponses []chunkResponse

func (crs chunkResponses) Len() int {
	return len(crs)
}

func (crs chunkResponses) Less(i, j int) bool {
	return crs[i].from < crs[j].from
}

func (crs chunkResponses) Swap(i, j int) {
	crs[i], crs[j] = crs[j], crs[i]
}

func (cr *chunkRequest) beginReceiving() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.busy = true
}

func (cr *chunkRequest) receive(res chunkResponse) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.ret = append(cr.ret, res)
	cr.busy = false
}

func (cr *chunkRequest) done() (rest [][2]uint64, done bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	sort.Sort(cr.ret)

	done = true
	from := cr.from
	for i := 0; i < len(cr.ret); i++ {
		res := cr.ret[i]
		// useless
		if res.to < from {
			continue
		}

		// missing front piece
		if res.from > from {
			done = false
			rest = append(rest, [2]uint64{
				from + 1,
				res.from - 1,
			})
		}

		// next response
		from = res.to + 1
	}

	// from should equal (cr.to + 1)
	if from-1 < cr.to {
		done = false
		rest = append(rest, [2]uint64{
			from,
			cr.to,
		})
	}

	return
}

type subLedger struct {
	*p2p.Msg
	sender Peer
}

type chunkPool struct {
	peers *peerSet
	gid   MsgIder

	mu    sync.Mutex
	queue list.List // wait to request

	chunks sync.Map

	handler blockReceiver
	term    chan struct{}
	wg      sync.WaitGroup

	resQueue blockQueue.BlockQueue

	log log15.Logger
}

func (p *chunkPool) download(from, to uint64) ([][2]uint64, error) {
	panic("implement me")
}

func newChunkPool(peers *peerSet, gid MsgIder, handler blockReceiver) *chunkPool {
	return &chunkPool{
		peers:    peers,
		gid:      gid,
		queue:    list.New(),
		handler:  handler,
		resQueue: blockQueue.New(),
		log:      log15.New("module", "chunkDownloader"),
	}
}

func (p *chunkPool) ID() string {
	return "chunk pool"
}

func (p *chunkPool) Cmds() []ViteCmd {
	return []ViteCmd{SubLedgerCode}
}

func (p *chunkPool) Handle(msg *p2p.Msg, sender Peer) error {
	cmd := ViteCmd(msg.Cmd)

	p.log.Info(fmt.Sprintf("receive %s from %s", cmd, sender.RemoteAddr()))

	p.resQueue.Push(subLedger{msg, sender})

	return nil
}

func (p *chunkPool) start() {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})

		p.wg.Add(1)
		common.Go(p.loop)

		p.wg.Add(1)
		common.Go(p.handleLoop)
	}
}

func (p *chunkPool) stop() {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		select {
		case <-p.term:
		default:
			close(p.term)
			p.resQueue.Close()
			p.wg.Wait()
		}
	}
}

func (p *chunkPool) handleLoop() {
	defer p.wg.Done()

	for {
		select {
		case <-p.term:
			return

		default:
			v := p.resQueue.Pop()
			if v == nil {
				break
			}

			leg := v.(subLedger)

			if err := p.handleResponse(leg); err != nil {
				p.retry(res.msg.Id)
			} else {
				if c := p.chunk(res.msg.Id); c != nil {
					c.count += uint64(len(subLedger.SBlocks))

					if c.count >= c.to-c.from+1 {
						p.done(res.msg.Id)
					}
				}
			}
		}
	}
}

func (p *chunkPool) handleResponse(leg subLedger) (err error) {
	chunk := new(message.SubLedger)
	if err = chunk.Deserialize(leg.Payload); err != nil {
		return
	}

	// receive account blocks first
	for _, block := range chunk.ABlocks {
		if err = p.handler.receiveAccountBlock(block); err != nil {
			return
		}
	}

	if len(chunk.SBlocks) == 0 {
		return
	}

	for _, block := range chunk.SBlocks {
		if err = p.handler.receiveSnapshotBlock(block); err != nil {
			return
		}
	}

	start, end := chunk.SBlocks[0].Height, chunk.SBlocks[len(chunk.SBlocks)-1].Height

	var res chunkResponse
	if start < end {
		for i, block := range chunk.SBlocks {
			if block.Height != start+uint64(i) {
				return
			}
		}
		res.from, res.to = start, end
	} else {
		for i, block := range chunk.SBlocks {
			if block.Height != start-uint64(i) {
				return
			}
		}
		res.from, res.to = end, start
	}

	if v, ok := p.chunks.Load(leg.Id); ok {
		request := v.(*chunkRequest)
		request.receive(res)
	}

	return
}

func (p *chunkPool) loop() {
	defer p.wg.Done()

	ticker := time.NewTicker(chunkTimeout)
	defer ticker.Stop()

	doTicker := time.NewTicker(time.Second)
	defer doTicker.Stop()

	var state reqState
	var id uint64
	var c *chunkRequest

loop:
	for {
		select {
		case <-p.term:
			break loop

		case <-doTicker.C:
			for {
				p.mu.Lock()
				if ele := p.queue.Shift(); ele != nil {
					p.mu.Unlock()
					c := ele.(*chunkRequest)
					if c.from <= p.to {
						p.chunks.Store(c.id, c)
						p.request(c)
					} else {
						// put back
						p.mu.Lock()
						p.queue.UnShift(c)
						p.mu.Unlock()
						break
					}
				} else {
					p.mu.Unlock()
					break
				}
			}

		case now := <-ticker.C:
			p.chunks.Range(func(key, value interface{}) bool {
				id, c = key.(uint64), value.(*chunkRequest)
				state = c.state
				if state == reqPending && now.After(c.deadline) {
					p.retry(id)
				}
				return true
			})
		}
	}
}

func (p *chunkPool) chunk(id uint64) *chunkRequest {
	v, ok := p.chunks.Load(id)

	if ok {
		return v.(*chunkRequest)
	}

	return nil
}

func (p *chunkPool) add(from, to uint64, front bool) {
	cs := splitChunk(from, to, 50)
	crs := make([]*chunkRequest, len(cs))

	for i, c := range cs {
		cr := &chunkRequest{from: c[0], to: c[1]}
		cr.id = p.gid.MsgID()
		cr.msg = &message.GetChunk{
			Start: cr.from,
			End:   cr.to,
		}
		crs[i] = cr
	}

	if front {
		p.mu.Lock()
		for i := len(crs) - 1; i > -1; i-- {
			p.queue.UnShift(crs[i])
		}
		p.mu.Unlock()
	} else {
		p.mu.Lock()
		for _, cr := range crs {
			p.queue.Append(cr)
		}
		p.mu.Unlock()
	}

	p.start()
}

func (p *chunkPool) done(id uint64) {
	if c, ok := p.chunks.Load(id); ok {
		p.chunks.Delete(id)
		p.handler.done(c.(*chunkRequest))
	}
}

func (p *chunkPool) request(c *chunkRequest) {
	if c.peer == nil {
		ps := p.peers.Pick(c.to)
		if len(ps) == 0 {
			p.catch(c)
			return
		}
		c.peer = ps[rand.Intn(len(ps))]
	}

	p.send(c)
}

func (p *chunkPool) retry(id uint64) {
	v, ok := p.chunks.Load(id)

	if ok {
		c := v.(*chunkRequest)
		if c == nil {
			return
		}

		old := c.peer
		c.peer = nil

		if ps := p.peers.Pick(c.to); len(ps) > 0 {
			for _, p := range ps {
				if p != old {
					c.peer = p
					break
				}
			}
		}

		if c.peer == nil {
			p.chunks.Delete(c.id)
			p.catch(c)
		} else {
			p.send(c)
		}
	}
}

func (p *chunkPool) catch(c *chunkRequest) {
	c.state = reqError
	p.handler.catch(c)
}

func (p *chunkPool) send(c *chunkRequest) {
	c.deadline = time.Now().Add(chunkTimeout)
	c.state = reqPending
	c.peer.Send(GetChunkCode, c.id, c.msg)
}
