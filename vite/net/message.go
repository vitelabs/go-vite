package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/monitor"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/p2p/list"
	"github.com/vitelabs/go-vite/vite/net/message"
	"sync"
	"time"
)

// @section Cmd
const Vite = "vite"

const CmdSet = 2

type ViteCmd p2p.Cmd

const (
	HandshakeCode ViteCmd = iota
	StatusCode
	ForkCode // tell peer it has forked, use for respond GetSnapshotBlocksCode
	GetSubLedgerCode
	GetSnapshotBlocksCode // get snapshotblocks without content
	GetSnapshotBlocksContentCode
	GetFullSnapshotBlocksCode   // get snapshotblocks with content
	GetSnapshotBlocksByHashCode // a batch of hash
	GetSnapshotBlocksContentByHashCode
	GetFullSnapshotBlocksByHashCode
	GetAccountBlocksCode       // query single AccountChain
	GetMultiAccountBlocksCode  // query multi AccountChain
	GetAccountBlocksByHashCode // query accountBlocks by hashList
	GetFilesCode
	GetChunkCode
	SubLedgerCode
	FileListCode
	SnapshotBlocksCode
	SnapshotBlocksContentCode
	FullSnapshotBlocksCode
	AccountBlocksCode
	NewSnapshotBlockCode
	NewAccountBlockCode

	ExceptionCode = 127
)

var msgNames = [...]string{
	HandshakeCode:                      "HandShakeMsg",
	StatusCode:                         "StatusMsg",
	ForkCode:                           "ForkMsg",
	GetSubLedgerCode:                   "GetSubLedgerMsg",
	GetSnapshotBlocksCode:              "GetSnapshotBlocksMsg",
	GetSnapshotBlocksContentCode:       "GetSnapshotBlocksContentMsg",
	GetFullSnapshotBlocksCode:          "GetFullSnapshotBlocksMsg",
	GetSnapshotBlocksByHashCode:        "GetSnapshotBlocksByHashMsg",
	GetSnapshotBlocksContentByHashCode: "GetSnapshotBlocksContentByHashMsg",
	GetFullSnapshotBlocksByHashCode:    "GetFullSnapshotBlocksByHashMsg",
	GetAccountBlocksCode:               "GetAccountBlocksMsg",
	GetMultiAccountBlocksCode:          "GetMultiAccountBlocksMsg",
	GetAccountBlocksByHashCode:         "GetAccountBlocksByHashMsg",
	GetFilesCode:                       "GetFileMsg",
	GetChunkCode:                       "GetChunkMsg",
	SubLedgerCode:                      "SubLedgerMsg",
	FileListCode:                       "FileListMsg",
	SnapshotBlocksCode:                 "SnapshotBlocksMsg",
	SnapshotBlocksContentCode:          "SnapshotBlocksContentMsg",
	FullSnapshotBlocksCode:             "FullSnapshotBlocksMsg",
	AccountBlocksCode:                  "AccountBlocksMsg",
	NewSnapshotBlockCode:               "NewSnapshotBlockMsg",
	NewAccountBlockCode:                "NewAccountBlockMsg",
}

func (t ViteCmd) String() string {
	if t == ExceptionCode {
		return "ExceptionMsg"
	}

	if t > NewAccountBlockCode {
		return "UnkownMsg"
	}

	return msgNames[t]
}

type MsgHandler interface {
	ID() string
	Cmds() []ViteCmd
	Handle(msg *p2p.Msg, sender Peer) error
}

// @section statusHandler
type _statusHandler func(msg *p2p.Msg, sender Peer) error

func statusHandler(msg *p2p.Msg, sender Peer) error {
	defer monitor.LogTime("net", "handle_StatusMsg", time.Now())

	status := new(ledger.HashHeight)

	if err := status.Deserialize(msg.Payload); err != nil {
		return err
	}

	sender.SetHead(status.Hash, status.Height)
	return nil
}

func (s _statusHandler) ID() string {
	return "status handler"
}

func (s _statusHandler) Cmds() []ViteCmd {
	return []ViteCmd{StatusCode}
}

func (s _statusHandler) Handle(msg *p2p.Msg, sender Peer) error {
	return s(msg, sender)
}

// @section queryHandler
type queryHandler struct {
	lock     sync.RWMutex
	queue    *list.List
	handlers map[ViteCmd]MsgHandler
	term     chan struct{}
	wg       sync.WaitGroup
}

func newQueryHandler(chain Chain) *queryHandler {
	q := &queryHandler{
		handlers: make(map[ViteCmd]MsgHandler),
		queue:    list.New(),
	}

	q.addHandler(&getSubLedgerHandler{chain})
	q.addHandler(&getSnapshotBlocksHandler{chain})
	q.addHandler(&getAccountBlocksHandler{chain})
	q.addHandler(&getChunkHandler{chain})

	return q
}

func (q *queryHandler) start() {
	q.term = make(chan struct{})

	q.wg.Add(1)
	common.Go(q.loop)
}

