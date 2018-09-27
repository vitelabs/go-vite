package net

import (
	"fmt"
	"github.com/ethereum/go-ethereum/p2p/netutil"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vite/net/message"
	"time"
)

var errHandshakeTwice = errors.New("handshake should send only once")
var errMsgTimeout = errors.New("message response timeout")

var subledgerTimeout = 10 * time.Second
var accountBlocksTimeout = 30 * time.Second
var snapshotBlocksTimeout = time.Minute

const maxStepHashCount = 1000
const hashStep = 20

// @section Cmd
const CmdSetName = "vite"

var cmdSets = []uint64{2}

type cmd uint64

const (
	HandshakeCode cmd = iota
	StatusCode
	ForkCode // tell peer it has forked, use for respond GetSnapshotBlocksCode
	GetSubLedgerCode
	GetSnapshotBlocksCode	// get snapshotblocks without content
	GetSnapshotBlocksContentCode
	GetFullSnapshotBlocksCode	// get snapshotblocks with content
	GetSnapshotBlocksByHashCode	// a batch of hash
	GetSnapshotBlocksContentByHashCode
	GetFullSnapshotBlocksByHashCode
	GetAccountBlocksCode       // query single AccountChain
	GetMultiAccountBlocksCode  // query multi AccountChain
	GetAccountBlocksByHashCode // query accountBlocks by hashList
	GetFileCode
	GetChunkCode
	SubLedgerCode
	FileListCode
	SnapshotBlocksCode
	SnapshotBlocksContentCode
	FullSnapshotBlocksCode
	AccountBlocksCode
	NewSnapshotBlockCode

	ExceptionCode = 127
)

