package net

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/monitor"
	"time"

	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
)

//var errHandshakeTwice = errors.New("handshake should send only once")

var subledgerTimeout = 10 * time.Second

//var accountBlocksTimeout = 30 * time.Second
//var snapshotBlocksTimeout = time.Minute

// @section Cmd
const CmdSetName = "vite"

const CmdSet = 2

type cmd uint32

const (
	HandshakeCode cmd = iota
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

func (t cmd) String() string {
	if t == ExceptionCode {
		return "ExceptionMsg"
	}
	return msgNames[t]
}

type MsgHandler interface {
	ID() string
	Cmds() []cmd
	Handle(msg *p2p.Msg, sender *Peer) error
}

// @section statusHandler
type _statusHandler func(msg *p2p.Msg, sender *Peer) error

func statusHandler(msg *p2p.Msg, sender *Peer) error {
	defer monitor.LogTime("net", "handle_StatusMsg", time.Now())

	status := new(ledger.HashHeight)
	err := status.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	sender.SetHead(status.Hash, status.Height)
	return nil
}

func (s _statusHandler) ID() string {
	return "status handler"
}

func (s _statusHandler) Cmds() []cmd {
	return []cmd{StatusCode}
}

func (s _statusHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	return s(msg, sender)
}

// @section getSubLedgerHandler
type getSubLedgerHandler struct {
	chain Chain
}

func (s *getSubLedgerHandler) ID() string {
	return "GetSubLedger Handler"
}

func (s *getSubLedgerHandler) Cmds() []cmd {
	return []cmd{GetSubLedgerCode}
}

func (s *getSubLedgerHandler) Handle(msg *p2p.Msg, sender *Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetSubledgerMsg", time.Now())

	req := new(message.GetSnapshotBlocks)
	err = req.Deserialize(msg.Payload)
	if err != nil {
		return
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req.String(), sender.RemoteAddr()))

	var files []*ledger.CompressedFileMeta
	var chunks [][2]uint64
	if req.From.Hash != types.ZERO_HASH {
		files, chunks, err = s.chain.GetSubLedgerByHash(&req.From.Hash, req.Count, req.Forward)
	} else {
		files, chunks = s.chain.GetSubLedgerByHeight(req.From.Height, req.Count, req.Forward)
	}

	if err != nil || (len(files) == 0 && len(chunks) == 0) {
		netLog.Error(fmt.Sprintf("handle %s from %s error: %v", req.String(), sender.RemoteAddr(), err))
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

func (s *getSnapshotBlocksHandler) Cmds() []cmd {
	return []cmd{GetSnapshotBlocksCode}
}

func (s *getSnapshotBlocksHandler) Handle(msg *p2p.Msg, sender *Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetSnapshotBlocksMsg", time.Now())

	req := new(message.GetSnapshotBlocks)
	err = req.Deserialize(msg.Payload)
	if err != nil {
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
		netLog.Error(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
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
			netLog.Error(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
			return sender.Send(ExceptionCode, msg.Id, message.Missing)
		}

		if err = sender.SendSnapshotBlocks(blocks, msg.Id); err != nil {
			netLog.Error(fmt.Sprintf("send %d SnapshotBlocks to %s error: %v", len(blocks), sender.RemoteAddr(), err))
			return
		} else {
			netLog.Info(fmt.Sprintf("send %d SnapshotBlocks to %s done", len(blocks), sender.RemoteAddr()))
		}
	}

	return
}

type getAccountBlocksHandler struct {
	chain Chain
}

func (a *getAccountBlocksHandler) ID() string {
	return "GetAccountBlocks Handler"
}

func (a *getAccountBlocksHandler) Cmds() []cmd {
	return []cmd{GetAccountBlocksCode}
}

var NULL_ADDRESS = types.Address{}
var errGetABlocksMissingParam = errors.New("missing param to GetAccountBlocks")

func (a *getAccountBlocksHandler) Handle(msg *p2p.Msg, sender *Peer) (err error) {
	defer monitor.LogTime("net", "handle_GetAccountBlocksMsg", time.Now())

	req := new(message.GetAccountBlocks)
	err = req.Deserialize(msg.Payload)
	if err != nil {
		return
	}

	netLog.Info(fmt.Sprintf("receive %s from %s", req, sender.RemoteAddr()))

	var block *ledger.AccountBlock
	if req.From.Hash != types.ZERO_HASH {
		block, err = a.chain.GetAccountBlockByHash(&req.From.Hash)
	} else if req.Address == NULL_ADDRESS {
		return errGetABlocksMissingParam
	} else {
		block, err = a.chain.GetAccountBlockByHeight(&req.Address, req.From.Height)
	}

	if err != nil || block == nil {
		netLog.Error(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
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
			netLog.Error(fmt.Sprintf("handle %s from %s error: %v", req, sender.RemoteAddr(), err))
			return sender.Send(ExceptionCode, msg.Id, message.Missing)
		}

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

func (c *getChunkHandler) Cmds() []cmd {
	return []cmd{GetChunkCode}
}

func (c *getChunkHandler) Handle(msg *p2p.Msg, sender *Peer) (err error) {
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
