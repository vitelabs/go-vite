package net

import (
	"fmt"
	"sync"
	"time"

	"github.com/vitelabs/go-vite/vitepb"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/interfaces"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/tools/list"
)

type msgHandler interface {
	name() string
	codes() []Code
	handle(msg Msg) error
}

type msgHandlers struct {
	_name    string
	handlers map[Code]msgHandler
}

func newHandlers(name string) *msgHandlers {
	return &msgHandlers{
		_name:    name,
		handlers: make(map[Code]msgHandler),
	}
}

func (m msgHandlers) name() string {
	return m._name
}

func (m msgHandlers) codes() (codes []Code) {
	for c := range m.handlers {
		codes = append(codes, c)
	}

	return
}

func (m msgHandlers) handle(msg Msg) error {
	if handler, ok := m.handlers[msg.Code]; ok {
		return handler.handle(msg)
	}

	return nil
}

func (m msgHandlers) register(h msgHandler) error {
	for _, c := range h.codes() {
		if _, ok := m.handlers[c]; ok {
			return fmt.Errorf("handler for code %d has existed", c)
		}
		m.handlers[c] = h
	}

	return nil
}

func (m msgHandlers) unregister(h msgHandler) (err error) {
	var codes []Code

	for _, c := range h.codes() {
		if _, ok := m.handlers[c]; ok {
			delete(m.handlers, c)
		} else {
			codes = append(codes, c)
		}
	}

	if len(codes) > 0 {
		return fmt.Errorf("handler for codes %v not exist", codes)
	}

	return nil
}

// @section queryHandler
type queryHandler struct {
	*msgHandlers
	lock  sync.RWMutex
	queue list.List
	term  chan struct{}
	wg    sync.WaitGroup
	log   log15.Logger
}

func newQueryHandler(chain Chain) (q *queryHandler, err error) {
	q = &queryHandler{
		msgHandlers: newHandlers("query"),
		queue:       list.New(),
		log:         netLog.New("module", "query"),
	}

	if err = q.register(&getSnapshotBlocksHandler{chain}); err != nil {
		return nil, err
	}
	if err = q.register(&getAccountBlocksHandler{chain}); err != nil {
		return nil, err
	}
	if err = q.register(&checkHandler{chain, netLog.New("module", "checkHandler")}); err != nil {
		return nil, err
	}

	return q, nil
}

func (q *queryHandler) start() {
	q.term = make(chan struct{})

	q.wg.Add(1)
	go q.loop()
}

func (q *queryHandler) stop() {
	if q.term == nil {
		return
	}

	select {
	case <-q.term:
	default:
		close(q.term)
		q.wg.Wait()
	}
}

func (q *queryHandler) handle(msg Msg) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	//if q.queue.Size() > 2000 {
	//	q.log.Warn(fmt.Sprintf("query queue is more than 1000, discard message from %s", msg.Sender.String()))
	//	return nil
	//}
	q.queue.Append(msg)

	return nil
}

func (q *queryHandler) loop() {
	defer q.wg.Done()

	const batch = 10
	var tasks = make([]Msg, batch)
	var index = 0
	var ele interface{}

	for {
		select {
		case <-q.term:
			return
		default:
			// next
		}

		q.lock.Lock()
		for index, ele = 0, q.queue.Shift(); ele != nil; ele = q.queue.Shift() {
			tasks[index] = ele.(Msg)
			index++
			if index >= batch {
				break
			}
		}
		q.lock.Unlock()

		if index == 0 {
			time.Sleep(10 * time.Millisecond)
		} else {
			now := time.Now().Unix()
			for _, msg := range tasks[:index] {
				if now-msg.ReceivedAt > 20 {
					q.log.Warn(fmt.Sprintf("fetch message from %s is expired", msg.Sender))
					continue
				}
				// allocate to handlers
				if err := q.msgHandlers.handle(msg); err != nil {
					msg.Sender.catch(err)
				}
			}
		}
	}
}

type checkHandler struct {
	chain interface {
		GetSnapshotBlockByHeight(height uint64) (*ledger.SnapshotBlock, error)
		GetLedgerReaderByHeight(startHeight uint64, endHeight uint64) (cr interfaces.LedgerReader, err error)
	}
	log log15.Logger
}

func (c *checkHandler) name() string {
	return "Check"
}

func (c *checkHandler) codes() []Code {
	return []Code{CodeGetHashList}
}