var msgNames = [...]string{
	HandshakeCode:               "HandShakeMsg",
	StatusCode:                  "StatusMsg",
	ForkCode:                    "ForkMsg",
	GetSubLedgerCode:            "GetSubLedgerMsg",
	GetSnapshotBlocksCode: "GetSnapshotBlocksMsg",
	GetSnapshotBlocksContentCode:  "GetSnapshotBlocksContentMsg",
	GetFullSnapshotBlocksCode:       "GetFullSnapshotBlocksMsg",
	GetSnapshotBlocksByHashCode: "GetSnapshotBlocksByHashMsg",
	GetSnapshotBlocksContentByHashCode: "GetSnapshotBlocksContentByHashMsg",
	GetFullSnapshotBlocksByHashCode: "GetFullSnapshotBlocksByHashMsg",
	GetAccountBlocksCode:        "GetAccountBlocksMsg",
	GetMultiAccountBlocksCode:   "GetMultiAccountBlocksMsg",
	GetAccountBlocksByHashCode:  "GetAccountBlocksByHashMsg",
	GetFileCode:                 "GetFileMsg",
	GetChunkCode: "GetChunkMsg",
	SubLedgerCode:               "SubLedgerMsg",
	FileListCode:                "FileListMsg",
	SnapshotBlocksCode:    "SnapshotBlocksMsg",
	SnapshotBlocksContentCode:     "SnapshotBlocksContentMsg",
	FullSnapshotBlocksCode:          "FullSnapshotBlocksMsg",
	AccountBlocksCode:           "AccountBlocksMsg",
	NewSnapshotBlockCode:        "NewSnapshotBlockMsg",
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
type _statusHandler	func (msg *p2p.Msg, sender *Peer) error
func statusHandler(msg *p2p.Msg, sender *Peer) error {
	status := new(ledger.HashHeight)
	err := status.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	sender.SetHead(status.Hash, status.Height)
	return nil
}

func (s _statusHandler) ID() string {
	return "default status handler"
}

func (s _statusHandler) Cmds() []cmd {
	return []cmd{StatusCode}
}

func (s _statusHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	return s(msg, sender)
}

// @section forkHandler
type forkHandler struct {
	chain Chain
}

func (f *forkHandler) ID() string {
	return "default fork handler"
}

func (f *forkHandler) Cmds() []cmd {
	return []cmd{ForkCode}
}

func (f *forkHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	fork := new(message.Fork)
	err := fork.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	high := fork.List[0]
	ids, err := f.chain.GetAbHashList(high.Height, uint64(len(fork.List)), hashStep, false)

	var commonID *ledger.HashHeight
	if err != nil {
		// can`t get hashList, use binary search to find commonID
		low, high := 0, len(fork.List)
		for low < high {
			mid := (low + high) / 2
			id := fork.List[mid]
			_, err = f.chain.GetSnapshotBlocksByHash(&id.Hash, 1, true, false)
			if err == nil {
				commonID = id
				high = mid - 1
			} else {
				low = mid + 1
			}
		}
	} else {
		for i, id := range ids {
			pid := fork.List[i]
			if id.Equal(pid.Hash, pid.Height) {
				commonID = id
				break
			}
		}
	}

	if commonID != nil {
		// get commonID, resend the request message
		return sender.GetSnapshotBlocks(&message.GetSnapshotBlocks{
			From:    commonID,
			Count:   sender.height - commonID.Height,
			Forward: true,
		})
	} else {
		// can`t get commonID, then getSnapshotBlocks from the lowest block
		// last item is lowest
		lowest := fork.List[len(fork.List)-1].Height
		return sender.GetSnapshotBlocks(&message.GetSnapshotBlocks{
			From:    &ledger.HashHeight{Height: lowest,},
			Count:   sender.height - lowest,
			Forward: true,
		})
	}
}

// @section getSubLedgerHandler
type getSubLedgerHandler struct {
	chain Chain
}

func (s *getSubLedgerHandler) ID() string {
	return "default GetSubLedger Handler"
}

func (s *getSubLedgerHandler) Cmds() []cmd {
	return []cmd{GetSubLedgerCode}
}

func (s *getSubLedgerHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	req := new(message.GetSnapshotBlocks)
	err := req.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	var files []*message.File
	var batch [][2]uint64
	if req.From.Height != 0 {
		files, batch = s.chain.GetSubLedgerByHeight(req.From.Height, req.Count, req.Forward)
	} else {
		files, batch, err = s.chain.GetSubLedgerByHash(&req.From.Hash, req.Count, req.Forward)
	}

	if err != nil {
		return sender.Send(ExceptionCode, msg.Id, message.Missing)
	} else {
		return sender.Send(FileListCode, msg.Id, &message.FileList{
			Files: files,
			Chunk: batch,
			Nonce: 0,
		})
	}
}

type getSnapshotBlocksHandler struct {
	chain Chain
}

func (s *getSnapshotBlocksHandler) ID() string {
	return "default GetSnapshotBlocks Handler"
}

func (s *getSnapshotBlocksHandler) Cmds() []cmd {
	return []cmd{GetSnapshotBlocksCode}
}

func (s *getSnapshotBlocksHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	req := new(message.GetSnapshotBlocks)
	err := req.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	var blocks []*ledger.SnapshotBlock
	if req.From.Height != 0 {
		blocks, err = s.chain.GetSnapshotBlocksByHeight(req.From.Height, req.Count, req.Forward, false)
	} else {
		blocks, err = s.chain.GetSnapshotBlocksByHash(&req.From.Hash, req.Count, req.Forward, false)
	}

	var ids []*ledger.HashHeight
	if err != nil {
		count := req.From.Height / hashStep
		if count > maxStepHashCount {
			count = maxStepHashCount
		}
		// from high to low
		ids, err = s.chain.GetAbHashList(req.From.Height, count, hashStep, false)
		if err != nil {
			return err
		}

		return sender.SendFork(&message.Fork{
			List: ids,
		}, msg.Id)
	} else {
		return sender.SendSnapshotBlocks(blocks, msg.Id)
	}
}

type getAccountBlocksHandler struct {
	chain Chain
}

func (a *getAccountBlocksHandler) ID() string {
	return "default GetAccountBlocks Handler"
}

func (a *getAccountBlocksHandler) Cmds() []cmd {
	return []cmd{GetAccountBlocksCode}
}

func (a *getAccountBlocksHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	as := new(message.GetAccountBlocks)
	err := as.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	var blocks []*ledger.AccountBlock
	if as.From.Height != 0 {
		blocks, err = a.chain.GetAccountBlocksByHeight(as.Address, as.From.Height, as.Count, as.Forward)
	} else {
		blocks, err = a.chain.GetAccountBlocksByHash(as.Address, &as.From.Hash, as.Count, as.Forward)
	}

	if err != nil {
		return sender.SendAccountBlocks(as.Address, blocks, msg.Id)
	} else {
		return sender.Send(ExceptionCode, msg.Id, message.Missing)
	}
}

// @section getChunkHandler
type getChunkHandler struct {
	chain Chain
}

func (c *getChunkHandler) ID() string {
	return "default GetChunk Handler"
}

func (c *getChunkHandler) Cmds() []cmd {
	return []cmd{GetChunkCode}
}

func (c *getChunkHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	req := new(message.Chunk)
	err := req.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	sblocks, mblocks, err := c.chain.GetConfirmSubLedger(req.Start, req.End)
	if err == nil {
		return sender.SendSubLedger(&message.SubLedger{
			SBlocks: sblocks,
			ABlocks: mblocks,
		}, msg.Id)
	} else {
		return sender.Send(ExceptionCode, msg.Id, message.Missing)
	}

	return nil
}

// @section blocks
type BlockReceiver interface {
	receiveSnapshotBlocks(blocks []*ledger.SnapshotBlock)
	receiveAccountBlocks(blocks map[types.Address][]*ledger.AccountBlock)
}

type blocksHandler struct {
	rec BlockReceiver
}

func (s *blocksHandler) ID() string {
	return "default blocks Handler"
}

func (s *blocksHandler) Cmds() []cmd {
	return []cmd{SubLedgerCode, SnapshotBlocksCode, AccountBlocksCode}
}

func (s *blocksHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	var sblocks []*ledger.SnapshotBlock
	var mblocks map[types.Address][]*ledger.AccountBlock

	switch cmd(msg.Cmd) {
	case SubLedgerCode:
		subledger := new(message.SubLedger)
		err := subledger.Deserialize(msg.Payload)
		if err != nil {
			return err
		}
		sblocks = subledger.SBlocks
		mblocks = subledger.ABlocks
	case SnapshotBlocksCode:
		blocks := new(message.SnapshotBlocks)

		err := blocks.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		sblocks = blocks.Blocks
	case AccountBlocksCode:
		blocks := new(message.AccountBlocks)

		err := blocks.Deserialize(msg.Payload)
		if err != nil {
			return err
		}

		mblocks = make(map[types.Address][]*ledger.AccountBlock)
		mblocks[blocks.Address] = blocks.Blocks
	}

	for _, block := range sblocks {
		sender.SeeBlock(block.Hash)
	}
	for _, ablocks := range mblocks {
		for _, ablock := range ablocks {
			sender.SeeBlock(ablock.Hash)
		}
	}

	s.rec.receiveSnapshotBlocks(sblocks)
	s.rec.receiveAccountBlocks(mblocks)

	return nil
}

type newblockReceiver interface {
	BroadcastSnapshotBlock(block *ledger.SnapshotBlock)
	receiveNewBlock(block *ledger.SnapshotBlock)
}

type newSnapshotBlockHandler struct {
	rec newblockReceiver
}

func (s *newSnapshotBlockHandler) ID() string {
	return "default new snapshotblocks Handler"
}

func (s *newSnapshotBlockHandler) Cmds() []cmd {
	return []cmd{NewSnapshotBlockCode}
}

func (s *newSnapshotBlockHandler) Handle(msg *p2p.Msg, sender *Peer) error {
	block := new(ledger.SnapshotBlock)
	err := block.Deserialize(msg.Payload)
	if err != nil {
		return err
	}

	sender.SeeBlock(block.Hash)
	s.rec.BroadcastSnapshotBlock(block)
	s.rec.receiveNewBlock(block)

	return nil
}
