package net

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vite/net/protos"
	"github.com/vitelabs/go-vite/vitepb"
	"time"
)

var errHandshakeTwice = errors.New("handshake should send only once")
var errMsgTimeout = errors.New("message response timeout")

var subledgerTimeout = 10 * time.Second
var accountBlocksTimeout = 30 * time.Second
var snapshotBlocksTimeout = time.Minute

// @section BlockID
type BlockID struct {
	Hash   types.Hash
	Height uint64
}

func (b *BlockID) Equal(hash types.Hash, height uint64) bool {
	equalHash := true
	equalHeight := true

	if hash != types.ZERO_HASH && b.Hash != types.ZERO_HASH {
		equalHash = hash == b.Hash
	}

	if height != 0 && b.Height != 0 {
		equalHeight = height == b.Height
	}

	return equalHash && equalHeight
}

func (b *BlockID) Ceil() uint64 {
	return b.Height
}

func (b *BlockID) proto() *protos.BlockID {
	return &protos.BlockID{
		Hash:   b.Hash[:],
		Height: b.Height,
	}
}

func (b *BlockID) deProto(pb *protos.BlockID) {
	copy(b.Hash[:], pb.Hash)
	b.Height = b.Height
}

func (b *BlockID) Serialize() ([]byte, error) {
	return proto.Marshal(b.proto())
}

func (b *BlockID) Deserialize(data []byte) error {
	pb := new(protos.BlockID)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}
	b.deProto(pb)
	return nil
}

// @section Cmd
const CmdSetName = "vite"

var cmdSets = []uint64{2}

type cmd uint64

const (
	HandshakeCode cmd = iota
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

func (t cmd) String() string {
	if t == ExceptionCode {
		return "ExceptionMsg"
	}
	return msgNames[t]
}

// @section Msg Param

type Segment struct {
	From    *BlockID
	To      *BlockID
	Count   uint64
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

func (b *Segment) deProto(pb *protos.Segment) {
	b.From = new(BlockID)
	b.From.deProto(pb.From)

	b.To = new(BlockID)
	b.To.deProto(pb.To)

	b.Step = pb.Step
	b.Forward = pb.Forward
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

func (s *Segment) Equal(v interface{}) bool {
	s2, ok := v.(*Segment)
	if !ok {
		return false
	}
	fromeq := s.From.Equal(s2.From.Hash, s2.From.Height)
	toeq := s.To.Equal(s2.To.Hash, s2.To.Height)
	stepeq := s.Step == s2.Step
	feq := s.Forward == s2.Forward

	return fromeq && toeq && stepeq && feq
}

func (s *Segment) Ceil() uint64 {
	return s.From.Height
}

// @section Hashes
type Hashes []types.Hash

func (hs Hashes) Serialize() ([]byte, error) {
	panic("implement me")
}

func (hs Hashes) Deserialize(buf []byte) error {
	panic("implement me")
}

func (hs Hashes) Equal(v interface{}) bool {
	h2, ok := v.(Hashes)
	if !ok {
		return false
	}

	if len(hs) != len(h2) {
		return false
	}

	for _, hash := range hs {
		if !h2.Contain(hash) {
			return false
		}
	}

	return true
}

func (hs Hashes) Ceil() uint64 {
	return 0
}

func (hs Hashes) Contain(hash types.Hash) bool {
	for _, h := range hs {
		if h == hash {
			return true
		}
	}

	return false
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

func (as AccountSegment) Equal(v interface{}) bool {
	// todo
	return true
}

func (as AccountSegment) Ceil(v interface{}) uint64 {
	return 0
}

type subLedgerMsg struct {
	snapshotblocks []*ledger.SnapshotBlock
	accountblocks  []*ledger.AccountBlock
}

func (s *subLedgerMsg) Serialize() ([]byte, error) {
	panic("implement me")
}

func (s *subLedgerMsg) Deserialize(buf []byte) error {
	panic("implement me")
}

// @message HandShake
type HandShakeMsg struct {
	Version      uint64
	NetID        uint64
	Height       uint64
	CurrentBlock types.Hash
	GenesisBlock types.Hash
}

func (st *HandShakeMsg) Serialize() ([]byte, error) {
	pb := &vitepb.StatusMsg{
		Version:      st.Version,
		Height:       st.Height,
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

	st.Height = pb.Height
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
type AccountBlocksMsg map[string][]*ledger.AccountBlock

type NewBlockMsg *ledger.SnapshotBlock

// @message ExceptionMsg
type ExceptionMsg uint64

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

var exception = [...]string{
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
	return exception[exp]
}

func (exp ExceptionMsg) Error() string {
	return exception[exp]
}

func (exp ExceptionMsg) Serialize() ([]byte, error) {
	buf := make([]byte, 10)
	n := binary.PutUvarint(buf, uint64(exp))
	return buf[:n], nil
}
func (exp ExceptionMsg) Deserialize(buf []byte) error {
	panic("use deserializeException instead")
}

func deserializeException(buf []byte) (e ExceptionMsg, err error) {
	u64, n := binary.Varint(buf)
	if n != len(buf) {
		err = errors.New("use incomplete data")
		return
	}

	return ExceptionMsg(u64), nil
}
