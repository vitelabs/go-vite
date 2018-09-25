package net

import (
	"encoding/binary"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
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
	return b.Hash == hash && b.Height == height
}

func (b *BlockID) Ceil() uint64 {
	return b.Height
}

func (b *BlockID) proto() *vitepb.BlockID {
	return &vitepb.BlockID{
		Hash:   b.Hash[:],
		Height: b.Height,
	}
}

func (b *BlockID) deProto(pb *vitepb.BlockID) {
	copy(b.Hash[:], pb.Hash)
	b.Height = b.Height
}

func (b *BlockID) Serialize() ([]byte, error) {
	return proto.Marshal(b.proto())
}

func (b *BlockID) Deserialize(data []byte) error {
	pb := new(vitepb.BlockID)
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
	ForkCode // tell peer it has forked
	GetSubLedgerCode
	GetSnapshotBlockHeadersCode
	GetSnapshotBlockBodiesCode
	GetSnapshotBlocksCode
	GetSnapshotBlocksByHashCode
	GetAccountBlocksCode       // query single AccountChain
	GetMultiAccountBlocksCode  // query multi AccountChain
	GetAccountBlocksByHashCode // query accountBlocks by hashList
	GetFileCode
	SubLedgerCode
	FileListCode
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
	ForkCode:                    "ForkMsg",
	GetSubLedgerCode:            "GetSubLedgerMsg",
	GetSnapshotBlockHeadersCode: "GetSnapshotBlockHeadersMsg",
	GetSnapshotBlockBodiesCode:  "GetSnapshotBlockBodiesMsg",
	GetSnapshotBlocksCode:       "GetSnapshotBlocksMsg",
	GetSnapshotBlocksByHashCode: "GetSnapshotBlocksByHashMsg",
	GetAccountBlocksCode:        "GetAccountBlocksMsg",
	GetMultiAccountBlocksCode:   "GetMultiAccountBlocksMsg",
	GetAccountBlocksByHashCode:  "GetAccountBlocksByHashMsg",
	GetFileCode:                 "GetFileMsg",
	SubLedgerCode:               "SubLedgerMsg",
	FileListCode:                "FileListMsg",
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

func (t cmd) canReceive(t2 cmd) bool {
	switch t {
	case HandshakeCode:
		return t2 == HandshakeCode
	case GetSubLedgerCode:
		return t2 == FileListCode || t2 == SubLedgerCode || t2 == ForkCode
	case GetSnapshotBlockHeadersCode:
		return t2 == SnapshotBlockHeadersCode || t2 == ForkCode
	case GetSnapshotBlockBodiesCode:
		return t2 == SnapshotBlockBodiesCode || t2 == ForkCode
	case GetSnapshotBlocksCode:
		return t2 == SnapshotBlocksCode || t2 == ForkCode
	case GetSnapshotBlocksByHashCode:
		return t2 == SnapshotBlocksCode || t2 == ForkCode
	case GetAccountBlocksCode:
		return t2 == AccountBlocksCode || t2 == ForkCode
	case GetMultiAccountBlocksCode:
		return t2 == AccountBlocksCode || t2 == ForkCode
	case GetAccountBlocksByHashCode:
		return t2 == AccountBlocksCode || t2 == ForkCode
	default:
		return false
	}
}

// @section use to query subLedger
type Segment struct {
	Origin  types.Hash // From.Height must little than To.height
	Start   uint64
	Count   uint64
	Forward bool
	Step    uint64
}

func (s *Segment) proto() *vitepb.Segment {
	return &vitepb.Segment{}
}

func (b *Segment) deProto(pb *vitepb.Segment) {

}

func (s *Segment) Serialize() ([]byte, error) {
	return proto.Marshal(s.proto())
}

func (s *Segment) Deserialize(data []byte) error {
	pb := &vitepb.Segment{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	s.deProto(pb)

	return nil
}

func (s *Segment) Equal(v interface{}) bool {
	s2, ok := v.(*Segment)
	if !ok {
		return false
	}

}

func (s *Segment) Ceil() uint64 {
	if s.Forward {
		return s.Start + s.Count
	}

	return s.Start
}

func (s *Segment) floor() uint64 {
	if s.Forward {
		return s.Start
	}
	return s.Start - s.Count
}

func (s *Segment) Handle(res *response) error {

}

// split long chain segment to small chunks
const batch = 3600

func (s *Segment) split() (segs []*Segment) {
	if s.Count <= batch {
		segs = append(segs, s)
		return
	}

	start := s.floor()
	end := s.Ceil()

	for from, step := start, start; step <= end; from = step + 1 {
		step = ((from / batch) + 1) * batch
		if step >= end {
			step = end
		}

		segs = append(segs, &Segment{
			Start:   from,
			Count:   step - from,
			Forward: true,
		})
	}

	return
}

// @section Forked
type Forked struct {
	StepID []*BlockID
}

func (f *Forked) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *Forked) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section single address
type AccountSegment struct {
	Address types.Address
	Segment *Segment
}

func (as *AccountSegment) Serialize() ([]byte, error) {
	panic("implement me")
}

func (as *AccountSegment) Deserialize(buf []byte) error {
	panic("implement me")
}

func (as *AccountSegment) Equal(v interface{}) bool {
	panic("implement me")
}

func (as *AccountSegment) Ceil() uint64 {
	panic("implement me")
}

// @section use to query accountblocks
type MultiAccountSegment map[types.Address]*Segment

func (as MultiAccountSegment) Serialize() ([]byte, error) {
	pb := new(vitepb.MultiAccountSegment)
	pb.Segments = make([]*vitepb.AccountSegment, len(as))

	i := 0
	for address, seg := range as {
		pb.Segments[i] = &vitepb.AccountSegment{
			Address: address[:],
			Segment: seg.proto(),
		}
		i++
	}

	return proto.Marshal(pb)
}

func (as MultiAccountSegment) Deserialize(data []byte) error {
	pb := new(vitepb.MultiAccountSegment)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	for _, pseg := range pb.Segments {
		var address types.Address
		copy(address[:], pseg.Address)

		seg := new(Segment)
		seg.deProto(pseg.Segment)

		as[address] = seg
	}

	return nil
}

func (as MultiAccountSegment) Equal(v interface{}) bool {
	as2, ok := v.(MultiAccountSegment)
	if !ok {
		return false
	}

	for address, seg := range as {
		if seg2, ok := as2[address]; ok && seg.Equal(seg2) {
			continue
		}

		return false
	}
	return true
}

func (as MultiAccountSegment) Ceil(v interface{}) uint64 {
	return 0
}

// @section subLedger response
type file struct {
	Name string
	Start uint64
	End uint64
}

type FileList struct {
	Files []*file
	Start uint64 // start and end means need query blocks from chainDB
	End   uint64 // because files don`t contain the latest snapshotblocks
	Nonce uint64 // use only once
}

func (f *FileList) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *FileList) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section
type FileRequestMsg struct {
	File  string
	Nonce uint64
}

