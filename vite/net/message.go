package net

import (
	"github.com/pkg/errors"
	"time"
)

var errHandshakeTwice = errors.New("handshake should send only once")
var errMsgTimeout = errors.New("message response timeout")

var subledgerTimeout = 10 * time.Second
var accountBlocksTimeout = 30 * time.Second
var snapshotBlocksTimeout = time.Minute

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

//func (t cmd) match(t2 cmd) bool {
//	switch t {
//	case HandshakeCode:
//		return t2 == HandshakeCode
//	case GetSubLedgerCode:
//		return t2 == FileListCode || t2 == SubLedgerCode || t2 == ForkCode
//	case GetSnapshotBlockHeadersCode:
//		return t2 == SnapshotBlockHeadersCode || t2 == ForkCode
//	case GetSnapshotBlockBodiesCode:
//		return t2 == SnapshotBlockBodiesCode || t2 == ForkCode
//	case GetSnapshotBlocksCode:
//		return t2 == SnapshotBlocksCode || t2 == ForkCode
//	case GetSnapshotBlocksByHashCode:
//		return t2 == SnapshotBlocksCode || t2 == ForkCode
//	case GetAccountBlocksCode:
//		return t2 == AccountBlocksCode || t2 == ForkCode
//	case GetMultiAccountBlocksCode:
//		return t2 == AccountBlocksCode || t2 == ForkCode
//	case GetAccountBlocksByHashCode:
//		return t2 == AccountBlocksCode || t2 == ForkCode
//	default:
//		return false
//	}
//}
