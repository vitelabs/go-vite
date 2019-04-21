package net

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/tools/list"
	"github.com/vitelabs/go-vite/vite/net/message"
)

type code = p2p.Code

const (
	GetSnapshotBlocksCode code = iota
	GetAccountBlocksCode       // query single AccountChain
	SnapshotBlocksCode
	AccountBlocksCode
	NewSnapshotBlockCode
	NewAccountBlockCode

	ExceptionCode = 127
)

//var msgNames = [...]string{
//	GetSnapshotBlocksCode: "GetSnapshotBlocksMsg",
//	GetAccountBlocksCode:  "GetAccountBlocksMsg",
//	GetChunkCode:          "GetChunkMsg",
//	SnapshotBlocksCode:    "SnapshotBlocksMsg",
//	AccountBlocksCode:     "AccountBlocksMsg",
//	NewSnapshotBlockCode:  "NewSnapshotBlockMsg",
//	NewAccountBlockCode:   "NewAccountBlockMsg",
//}
//
//func (t code) String() string {
//	if t == ExceptionCode {
//		return "ExceptionMsg"
//	}
//
//	if t > NewAccountBlockCode {
//		return "Unknown Message"
//	}
//
//	return msgNames[t]
//}

type msgHandler interface {
	name() string
	codes() []code
	handle(msg p2p.Msg, sender Peer) error
}

type msgHandlers struct {
	_name    string
	handlers map[code]msgHandler
}

func newHandlers(name string) *msgHandlers {
	return &msgHandlers{
		_name:    name,
		handlers: make(map[code]msgHandler),
	}
}

func (m msgHandlers) name() string {
	return m._name
}

func (m msgHandlers) codes() (codes []code) {
	for c := range m.handlers {
		codes = append(codes, c)
	}

	return
}

func (m msgHandlers) handle(msg p2p.Msg, sender Peer) error {
	if handler, ok := m.handlers[msg.Code]; ok {
		return handler.handle(msg, sender)
	}

	return p2p.PeerUnknownMessage
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
	var codes []code

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
}

func newQueryHandler(chain Chain) (q *queryHandler, err error) {
	q = &queryHandler{
		msgHandlers: newHandlers("query"),
		queue:       list.New(),
	}

	if err = q.register(&getSnapshotBlocksHandler{chain}); err != nil {
		return nil, err
	}
	if err = q.register(&getAccountBlocksHandler{chain}); err != nil {
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

type queryTask struct {
	msg    p2p.Msg
	sender Peer
}

func (q *queryHandler) handle(msg p2p.Msg, sender Peer) error {
	q.lock.Lock()
	q.queue.Append(queryTask{msg, sender})
	q.lock.Unlock()

	return nil
}

func (q *queryHandler) loop() {
	defer q.wg.Done()

	const batch = 10
	var tasks = make([]queryTask, batch)
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
			tasks[index] = ele.(queryTask)
			index++
			if index >= batch {
				break
			}
		}
		q.lock.Unlock()

		if index == 0 {
			time.Sleep(20 * time.Millisecond)
		} else {
			for _, event := range tasks[:index] {
				// allocate to handlers
				if err := q.msgHandlers.handle(event.msg, event.sender); err != nil {
					event.sender.catch(err)
				}
			}
		}
	}
}

type getSnapshotBlocksHandler struct {
	chain snapshotBlockReader
}

func (s *getSnapshotBlocksHandler) name() string {
	return "GetSnapshotBlocks"
}

func (s *getSnapshotBlocksHandler) codes() []code {
	return []code{GetSnapshotBlocksCode}
}

func (s *getSnapshotBlocksHandler) handle(msg p2p.Msg, sender Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetSnapshotBlocksMsg", time.Now())

	req := new(message.GetSnapshotBlocks)

	if err = req.Deserialize(msg.Payload); err != nil {
		return
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req, sender))

	var block *ledger.SnapshotBlock
	if req.From.Hash != types.ZERO_HASH {
		block, err = s.chain.GetSnapshotBlockByHash(req.From.Hash)
	} else {
		block, err = s.chain.GetSnapshotBlockByHeight(req.From.Height)
	}

	if err != nil || block == nil {
		netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender, err))
		return sender.send(ExceptionCode, msg.Id, message.Missing)
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
			netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender, err))
			return sender.send(ExceptionCode, msg.Id, message.Missing)
		}

		if err = sender.sendSnapshotBlocks(blocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send %d SnapshotBlocks to %s error: %v", len(blocks), sender, err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send %d SnapshotBlocks [%s/%d - %s/%d] to %s done", len(blocks), blocks[0].Hash, blocks[0].Height, blocks[len(blocks)-1].Hash, blocks[len(blocks)-1].Height, sender))
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

func (a *getAccountBlocksHandler) codes() []code {
	return []code{GetAccountBlocksCode}
}

var nilAddress = types.Address{}
var errGetABlocksMissingParam = errors.New("missing param to GetAccountBlocks")

func (a *getAccountBlocksHandler) handle(msg p2p.Msg, sender Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetAccountBlocksMsg", time.Now())

	req := new(message.GetAccountBlocks)

	if err = req.Deserialize(msg.Payload); err != nil {
		return
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req, sender))

	var block *ledger.AccountBlock
	if req.From.Hash != types.ZERO_HASH {
		// only need hash
		block, err = a.chain.GetAccountBlockByHash(req.From.Hash)
	} else if req.Address == nilAddress {
		// missing start hash and address, so we can`t handle it
		return errGetABlocksMissingParam
	} else {
		// address and height
		block, err = a.chain.GetAccountBlockByHeight(req.Address, req.From.Height)
	}

	if err != nil || block == nil {
		netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender, err))
		return sender.send(ExceptionCode, msg.Id, message.Missing)
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
			netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender, err))
			return sender.send(ExceptionCode, msg.Id, message.Missing)
		}

		if err = sender.sendAccountBlocks(blocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send %d AccountBlocks to %s error: %v", len(blocks), sender, err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send %d AccountBlocks [%s/%d - %s/%d] to %s done", len(blocks), blocks[0].Hash, blocks[0].Height, blocks[len(blocks)-1].Hash, blocks[len(blocks)-1].Height, sender))
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