func (f *FileRequestMsg) Serialize() ([]byte, error) {
	panic("implement me")
}

func (f *FileRequestMsg) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section subLedger query from chainDB
type SubLedger struct {
	SBlocks []*ledger.SnapshotBlock
	ABlocks []*ledger.AccountBlock
}

func (s *SubLedger) Serialize() ([]byte, error) {
	pb := new(vitepb.SubLedger)
	pb.SBlocks = make([]*vitepb.SnapshotBlockNet, len(s.SBlocks))
	pb.ABlocks = make([]*vitepb.AccountBlockNet, len(s.ABlocks))

	//var spb *vitepb.SnapshotBlockNet
	//for i, block := range s.SBlocks {
	//	spb = new(vitepb.SnapshotBlockNet)
	//	spb.
	//		pb.SBlocks[i] = pb
	//}

	return proto.Marshal(pb)
}

func (s *SubLedger) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section AccountBlocks
type AccountBlocks struct {
	Blocks []*ledger.AccountBlock
}

func (as *AccountBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (as *AccountBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

// @section SnapshotBlocks
type SnapshotBlocks struct {
	Blocks []*ledger.SnapshotBlock
}

func (s *SnapshotBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (s *SnapshotBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

// @message HandShake
type HandShakeMsg struct {
	CmdSet      uint64
	NetID        uint64
	Height       uint64
	Port         uint16
	CurrentBlock types.Hash
	GenesisBlock types.Hash
}

func (st *HandShakeMsg) Serialize() ([]byte, error) {
	pb := &vitepb.StatusMsg{
		Version:      st.CmdSet,
		Height:       st.Height,
		CurrentBlock: st.CurrentBlock[:],
		GenesisBlock: st.GenesisBlock[:],
		Port:         uint32(st.Port),
	}

	return proto.Marshal(pb)
}

func (st *HandShakeMsg) Deserialize(data []byte) error {
	pb := new(vitepb.StatusMsg)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	st.CmdSet = pb.Version
	st.Height = pb.Height
	st.Port = uint16(pb.Port)
	copy(st.GenesisBlock[:], pb.GenesisBlock)
	copy(st.CurrentBlock[:], pb.CurrentBlock)

	return nil
}

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