func (q *queryHandler) stop() {
	select {
	case <-q.term:
	default:
		close(q.term)
		q.wg.Wait()
	}
}

func (q *queryHandler) addHandler(handler MsgHandler) {
	for _, cmd := range handler.Cmds() {
		q.handlers[cmd] = handler
	}
}

func (q *queryHandler) ID() string {
	return "query handler"
}

func (q *queryHandler) Cmds() []ViteCmd {
	return []ViteCmd{GetSubLedgerCode, GetSnapshotBlocksCode, GetAccountBlocksCode, GetChunkCode}
}

type queryTask struct {
	Msg    *p2p.Msg `json:"Msg"`
	Sender Peer     `json:"Sender"`
}

func (e *queryTask) Recycle() {
	e.Msg = nil
	e.Sender = nil
	msgEventPool.Put(e)
}

var msgEventPool = &sync.Pool{
	New: func() interface{} {
		return &queryTask{}
	},
}

func newMsgEvent() *queryTask {
	//v := msgEventPool.Get()
	//return v.(*queryTask)
	return &queryTask{}
}

func (q *queryHandler) Handle(msg *p2p.Msg, sender Peer) error {
	e := newMsgEvent()
	e.Msg = msg
	e.Sender = sender

	q.lock.Lock()
	q.queue.Append(e)
	q.lock.Unlock()

	netLog.Info(fmt.Sprintf("put message %s into queue, rest %d query tasks in queue", ViteCmd(msg.Cmd), q.queue.Size()))

	return nil
}

func (q *queryHandler) loop() {
	defer q.wg.Done()

	const batch = 10
	tasks := make([]*queryTask, batch)
	index := 0
	var ele interface{}

	for {
		q.lock.Lock()
		for index, ele = 0, q.queue.Shift(); ele != nil; ele = q.queue.Shift() {
			tasks[index] = ele.(*queryTask)
			index++
			if index >= batch {
				break
			}
		}
		q.lock.Unlock()

		if index == 0 {
			time.Sleep(200 * time.Millisecond)
		} else {
			netLog.Info(fmt.Sprintf("retrive %d query tasks", index))

			for _, event := range tasks[:index] {
				cmd := ViteCmd(event.Msg.Cmd)
				if h, ok := q.handlers[cmd]; ok {
					if err := h.Handle(event.Msg, event.Sender); err != nil {
						event.Sender.Report(err)
					}
				}
			}
		}
	}
}

// @section getSubLedgerHandler
type getSubLedgerHandler struct {
	chain Chain
}

func (s *getSubLedgerHandler) ID() string {
	return "GetSubLedger Handler"
}

func (s *getSubLedgerHandler) Cmds() []ViteCmd {
	return []ViteCmd{GetSubLedgerCode}
}

func (s *getSubLedgerHandler) Handle(msg *p2p.Msg, sender Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetSubledgerMsg", time.Now())

	req := new(message.GetSnapshotBlocks)

	if err = req.Deserialize(msg.Payload); err != nil {
		return
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req, sender.RemoteAddr()))

	var files []*ledger.CompressedFileMeta
	var chunks [][2]uint64
	if req.From.Hash != types.ZERO_HASH {
		files, chunks, err = s.chain.GetSubLedgerByHash(&req.From.Hash, req.Count, req.Forward)
	} else {
		files, chunks = s.chain.GetSubLedgerByHeight(req.From.Height, req.Count, req.Forward)
	}

	if err != nil || (len(files) == 0 && len(chunks) == 0) {
		netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
		return sender.Send(ExceptionCode, msg.Id, message.Missing)
	}

	fileList := &message.FileList{
		Files:  files,
		Chunks: chunks,
		Nonce:  0,
	}

	err = sender.Send(FileListCode, msg.Id, fileList)

	if err != nil {
		netLog.Error(fmt.Sprintf("send %s to %s error: %v", fileList, sender.RemoteAddr(), err))
	} else {
		netLog.Info(fmt.Sprintf("send %s to %s done", fileList, sender.RemoteAddr()))
	}

	return
}

type getSnapshotBlocksHandler struct {
	chain Chain
}

func (s *getSnapshotBlocksHandler) ID() string {
	return "GetSnapshotBlocks"
}

func (s *getSnapshotBlocksHandler) Cmds() []ViteCmd {
	return []ViteCmd{GetSnapshotBlocksCode}
}

