package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
)

// @section Msg
type Serializable interface {
	Serialize() ([]byte, error)
}

type Msg struct {
	Code uint64
	Payload Serializable
}

// @message current blockchain status.
type StatusMsg struct {
	NetworkId uint64
	ProtocolVersion uint32
	Height big.Int
	CurrentBlock types.Hash
	GenesisBlock types.Hash
}

func (stmsg *StatusMsg) Serialize() ([]byte, error) {
	// todo
}

// @message get multiple snapshot blocks.
type GetSnapshotBlocksMsg struct {
	Origin 	types.Hash
	Count 	uint64
	Forward bool
}

func (g *GetSnapshotBlocksMsg) Serialize() ([]byte, error) {
	// todo
}

// @message send multiple snapshot block data.
type SnapshotBlocksMsg []*ledger.SnapshotBlock

func (g *SnapshotBlocksMsg) Serialize() ([]byte, error) {
	// todo
}

// @message get multiple account blocks.
type GetAccountBlocksMsg struct {
	Origin 	types.Hash
	Count 	uint64
	Forward bool
}

func (ab *GetAccountBlocksMsg) Serialize() ([]byte, error) {
	// todo
}

// @message send multiple account block data.
type AccountBlocksMsg []*ledger.AccountBlock

func (abmsg *AccountBlocksMsg) Serialize() ([]byte, error) {
	// todo.
}

// @section
type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	// WriteMsg sends a message. It will block until the message's
	// Payload has been consumed by the other end.
	//
	// Note that messages can be sent only once because their
	// payload reader is drained.
	WriteMsg(Msg) error
}

// MsgReadWriter provides reading and writing of encoded messages.
// Implementations should ensure that ReadMsg and WriteMsg can be
// called simultaneously from multiple goroutines.
type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

// @section
type NetSender interface {
	SendMsg(MsgWriter, Msg) error
}