// return [startHeight+1 ... end]
func (c *checkHandler) handleGetHashHeightList(get *GetHashHeightList) (code Code, payload Serializable) {
	var block *ledger.SnapshotBlock
	var err error
	for _, hh := range get.From {
		block, err = c.chain.GetSnapshotBlockByHeight(hh.Height)
		if err != nil || block == nil {
			c.log.Warn(fmt.Sprintf("failed to find snapshotblock %s/%d", hh.Hash, hh.Height))
			continue
		}
		if block.Hash != hh.Hash {
			c.log.Warn(fmt.Sprintf("snapshotblock at %d is %s not %s", hh.Height, block.Hash, hh.Hash))
			continue
		}

		break
	}

	if block == nil {
		return CodeException, ExpMissing
	}

	step := get.Step
	start := (block.Height + 1) / step * step
	if start < block.Height+1 {
		start = block.Height + 1
	}

	var points []*HashHeightPoint
	for start <= get.To {
		block, err = c.chain.GetSnapshotBlockByHeight(start)
		if err != nil || block == nil {
			c.log.Warn(fmt.Sprintf("failed to find snapshotblock at %d", start))
			break
		}

		points = append(points, &HashHeightPoint{
			HashHeight: ledger.HashHeight{
				Height: block.Height,
				Hash:   block.Hash,
			},
		})

		if start == get.To {
			break
		}

		start = (start + step) / step * step
		if start > get.To {
			start = get.To
		}
	}

	if len(points) == 0 {
		return CodeException, ExpMissing
	}

	var reader interfaces.LedgerReader
	start = points[0].Height
	for i := 1; i < len(points); i++ {
		point := points[i]
		reader, err = c.chain.GetLedgerReaderByHeight(start, point.Height)
		if err != nil {
			break
		}
		point.Size = uint64(reader.Size())
		start = point.Height + 1
	}

	return CodeHashList, &HashHeightPointList{
		Points: points,
	}
}

func (c *checkHandler) handle(msg Msg) (err error) {
	var get = &GetHashHeightList{}
	err = get.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	cd, payload := c.handleGetHashHeightList(get)
	return msg.Sender.send(cd, msg.Id, payload)
}

type getSnapshotBlocksHandler struct {
	chain snapshotBlockReader
}

func (s *getSnapshotBlocksHandler) name() string {
	return "GetSnapshotBlocks"
}

func (s *getSnapshotBlocksHandler) codes() []Code {
	return []Code{CodeGetSnapshotBlocks}
}

func (s *getSnapshotBlocksHandler) handle(msg Msg) (err error) {
	defer monitor.LogTime("net", "handle_GetSnapshotBlocksMsg", time.Now())

	req := new(GetSnapshotBlocks)

	if err = req.Deserialize(msg.Payload); err != nil {
		msg.Recycle()
		return
	}
	msg.Recycle()

	netLog.Info(fmt.Sprintf("receive %s from %s", req, msg.Sender))

	var block *ledger.SnapshotBlock
	if req.From.Hash != types.ZERO_HASH {
		block, err = s.chain.GetSnapshotBlockByHash(req.From.Hash)
	} else {
		block, err = s.chain.GetSnapshotBlockByHeight(req.From.Height)
	}

	if err != nil || block == nil {
		netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, msg.Sender, err))
		return msg.Sender.send(CodeException, msg.Id, ExpMissing)
	}

	// use for split
	var from, to uint64
	if req.Forward {
		from = block.Height
		to = from + req.Count - 1
	} else {
		to = block.Height
		if to >= req.Count {
			from = to - req.Count + 1
		} else {
			from = 0
		}
	}

	chunks := splitChunk(from, to, syncTaskSize)

	var blocks []*ledger.SnapshotBlock
	for _, c := range chunks {
		blocks, err = s.chain.GetSnapshotBlocksByHeight(c[0], true, c[1]-c[0]+1)
		if err != nil || len(blocks) == 0 {
			netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, msg.Sender, err))
			return msg.Sender.send(CodeException, msg.Id, ExpMissing)
		}

		if err = msg.Sender.sendSnapshotBlocks(blocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send %d SnapshotBlocks to %s error: %v", len(blocks), msg.Sender, err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send %d SnapshotBlocks [%s/%d - %s/%d] to %s done", len(blocks), blocks[0].Hash, blocks[0].Height, blocks[len(blocks)-1].Hash, blocks[len(blocks)-1].Height, msg.Sender))
		}
	}

	return
}

// @section get account blocks
type getAccountBlocksHandler struct {
	chain accountBockReader
}

func (a *getAccountBlocksHandler) name() string {
	return "GetAccountBlocks Handler"
}

func (a *getAccountBlocksHandler) codes() []Code {
	return []Code{CodeGetAccountBlocks}
}

var ZERO_ADDRESS types.Address
var errGetABlocksMissingParam = errors.New("missing param to GetAccountBlocks")

