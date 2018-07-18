package protocols

import (
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"math/big"
	"github.com/vitelabs/go-vite/protocols/protos"
	"github.com/golang/protobuf/proto"
	"sync"
	"github.com/vitelabs/go-vite/p2p"
	"github.com/vitelabs/go-vite/vitepb/proto"
	"log"
)

// @section Peer for protocol handle, not p2p Peer.
type Peer struct {
	*p2p.Peer
	ID 		string
	Head 	types.Hash
	Height	*big.Int
	Version int
	RW 		MsgReadWriter
	Lock 	sync.RWMutex
}

// @section Msg
type Serializable interface {
	NetSerialize() ([]byte, error)
	NetDeserialize([]byte) error
}

type Msg struct {
	Code uint64
	Payload Serializable
}

const vite1 = 1
var protocolVersions = []uint{vite1}
var protocolBand = []uint{10}

// @section msg code
const (
	StatusMsgCode uint64 = 17
	GetSnapshotBlocksMsgCode = 18
	SnapshotBlocksMsgCode = 19
	GetAccountBlocksMsgCode = 20
	AccountBlocksMsgCode = 21
)

// @message current blockchain status.
type StatusMsg struct {
	ProtocolVersion uint32
	Height *big.Int
	CurrentBlock types.Hash
	GenesisBlock types.Hash
}

func (st *StatusMsg) NetSerialize() ([]byte, error) {
	stpb := &protos.StatusMsg{
		ProtocolVersion: st.ProtocolVersion,
		Height: st.Height.Bytes(),
		CurrentBlock: st.CurrentBlock[:],
		GenesisBlock: st.GenesisBlock[:],
	}

	return proto.Marshal(stpb)
}

func (st *StatusMsg) NetDeserialize(data []byte) error {
	stpb := &protos.StatusMsg{}
	err := proto.Unmarshal(data, stpb)
	if err != nil {
		return err
	}
	st.ProtocolVersion = stpb.ProtocolVersion

	var bi *big.Int
	st.Height = bi.SetBytes(stpb.Height)
	copy(st.GenesisBlock[:], stpb.GenesisBlock)
	copy(st.CurrentBlock[:], stpb.CurrentBlock)

	return nil
}

// @message get multiple snapshot blocks.
type GetSnapshotBlocksMsg struct {
	Origin 	types.Hash
	Count 	uint64
	Forward bool
}

func (gs *GetSnapshotBlocksMsg) NetSerialize() ([]byte, error) {
	gspb := &protos.GetSnapshotBlocksMsg{
		Origin: gs.Origin[:],
		Count: gs.Count,
		Forward: gs.Forward,
	}

	return proto.Marshal(gspb)
}

func (gs *GetSnapshotBlocksMsg) NetDeserialize(data []byte) error {
	gspb := &protos.GetSnapshotBlocksMsg{}
	err := proto.Unmarshal(data, gspb)
	if err != nil {
		return err
	}
	copy(gs.Origin[:], gspb.Origin)
	gs.Count = gspb.Count
	gs.Forward = gspb.Forward
	return nil
}

// @message send multiple snapshot block data.
type SnapshotBlocksMsg []*ledger.SnapshotBlock

func (s *SnapshotBlocksMsg) NetSerialize() ([]byte, error) {
	bs := make([]*vitepb.SnapshotBlock, len(*s))

	for i, b := range *s {
		bs[i] = b.GetNetPB()
	}
	spb := &protos.SnapshotBlocksMsg{
		Blocks: bs,
	}

	return proto.Marshal(spb)
}

func (s *SnapshotBlocksMsg) NetDeserialize(data []byte) error {
	spb := &protos.SnapshotBlocksMsg{}
	err := proto.Unmarshal(data, spb)
	if err != nil {
		return err
	}

	for _, pb := range spb.Blocks {
		b := new(ledger.SnapshotBlock)
		if err := b.SetByNetPB(pb); err == nil {
			*s = append(*s, b)
		} else {
			log.Printf("AccountBlock.SetByNetPB error: %v\n", err)
		}
	}

	return nil
}

// @message get multiple account blocks.
type GetAccountBlocksMsg struct {
	Origin 	types.Hash
	Count 	uint64
	Forward bool
}

func (ga *GetAccountBlocksMsg) NetSerialize() ([]byte, error) {
	gapb := &protos.GetAccountBlocksMsg{
		Origin: ga.Origin[:],
		Count: ga.Count,
		Forward: ga.Forward,
	}

	return proto.Marshal(gapb)
}

func (ga *GetAccountBlocksMsg) NetDeserialize(data []byte) error {
	gapb := &protos.GetAccountBlocksMsg{}
	err := proto.Unmarshal(data, gapb)
	if err != nil {
		return err
	}
	copy(ga.Origin[:], gapb.Origin)
	ga.Count = gapb.Count
	ga.Forward = gapb.Forward
	return nil
}

// @message send multiple account block data.
type AccountBlocksMsg []*ledger.AccountBlock

func (a *AccountBlocksMsg) NetSerialize() ([]byte, error) {
	bs := make([]*vitepb.AccountBlockNet, len(*a))

	for i, b := range *a {
		bs[i] = b.GetNetPB()
	}

	apb := &protos.AccountBlocksMsg{
		Blocks: bs,
	}

	return proto.Marshal(apb)
}

func (a *AccountBlocksMsg) NetDeserialize(data []byte) error {
	apb := &protos.AccountBlocksMsg{}
	err := proto.Unmarshal(data, apb)
	if err != nil {
		return err
	}

	for _, b := range apb.Blocks {
		ab := new(ledger.AccountBlock)
		if err := ab.SetByNetPB(b); err == nil {
			*a = append(*a, ab)
		} else {
			log.Printf("AccountBlock.SetByNetPB error: %v\n", err)
		}
	}

	return nil
}

// @section
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
