package message

import (
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/ledger"
	"github.com/vitelabs/go-vite/vitepb"
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
	pb.From = &vitepb.HashHeight{
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

// SnapshotBlocks is batch of snapshot blocks
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
		err = block.DeProto(bp)
		if err != nil {
			return err
		}
		b.Blocks[i] = block
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
	pb.From = &vitepb.HashHeight{
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

// AccountBlocks is batch of account blocks
type AccountBlocks struct {
	Blocks []*ledger.AccountBlock
	TTL    int32
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
		err = block.DeProto(bp)
		if err != nil {
			return err
		}
		a.Blocks[i] = block
	}

	return nil
}

// NewSnapshotBlock is use to propagate block, stop propagate when TTL is decrease to zero
type NewSnapshotBlock struct {
	Block *ledger.SnapshotBlock
	TTL   int32
}

func (b *NewSnapshotBlock) Serialize() ([]byte, error) {
	pb := new(vitepb.NewSnapshotBlock)

	pb.Block = b.Block.Proto()

	pb.TTL = b.TTL

	return proto.Marshal(pb)
}

func (b *NewSnapshotBlock) Deserialize(buf []byte) error {
	pb := new(vitepb.NewSnapshotBlock)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	b.Block = new(ledger.SnapshotBlock)
	err = b.Block.DeProto(pb.Block)
	if err != nil {
		return err
	}

	b.TTL = pb.TTL

	return nil
}

// NewAccountBlock is use to propagate block, stop propagate when TTL is decrease to zero
type NewAccountBlock struct {
	Block *ledger.AccountBlock
	TTL   int32
}

func (b *NewAccountBlock) Serialize() ([]byte, error) {
	pb := new(vitepb.NewAccountBlock)

	pb.Block = b.Block.Proto()

	pb.TTL = b.TTL

	return proto.Marshal(pb)
}

func (b *NewAccountBlock) Deserialize(buf []byte) error {
	pb := new(vitepb.NewAccountBlock)

	err := proto.Unmarshal(buf, pb)
	if err != nil {
		return err
	}

	b.Block = new(ledger.AccountBlock)
	err = b.Block.DeProto(pb.Block)
	if err != nil {
		return err
	}

	b.TTL = pb.TTL

	return nil
}
