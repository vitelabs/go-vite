package message

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

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

type GetSnapshotBlocks struct {
	From *BlockID
	Count uint64
	Forward bool
}

func (b *GetSnapshotBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (b *GetSnapshotBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

type SubLedger struct {
	SBlocks []*ledger.SnapshotBlock
	ABlocks []*ledger.AccountBlock
}

func (s *SubLedger) Serialize() ([]byte, error) {
	panic("implement me")
}

func (s *SubLedger) Deserialize(buf []byte) error {
	panic("implement me")
}

type SnapshotBlocks struct {
	Blocks []*ledger.SnapshotBlock
}

func (b *SnapshotBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (b *SnapshotBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

type GetAccountBlocks struct {
	Address types.Address
	From *BlockID
	Count uint64
	Forward bool
}

func (b *GetAccountBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (b *GetAccountBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}

type AccountBlocks struct {
	Address types.Address
	Blocks []*ledger.AccountBlock
}

func (a *AccountBlocks) Serialize() ([]byte, error) {
	panic("implement me")
}

func (a *AccountBlocks) Deserialize(buf []byte) error {
	panic("implement me")
}
