package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"math/big"
)

// @section MsgTyp
type MsgTyp int

const (
	StatusMsg MsgTyp = iota
	GetSubLedgerMsg
	GetSnapshotBlockHeadersMsg
	GetSnapshotBlockBodiesMsg
	GetSnapshotBlockMsg
	GetAccountBlocksMsg
	SubLedgerMsg
	SnapshotBlockHeadersMsg
	SnapshotBlockBodiesMsg
	SnapshotBlockMsg
	AccountBlocksMsg

	ExceptionMsg = 127
)

var msgStrs = [...]string{
	StatusMsg:                  "StatusMsg",
	GetSubLedgerMsg:            "GetSubLedgerMsg",
	GetSnapshotBlockHeadersMsg: "GetSnapshotBlockHeadersMsg",
	GetSnapshotBlockBodiesMsg:  "GetSnapshotBlockBodiesMsg",
	GetSnapshotBlockMsg:        "GetSnapshotBlockMsg",
	GetAccountBlocksMsg:        "GetAccountBlocksMsg",
	SubLedgerMsg:               "SubLedgerMsg",
	SnapshotBlockHeadersMsg:    "SnapshotBlockHeadersMsg",
	SnapshotBlockBodiesMsg:     "SnapshotBlockBodiesMsg",
	SnapshotBlockMsg:           "SnapshotBlockMsg",
	AccountBlocksMsg:           "AccountBlocksMsg",
}

func (t MsgTyp) String() string {
	if t == ExceptionMsg {
		return "ExceptionMsg"
	}
	return msgStrs[t]
}

// @section Msg

type section struct {
	From    types.Hash
	Start   *big.Int
	To      types.Hash
	End     *big.Int
	Step    int
	Forward bool
}

type accountSection map[string]*section
