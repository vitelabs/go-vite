package net

import (
	"context"
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
	"github.com/vitelabs/go-vite/vite/net/message"
)

const CHUNK_SIZE = 50
const maxBlocksOneTrip = 1000
const maxFilesOneTrip = 1000
const chunkTimeout = 6 * time.Second

func splitChunk(from, to, chunkSize uint64) (chunks [][2]uint64) {
	// chunks may be only one block, then from == to
	if from > to || to == 0 {
		return
	}

	total := (to-from)/chunkSize + 1
	chunks = make([][2]uint64, total)

	var cTo uint64
	var i int
	for from <= to {
		if cTo = from + chunkSize - 1; cTo > to {
			cTo = to
		}

		chunks[i] = [2]uint64{from, cTo}

		from = cTo + 1
		i++
	}

	return chunks[:i]
}

func splitChunkCount(from, to, chunkSize, count uint64) (chunks [][2]uint64) {
	to2 := from + chunkSize*count - 1
	if to > to2 {
		to = to2
	}

	return splitChunk(from, to, chunkSize)
}

type ChunkReqStatus struct {
	From, To uint64
	Done     bool
	Pieces   chunkResponses
	Deadline string
	Target   string
}

type chunkRequest struct {
	id       uint64
	from, to uint64
	msg      message.GetChunk
	deadline time.Time
	pieces   chunkResponses
	mu       sync.Mutex
	ch       chan error
	once     sync.Once
	ctx      context.Context
	// for status
	_done  bool
	target string
}

type ChunkResponse struct {
	From, To uint64
}

type chunkResponses []ChunkResponse

func (crs chunkResponses) Len() int {
	return len(crs)
}

func (crs chunkResponses) Less(i, j int) bool {
	return crs[i].From < crs[j].From
}

func (crs chunkResponses) Swap(i, j int) {
	crs[i], crs[j] = crs[j], crs[i]
}

func (cr *chunkRequest) expired(t time.Time) bool {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	return cr.deadline.Before(t)
}

func (cr *chunkRequest) done(err error) {
	cr.once.Do(func() {
		if err != nil {
			cr._done = false
		} else {
			cr._done = true
		}
		cr.ch <- err
	})
}

func (cr *chunkRequest) receive(crs ChunkResponse) (rest [][2]uint64, done bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.pieces = append(cr.pieces, crs)
	sort.Sort(cr.pieces)

	done = true
	from := cr.from
	for i := 0; i < len(cr.pieces); i++ {
		piece := cr.pieces[i]
		// useless
		if piece.To < from {
			continue
		}

		// missing front piece
		if piece.From > from {
			done = false
			rest = append(rest, [2]uint64{
				from + 1,
				piece.From - 1,
			})
		}

		// next response
		from = piece.To + 1
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

func (cr *chunkRequest) status() ChunkReqStatus {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	return ChunkReqStatus{
		From:     cr.from,
		To:       cr.to,
		Done:     cr._done,
		Pieces:   cr.pieces,
		Deadline: cr.deadline.Format("2006-1-2 15:3:4"),
		Target:   cr.target,
	}
}

type subLedger struct {
	*p2p.Msg
	sender Peer
}

type chunkPool struct {
	peers *peerSet
	gid   MsgIder

	chunks sync.Map // uint64: *chunkRequest

	handler blockReceiver

	resQueue blockQueue.BlockQueue

	running int32
	term    chan struct{}
	wg      sync.WaitGroup

	log log15.Logger
}

type ChunkPoolStatus struct {
	Request []ChunkReqStatus
}

func (p *chunkPool) download(ctx context.Context, from, to uint64) <-chan error {
	cr := &chunkRequest{
		id:   p.gid.MsgID(),
		from: from,
		to:   to,
		ch:   make(chan error, 1),
		msg:  message.GetChunk{from, to},
		ctx:  ctx,
	}

	p.chunks.Store(cr.id, cr)
	p.request(cr)

	return cr.ch
}

func newChunkPool(peers *peerSet, gid MsgIder, handler blockReceiver) *chunkPool {
	return &chunkPool{
		peers:    peers,
		gid:      gid,
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
	p.resQueue.Push(subLedger{msg, sender})
	return nil
}

func (p *chunkPool) start() {
	if atomic.CompareAndSwapInt32(&p.running, 0, 1) {
		p.term = make(chan struct{})

		p.wg.Add(1)
		common.Go(p.checkLoop)

		p.wg.Add(1)
		common.Go(p.handleLoop)
	}
}

func (p *chunkPool) stop() {
	if atomic.CompareAndSwapInt32(&p.running, 1, 0) {
		if p.term == nil {
			return
		}

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
				p.log.Error(fmt.Sprintf("handle SubLedgerMsg from %s error: %v", leg.sender.RemoteAddr(), err))
				leg.sender.Report(err)
				v, ok := p.chunks.Load(leg.Id)
				if ok {
					go p.request(v.(*chunkRequest))
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

	var res ChunkResponse
	if start < end {
		for i, block := range chunk.SBlocks {
			if block.Height != start+uint64(i) {
				return
			}
		}
		res.From, res.To = start, end
	} else {
		for i, block := range chunk.SBlocks {
			if block.Height != start-uint64(i) {
				return
			}
		}
		res.From, res.To = end, start
	}

	if v, ok := p.chunks.Load(leg.Id); ok {
		request := v.(*chunkRequest)
		_, done := request.receive(res)
		if done {
			p.chunks.Delete(leg.Id)
			request.done(nil)
			p.log.Info(fmt.Sprintf("chunkRequest<%d-%d> done", request.from, request.to))
		}
	}

	return
}

func (p *chunkPool) checkLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(chunkTimeout / 2)
	defer ticker.Stop()

	check := func(key, value interface{}) bool {
		c := value.(*chunkRequest)

		select {
		case <-c.ctx.Done():
			// chunk request is canceled
			p.chunks.Delete(key)
			c.done(nil)
			p.log.Info(fmt.Sprintf("chunkRequest<%d-%d> is canceled", c.from, c.to))

		default:
			if time.Now().After(c.deadline) {
				p.request(c)
				p.log.Info(fmt.Sprintf("chunkRequest<%d-%d> is timeout", c.from, c.to))
			}
		}

		return true
	}

Loop:
	for {
		select {
		case <-p.term:
			break Loop

		case <-ticker.C:
			p.chunks.Range(check)
		}
	}

	del := func(key, value interface{}) bool {
		c := value.(*chunkRequest)
		c.done(nil)
		p.chunks.Delete(key)
		return true
	}

	p.chunks.Range(del)
}

func (p *chunkPool) request(c *chunkRequest) {
	ps := p.peers.Pick(c.to)

	if len(ps) == 0 {
		c.done(errNoSuitablePeers)
		return
	}

	p1 := ps[rand.Intn(len(ps))]
	c.deadline = time.Now().Add(chunkTimeout)
	err := p1.Send(GetChunkCode, c.id, &c.msg)
	if err != nil {
		p.log.Error(fmt.Sprintf("send %s to %s error: %v", c.msg.String(), p1.RemoteAddr(), err))
	} else {
		c.target = p1.RemoteAddr().String()
		p.log.Info(fmt.Sprintf("send %s to %s", c.msg.String(), p1.RemoteAddr()))
	}
}

func (p *chunkPool) status() ChunkPoolStatus {
	var request []ChunkReqStatus

	p.chunks.Range(func(key, value interface{}) bool {
		cr := value.(*chunkRequest)
		request = append(request, cr.status())
		return true
	})

	return ChunkPoolStatus{
		Request: request,
	}
}