func (s *getSnapshotBlocksHandler) Handle(msg *p2p.Msg, sender Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetSnapshotBlocksMsg", time.Now())

	req := new(message.GetSnapshotBlocks)

	if err = req.Deserialize(msg.Payload); err != nil {
		return
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req, sender.RemoteAddr()))

	var block *ledger.SnapshotBlock
	if req.From.Hash != types.ZERO_HASH {
		block, err = s.chain.GetSnapshotBlockByHash(&req.From.Hash)
	} else {
		block, err = s.chain.GetSnapshotBlockByHeight(req.From.Height)
	}

	if err != nil || block == nil {
		netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
		return sender.Send(ExceptionCode, msg.Id, message.Missing)
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
	chunks := splitChunk(from, to)

	var blocks []*ledger.SnapshotBlock
	for _, chunk := range chunks {
		blocks, err = s.chain.GetSnapshotBlocksByHeight(chunk[0], chunk[1]-chunk[0]+1, true, true)
		if err != nil || len(blocks) == 0 {
			netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
			monitor.LogEvent("net/handle", "GetSnapshotBlocks_Fail")
			return sender.Send(ExceptionCode, msg.Id, message.Missing)
		}
		monitor.LogEvent("net/handle", "GetSnapshotBlocks_Success")

		if err = sender.SendSnapshotBlocks(blocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send %d SnapshotBlocks to %s error: %v", len(blocks), sender.RemoteAddr(), err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send %d SnapshotBlocks to %s done", len(blocks), sender.RemoteAddr()))
		}
	}

	return
}

// @section get account blocks
type getAccountBlocksHandler struct {
	chain Chain
}

func (a *getAccountBlocksHandler) ID() string {
	return "GetAccountBlocks Handler"
}

func (a *getAccountBlocksHandler) Cmds() []ViteCmd {
	return []ViteCmd{GetAccountBlocksCode}
}

var NULL_ADDRESS = types.Address{}
var errGetABlocksMissingParam = errors.New("missing param to GetAccountBlocks")

func (a *getAccountBlocksHandler) Handle(msg *p2p.Msg, sender Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetAccountBlocksMsg", time.Now())

	req := new(message.GetAccountBlocks)

	if err = req.Deserialize(msg.Payload); err != nil {
		return
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req, sender.RemoteAddr()))

	var block *ledger.AccountBlock
	if req.From.Hash != types.ZERO_HASH {
		// only need hash
		block, err = a.chain.GetAccountBlockByHash(&req.From.Hash)
	} else if req.Address == NULL_ADDRESS {
		// missing start hash and address, so we can`t handle it
		return errGetABlocksMissingParam
	} else {
		// address and height
		block, err = a.chain.GetAccountBlockByHeight(&req.Address, req.From.Height)
	}

	if err != nil || block == nil {
		netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
		monitor.LogEvent("net/handle", "GetAccountBlocks_Fail")
		return sender.Send(ExceptionCode, msg.Id, message.Missing)
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

	chunks := splitChunk(from, to)

	var blocks []*ledger.AccountBlock
	for _, chunk := range chunks {
		blocks, err = a.chain.GetAccountBlocksByHeight(address, chunk[0], chunk[1]-chunk[0]+1, true)
		if err != nil || len(blocks) == 0 {
			netLog.Warn(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
			monitor.LogEvent("net/handle", "GetAccountBlocks_Fail")
			return sender.Send(ExceptionCode, msg.Id, message.Missing)
		}

		monitor.LogEvent("net/handle", "GetAccountBlocks_Success")

		if err = sender.SendAccountBlocks(blocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send %d AccountBlocks to %s error: %v", len(blocks), sender.RemoteAddr(), err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send %d AccountBlocks to %s done", len(blocks), sender.RemoteAddr()))
		}
	}

	return
}

// @section getChunkHandler
type getChunkHandler struct {
	chain Chain
}

func (c *getChunkHandler) ID() string {
	return "GetChunk Handler"
}

func (c *getChunkHandler) Cmds() []ViteCmd {
	return []ViteCmd{GetChunkCode}
}

func (c *getChunkHandler) Handle(msg *p2p.Msg, sender Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetChunkMsg", time.Now())

	req := new(message.GetChunk)
	err = req.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req, sender.RemoteAddr()))

	start, end := req.Start, req.End
	if start > end {
		start, end = end, start
	}

	// split chunk
	chunks := splitChunk(start, end)

	var sblocks []*ledger.SnapshotBlock
	var mblocks accountBlockMap
	for _, chunk := range chunks {
		sblocks, mblocks, err = c.chain.GetConfirmSubLedger(chunk[0], chunk[1])

		if err != nil || len(sblocks) == 0 {
			netLog.Error(fmt.Sprintf("query chunk<%d-%d> error: %v", chunk[0], chunk[1], err))
			return sender.Send(ExceptionCode, msg.Id, message.Missing)
		}

		ablockCount := countAccountBlocks(mblocks)
		ablocks := make([]*ledger.AccountBlock, 0, ablockCount)
		for _, blocks := range mblocks {
			ablocks = append(ablocks, blocks...)
		}

		if err = sender.SendSubLedger(sblocks, ablocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send Chunk<%d-%d>(%d SnapshotBlocks %d AccountBlocks) to %s error: %v", chunk[0], chunk[1], len(sblocks), ablockCount, sender.RemoteAddr(), err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send Chunk<%d-%d>(%d SnapshotBlocks %d AccountBlocks) to %s done", chunk[0], chunk[1], len(sblocks), ablockCount, sender.RemoteAddr()))
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