func (a *getAccountBlocksHandler) handle(msg Msg) (err error) {
	defer monitor.LogTime("net", "handle_GetAccountBlocksMsg", time.Now())

	req := new(GetAccountBlocks)

	if err = req.Deserialize(msg.Payload); err != nil {
		msg.Recycle()
		return
	}
	msg.Recycle()

	netLog.Info(fmt.Sprintf("receive %s from %s", req, msg.Sender))

	var block *ledger.AccountBlock
	if req.From.Hash != types.ZERO_HASH {
		// only need hash
		block, err = a.chain.GetAccountBlockByHash(req.From.Hash)
	} else if req.Address == ZERO_ADDRESS {
		// missing start hash and address, so we can`t handle it
		return errGetABlocksMissingParam
	} else {
		// address and height
		block, err = a.chain.GetAccountBlockByHeight(req.Address, req.From.Height)
	}

	if err != nil || block == nil {
		netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, msg.Sender, err))
		return msg.Sender.send(CodeException, msg.Id, ExpMissing)
	}

	address := block.AccountAddress

	// use for split
	var from, to uint64
	if req.Forward {
		from = block.Height
		to = from + req.Count - 1
	} else {
		to = block.Height
		if to >= req.Count {
			from = to - req.Count + 1
		} else {
			from = 0
		}
	}

	chunks := splitChunk(from, to, syncTaskSize)

	var blocks []*ledger.AccountBlock
	for _, c := range chunks {
		blocks, err = a.chain.GetAccountBlocksByHeight(address, c[1], c[1]-c[0]+1)
		if err != nil || len(blocks) == 0 {
			netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, msg.Sender, err))
			return msg.Sender.send(CodeException, msg.Id, ExpMissing)
		}

		if err = msg.Sender.sendAccountBlocks(blocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send %d AccountBlocks to %s error: %v", len(blocks), msg.Sender, err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send %d AccountBlocks [%s/%d - %s/%d] to %s done", len(blocks), blocks[0].Hash, blocks[0].Height, blocks[len(blocks)-1].Hash, blocks[len(blocks)-1].Height, msg.Sender))
		}
	}

	return
}

// helper
type accountBlockMap = map[types.Address][]*ledger.AccountBlock

func countAccountBlocks(mblocks accountBlockMap) (count uint64) {
	for _, blocks := range mblocks {
		for range blocks {
			count++
		}
	}

	return
}

func splitAccountMap(mblocks accountBlockMap) (ret [][]*ledger.AccountBlock) {
	const batch = 1000
	var index, end int
	var add bool
	s := make([]*ledger.AccountBlock, 0, batch)

	for _, blocks := range mblocks {
		index = 0

		for index < len(blocks) {
			slotLen := cap(s) - len(s)
			if len(blocks[index:]) <= slotLen {
				s = append(s, blocks[index:]...)
				add = false
				break
			} else {
				end = index + slotLen
				s = append(s, blocks[index:end]...)
				index = end

				ret = append(ret, s)
				add = true
				s = make([]*ledger.AccountBlock, 0, batch)
			}
		}
	}

	if !add && len(s) != 0 {
		ret = append(ret, s)
	}

	return
}

type stateHandler struct {
	maxNeighbors int
	peers        *peerSet
}

func (s stateHandler) name() string {
	return "state"
}

func (s stateHandler) codes() []Code {
	return []Code{CodeHeartBeat}
}

func (s stateHandler) handle(msg Msg) (err error) {
	var heartbeat = &vitepb.State{}

	err = proto.Unmarshal(msg.Payload, heartbeat)
	if err != nil {
		return
	}

	var head types.Hash
	head, err = types.BytesToHash(heartbeat.Head)
	if err != nil {
		return
	}

	msg.Sender.SetState(head, heartbeat.Height)

	// max 100 neighbors
	var count = len(heartbeat.Peers)
	if count > maxNeighbors {
		count = maxNeighbors
	}
	var pl = make([]peerConn, count)
	for i := 0; i < count; i++ {
		hp := heartbeat.Peers[i]
		pl[i] = peerConn{
			id:  hp.ID,
			add: hp.Status != vitepb.State_Disconnected,
		}
	}
	msg.Sender.setPeers(pl, heartbeat.Patch)

	return
}

func splitChunk(from, to uint64, size uint64) (chunks [][2]uint64) {
	// chunks may be only one block, then from == to
	if from > to || to == 0 {
		return
	}

	total := (to-from)/size + 1
	chunks = make([][2]uint64, total)

	var cTo uint64
	var i int
	for from <= to {
		if cTo = from + size - 1; cTo > to {
			cTo = to
		}

		chunks[i] = [2]uint64{from, cTo}

		from = cTo + 1
		i++
	}

	return chunks[:i]
}
