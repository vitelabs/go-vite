package protocols

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
)

const Version uint32 = 2

type NetID uint32

const (
	MainNet NetID = iota + 1
	TestNet
)

func (i NetID) String() string {
	switch i {
	case MainNet:
		return "MainNet"
	case TestNet:
		return "TestNet"
	default:
		return "Unknown"
	}
}

// @section Msg
type Serializable interface {
	Serialize() ([]byte, error)
	Deserialize([]byte) error
}

type Msg struct {
	Code    MsgCode
	Payload []byte
}

// @section Transport
type MsgReader interface {
	ReadMsg() (Msg, error)
}

type MsgWriter interface {
	WriteMsg(Msg) error
}

type MsgReadWriter interface {
	MsgReader
	MsgWriter
}

type Transport interface {
	MsgReadWriter
	Close(ExceptionMsg)
}

// @section BlockID
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

// @section MsgCode
type MsgCode uint64

const (
	HandShakeCode MsgCode = iota
	StatusCode
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
	NewSnapshotBlockCode

	ExceptionCode = 127
)

var msgNames = [...]string{
	HandShakeCode:               "HandShakeMsg",
	StatusCode:                  "StatusMsg",
	GetSubLedgerCode:            "GetSubLedgerMsg",
	GetSnapshotBlockHeadersCode: "GetSnapshotBlockHeadersMsg",
	GetSnapshotBlockBodiesCode:  "GetSnapshotBlockBodiesMsg",
	GetSnapshotBlocksCode:       "GetSnapshotBlocksMsg",
	GetAccountBlocksCode:        "GetAccountBlocksMsg",
	SubLedgerCode:               "SubLedgerMsg",
	SnapshotBlockHeadersCode:    "SnapshotBlockHeadersMsg",
	SnapshotBlockBodiesCode:     "SnapshotBlockBodiesMsg",
	SnapshotBlocksCode:          "SnapshotBlocksMsg",
	AccountBlocksCode:           "AccountBlocksMsg",
	NewSnapshotBlockCode:        "NewSnapshotBlockMsg",
}

func (t MsgCode) String() string {
	if t == ExceptionCode {
		return "ExceptionMsg"
	}
	return msgNames[t]
}

// @section Msg Param

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

// @message HandShake

type HandShakeMsg struct {
	NetID
	Version      uint32
	Height       *big.Int
	CurrentBlock types.Hash
	GenesisBlock types.Hash
}

func (st *HandShakeMsg) Serialize() ([]byte, error) {
	stpb := &vitepb.StatusMsg{
		ProtocolVersion: st.Version,
		Height:          st.Height.Bytes(),
		CurrentBlock:    st.CurrentBlock[:],
		GenesisBlock:    st.GenesisBlock[:],
	}

	return proto.Marshal(stpb)
}

func (st *HandShakeMsg) Deserialize(data []byte) error {
	stpb := &vitepb.StatusMsg{}
	err := proto.Unmarshal(data, stpb)
	if err != nil {
		return err
	}
	st.Version = stpb.ProtocolVersion

	bi := new(big.Int)
	st.Height = bi.SetBytes(stpb.Height)
	copy(st.GenesisBlock[:], stpb.GenesisBlock)
	copy(st.CurrentBlock[:], stpb.CurrentBlock)

	return nil
}

type GetSubLedgerMsg = *Segment
type GetSnapshotBlockHeadersMsg = *Segment
type GetSnapshotBlockBodiesMsg []*types.Hash
type GetSnapshotBlocksMsg []*types.Hash
type GetAccountBlocksMsg = AccountSegment

type SubLedgerMsg struct {
	Files []string
}

// todo type SnapshotBlockHeadersMsg
// todo type SnapshotBlockBodiesMsg
type SnapshotBlocksMsg = []*ledger.SnapshotBlock
type AccountBlocksMsg map[string]*ledger.AccountBlock

type NewBlockMsg *ledger.SnapshotBlock

// @message ExceptionMsg
type ExceptionMsg uint

const (
	Fork                ExceptionMsg = iota // you have forked
	Missing                                 // I don`t have the resource you requested
	Canceled                                // the request have been canceled
	Unsolicited                             // the request must have pre-checked
	Blocked                                 // you have been blocked
	RepetitiveHandshake                     // handshake should happen only once, as the first msg
	Connected                               // you have been connected with me
	DifferentNet
	UnMatchedMsgVersion
	UnIdenticalGenesis
)

var execption = [...]string{
	Fork:                "you have forked",
	Missing:             "I don`t have the resource you requested",
	Canceled:            "the request have been canceled",
	Unsolicited:         "your request must have pre-checked",
	Blocked:             "you have been blocked",
	RepetitiveHandshake: "handshake should happen only once, as the first msg",
	Connected:           "you have connected to me",
	DifferentNet:        "we are at different network",
	UnMatchedMsgVersion: "UnMatchedMsgVersion",
	UnIdenticalGenesis:  "UnIdenticalGenesis",
}

func (exp ExceptionMsg) String() string {
	return execption[exp]
}

func (exp *ExceptionMsg) Serialize() ([]byte, error) {
	buf := make([]byte, 10)
	n := binary.PutUvarint(buf, uint64(*exp))
	return buf[:n], nil
}
func (exp *ExceptionMsg) Deserialize(buf []byte) error {
	i, _ := binary.Varint(buf)
	*exp = ExceptionMsg(i)
	return nil
}
