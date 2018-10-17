package message

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
	"strconv"
)

var errDeserialize = errors.New("deserialize error")

// @section GetSnapshotBlocks

type GetSnapshotBlocks struct {
	From    ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetSnapshotBlocks) String() string {
	var from string
	if b.From.Hash == types.ZERO_HASH {
		from = strconv.FormatUint(b.From.Height, 10)
	} else {
		from = b.From.Hash.String()
	}

	return "GetSnapshotBlocks<" + from + "/" + strconv.FormatUint(b.Count, 10) + "/" + strconv.FormatBool(b.Forward) + ">"
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

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	if pb.From == nil {
		return errDeserialize
	}

	b.From = ledger.HashHeight{
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

func (b *SnapshotBlocks) String() string {
	return "SnapshotBlocks<" + strconv.FormatInt(int64(len(b.Blocks)), 10) + ">"
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
	ABlocks []*ledger.AccountBlock
}

func (s *SubLedger) String() string {
	return "SubLedger<" + strconv.FormatInt(int64(len(s.SBlocks)), 10) + "/" + strconv.FormatInt(int64(len(s.ABlocks)), 10) + ">"
}

func (s *SubLedger) Serialize() ([]byte, error) {
	pb := new(vitepb.SubLedger)

	pb.SBlocks = make([]*vitepb.SnapshotBlock, len(s.SBlocks))
	for i, b := range s.SBlocks {
		pb.SBlocks[i] = b.Proto()
	}

	pb.ABlocks = make([]*vitepb.AccountBlock, len(s.ABlocks))
	for i, b := range s.ABlocks {
		pb.ABlocks[i] = b.Proto()
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

	s.ABlocks = make([]*ledger.AccountBlock, len(pb.ABlocks))
	for i, abp := range pb.ABlocks {
		block := new(ledger.AccountBlock)
		block.DeProto(abp)
		s.ABlocks[i] = block
	}

	return nil
}

// @section GetAccountBlocks

type GetAccountBlocks struct {
	Address types.Address
	From    ledger.HashHeight
	Count   uint64
	Forward bool
}

func (b *GetAccountBlocks) String() string {
	var from string
	if b.From.Hash == types.ZERO_HASH {
		from = strconv.FormatUint(b.From.Height, 10)
	} else {
		from = b.From.Hash.String()
	}

	return "GetAccountBlocks<" + from + "/" + strconv.FormatUint(b.Count, 10) + "/" + strconv.FormatBool(b.Forward) + ">"
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

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	if pb.From == nil {
		return errDeserialize
	}

	b.From = ledger.HashHeight{
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
	Blocks []*ledger.AccountBlock
}

func (a *AccountBlocks) String() string {
	return "AccountBlocks<" + strconv.FormatInt(int64(len(a.Blocks)), 10) + ">"
}

func (a *AccountBlocks) Serialize() ([]byte, error) {
	pb := new(vitepb.AccountBlocks)

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

	return nil
}
