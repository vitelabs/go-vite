package message

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
)

// @section GetSnapshotBlocks

type GetSnapshotBlocks struct {
	From    *ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetSnapshotBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.GetSnapshotBlocks)
	pb.From = &vitepb.BlockID{
		Hash:   b.From.Hash[:],
		Height: b.From.Height,
	}
	pb.Count = b.Count
	pb.Forward = b.Forward

	return proto.Marshal(pb)
}

func (b *GetSnapshotBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.GetSnapshotBlocks)
	pb.From = new(vitepb.BlockID)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	b.From = &ledger.HashHeight{
		Height: pb.From.Height,
	}
	copy(b.From.Hash[:], pb.From.Hash)

	b.Count = pb.Count
	b.Forward = pb.Forward

	return nil
}

// @section SnapshotBlocks

type SnapshotBlocks struct {
	Blocks []*ledger.SnapshotBlock
}

func (b *SnapshotBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.SnapshotBlocks)

	pb.Blocks = make([]*vitepb.SnapshotBlock, len(b.Blocks))

	for i, block := range b.Blocks {
		pb.Blocks[i] = block.Proto()
	}

	return proto.Marshal(pb)
}

func (b *SnapshotBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.SnapshotBlocks)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	b.Blocks = make([]*ledger.SnapshotBlock, len(pb.Blocks))
	for i, bp := range pb.Blocks {
		block := new(ledger.SnapshotBlock)
		block.DeProto(bp)
		b.Blocks[i] = block
	}

	return nil
}

// @section SubLedger

type SubLedger struct {
	SBlocks []*ledger.SnapshotBlock
	ABlocks map[types.Address][]*ledger.AccountBlock
}

func (s *SubLedger) Serialize() ([]byte, error) {
	pb := new(vitepb.SubLedger)
	pb.SBlocks = make([]*vitepb.SnapshotBlock, len(s.SBlocks))

	for i, b := range s.SBlocks {
		pb.SBlocks[i] = b.Proto()
	}
	for _, bs := range s.ABlocks {
		bps := make([]*vitepb.AccountBlock, len(bs))
		for i, b := range bs {
			bps[i] = b.Proto()
		}

		pb.ABlocks = append(pb.ABlocks, bps...)
	}

	return proto.Marshal(pb)
}

func (s *SubLedger) Deserialize(buf []byte) error {
	pb := new(vitepb.SubLedger)
	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	s.SBlocks = make([]*ledger.SnapshotBlock, len(pb.SBlocks))
	for i, p := range pb.SBlocks {
		block := new(ledger.SnapshotBlock)
		block.DeProto(p)
		s.SBlocks[i] = block
	}

	s.ABlocks = make(map[types.Address][]*ledger.AccountBlock)
	for _, abp := range pb.ABlocks {
		ab := new(ledger.AccountBlock)
		ab.DeProto(abp)
		s.ABlocks[ab.AccountAddress] = append(s.ABlocks[ab.AccountAddress], ab)
	}

	return nil
}

// @section GetAccountBlocks

type GetAccountBlocks struct {
	Address types.Address // maybe nil
	From    *ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetAccountBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.GetAccountBlocks)
	pb.Address = b.Address[:]
	pb.From = &vitepb.BlockID{
		Hash:   b.From.Hash[:],
		Height: b.From.Height,
	}
	pb.Count = b.Count
	pb.Forward = b.Forward

	return proto.Marshal(pb)
}

func (b *GetAccountBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.GetAccountBlocks)
	pb.From = new(vitepb.BlockID)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	b.From = &ledger.HashHeight{
		Height: pb.From.Height,
	}
	copy(b.From.Hash[:], pb.From.Hash)

	b.Count = pb.Count
	b.Forward = pb.Forward
	copy(b.Address[:], pb.Address)

	return nil
}

// @section AccountBlocks

type AccountBlocks struct {
	Address types.Address
	Blocks  []*ledger.AccountBlock
}

func (a *AccountBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.AccountBlocks)

	pb.Address = a.Address[:]
	pb.Blocks = make([]*vitepb.AccountBlock, len(a.Blocks))

	for i, block := range a.Blocks {
		pb.Blocks[i] = block.Proto()
	}

	return proto.Marshal(pb)
}

func (a *AccountBlocks) Deserialize(buf []byte) error {
	pb := new(vitepb.AccountBlocks)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	a.Blocks = make([]*ledger.AccountBlock, len(pb.Blocks))
	for i, bp := range pb.Blocks {
		block := new(ledger.AccountBlock)
		block.DeProto(bp)
		a.Blocks[i] = block
	}

	copy(a.Address[:], pb.Address)

	return nil
}
