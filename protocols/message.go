package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

type BlockID struct {
	Hash   types.Hash
	Height *big.Int
}

var ZERO_HASH = types.Hash{}

func (this *BlockID) Equal(hash types.Hash, height *big.Int) bool {
	equalHash := true
	equalHeight := true
	if hash != ZERO_HASH && this.Hash != ZERO_HASH {
		equalHash = hash == this.Hash
	}
	if height != nil && this.Height != nil {
		equalHeight = height.Cmp(this.Height) == 0
	}

	return equalHash && equalHeight
}

func (this *BlockID) Serialize() ([]byte, error) {

}

// @section MsgTyp
type MsgTyp int

const (
	StatusCode MsgTyp = iota
	GetSubLedgerCode
	GetSnapshotBlockHeadersCode
	GetSnapshotBlockBodiesCode
	GetSnapshotBlocksCode
	GetAccountBlocksCode
	SubLedgerCode
	SnapshotBlockHeadersCode
	SnapshotBlockBodiesCode
	SnapshotBlocksCode
	AccountBlocksCode

	ExceptionCode = 127
)

var msgStrs = [...]string{
	StatusCode:                  "StatusMsg",
	GetSubLedgerCode:            "GetSubLedgerMsg",
	GetSnapshotBlockHeadersCode: "GetSnapshotBlockHeadersMsg",
	GetSnapshotBlockBodiesCode:  "GetSnapshotBlockBodiesMsg",
	GetSnapshotBlocksCode:       "GetSnapshotBlockMsg",
	GetAccountBlocksCode:        "GetAccountBlocksMsg",
	SubLedgerCode:               "SubLedgerMsg",
	SnapshotBlockHeadersCode:    "SnapshotBlockHeadersMsg",
	SnapshotBlockBodiesCode:     "SnapshotBlockBodiesMsg",
	SnapshotBlocksCode:          "SnapshotBlockMsg",
	AccountBlocksCode:           "AccountBlocksMsg",
}

func (t MsgTyp) String() string {
	if t == ExceptionCode {
		return "ExceptionMsg"
	}
	return msgStrs[t]
}

// @section Msg

type Segment struct {
	From    *BlockID
	To      *BlockID
	Step    int
	Forward bool
}

func (this *Segment) Serialize() ([]byte, error) {

}

type AccountSegment map[string]*Segment

func (this *AccountSegment) Serialize() ([]byte, error) {

}

type GetSnapshotBlockHeadersMsg = *Segment
type GetSnapshotBlockBodiesMsg []*types.Hash
type GetSnapshotBlocksMsg []*types.Hash
type SubLedgerMsg = *Segment
type GetAccountBlocksMsg = AccountSegment

type SnapshotBlocksMsg = []*ledger.SnapshotBlock
type AccountBlocksMsg map[string]*ledger.AccountBlock

// @message ExceptionMsg
type ExceptionMsgCode int

const (
	Fork                ExceptionMsgCode = iota // you have forked
	Missing                                     // I don`t have the resource you requested
	Canceled                                    // the request have been canceled
	Unsolicited                                 // the request must have pre-checked
	Blocked                                     // you have been blocked
	RepetitiveHandshake                         // handshake should happen only once, as the first msg
	Connected                                   // you have been connected with me
	DifferentNet
	UnMatchedMsgVersion
	UnIdenticalGenesis
)

type ExceptionMsg struct {
	Code ExceptionMsgCode
}
