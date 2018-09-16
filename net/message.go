package net

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/protocols/protos"
	"github.com/vitelabs/go-vite/vitepb"
	"math/big"
)

const CmdSetName = "vite"
const CmdSetID uint64 = 2

// @section BlockID
type BlockID struct {
	Hash   types.Hash
	Height *big.Int
}

func (b *BlockID) Equal(hash types.Hash, height *big.Int) bool {
	equalHash := true
	equalHeight := true

	if hash != types.ZERO_HASH && b.Hash != types.ZERO_HASH {
		equalHash = hash == b.Hash
	}

	if height != nil && b.Height != nil {
		equalHeight = height.Cmp(b.Height) == 0
	}

	return equalHash && equalHeight
}

func (b *BlockID) proto() *protos.BlockID {
	return &protos.BlockID{
		Hash:   b.Hash[:],
		Height: b.Height.Bytes(),
	}
}

func (b *BlockID) deProto(pb *protos.BlockID) {
	copy(b.Hash[:], pb.Hash)
	b.Height = new(big.Int)
	b.Height.SetBytes(pb.Height)
}

func (b *BlockID) Serialize() ([]byte, error) {
	return proto.Marshal(b.proto())
}

func (b *BlockID) Deserialize(data []byte) error {
	pb := &protos.BlockID{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}
	b.deProto(pb)
	return nil
}

// @section Cmd
type CmdSet uint64
type Cmd uint64

const (
	HandshakeCode Cmd = iota
	StatusCode
	GetSubLedgerCode
	GetSnapshotBlockHeadersCode
	GetSnapshotBlockBodiesCode
	GetSnapshotBlocksCode
	GetSnapshotBlocksByHashCode
	GetAccountBlocksCode
	GetAccountBlocksByHashCode
	SubLedgerCode
	SnapshotBlockHeadersCode
	SnapshotBlockBodiesCode
	SnapshotBlocksCode
	AccountBlocksCode
	NewSnapshotBlockCode

	ExceptionCode = 127
)

var msgNames = [...]string{
	HandshakeCode:               "HandShakeMsg",
	StatusCode:                  "StatusMsg",
	GetSubLedgerCode:            "GetSubLedgerMsg",
	GetSnapshotBlockHeadersCode: "GetSnapshotBlockHeadersMsg",
	GetSnapshotBlockBodiesCode:  "GetSnapshotBlockBodiesMsg",
	GetSnapshotBlocksCode:       "GetSnapshotBlocksMsg",
	GetSnapshotBlocksByHashCode: "GetSnapshotBlocksByHashMsg",
	GetAccountBlocksCode:        "GetAccountBlocksMsg",
	GetAccountBlocksByHashCode:  "GetAccountBlocksByHashMsg",
	SubLedgerCode:               "SubLedgerMsg",
	SnapshotBlockHeadersCode:    "SnapshotBlockHeadersMsg",
	SnapshotBlockBodiesCode:     "SnapshotBlockBodiesMsg",
	SnapshotBlocksCode:          "SnapshotBlocksMsg",
	AccountBlocksCode:           "AccountBlocksMsg",
	NewSnapshotBlockCode:        "NewSnapshotBlockMsg",
}

func (t Cmd) String() string {
	if t == ExceptionCode {
		return "ExceptionMsg"
	}
	return msgNames[t]
}

// @section Msg Param

type Segment struct {
	From    *BlockID
	To      *BlockID
	Step    uint64
	Forward bool
}

func (s *Segment) proto() *protos.Segment {
	return &protos.Segment{
		From:    s.From.proto(),
		To:      s.To.proto(),
		Step:    s.Step,
		Forward: s.Forward,
	}
}

func (s *Segment) Serialize() ([]byte, error) {
	return proto.Marshal(s.proto())
}

func (s *Segment) Deserialize(data []byte) error {
	pb := &protos.Segment{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	s.Forward = pb.Forward
	s.Step = pb.Step

	s.From = new(BlockID)
	s.To = new(BlockID)
	s.From.deProto(pb.From)
	s.To.deProto(pb.To)

	return nil
}

type AccountSegment map[string]*Segment

func (as AccountSegment) Serialize() ([]byte, error) {
	// todo

	return nil, nil
}

func (as AccountSegment) Deserialize(data []byte) error {
	// todo
	return nil
}

// @message HandShake
type HandShakeMsg struct {
	Version      uint64
	NetID        uint64
	Height       *big.Int
	CurrentBlock types.Hash
	GenesisBlock types.Hash
}

func (st *HandShakeMsg) Serialize() ([]byte, error) {
	pb := &vitepb.StatusMsg{
		Version:      st.Version,
		Height:       st.Height.Bytes(),
		CurrentBlock: st.CurrentBlock[:],
		GenesisBlock: st.GenesisBlock[:],
	}

	return proto.Marshal(pb)
}

func (st *HandShakeMsg) Deserialize(data []byte) error {
	pb := new(vitepb.StatusMsg)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}
	st.Version = pb.Version

	bi := new(big.Int)
	st.Height = bi.SetBytes(pb.Height)
	copy(st.GenesisBlock[:], pb.GenesisBlock)
	copy(st.CurrentBlock[:], pb.CurrentBlock)

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
