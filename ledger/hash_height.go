package ledger

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb"
)

type HashHeight struct {
	Height uint64
	Hash   types.Hash
}

func (b *HashHeight) Equal(hash types.Hash, height uint64) bool {
	return b.Hash == hash && b.Height == height
}

func (b *HashHeight) Proto() *vitepb.BlockID {
	return &vitepb.BlockID{
		Hash:   b.Hash[:],
		Height: b.Height,
	}
}

func (b *HashHeight) DeProto(pb *vitepb.BlockID) {
	copy(b.Hash[:], pb.Hash)
	b.Height = pb.Height
}

func (b *HashHeight) Serialize() ([]byte, error) {
	return proto.Marshal(b.Proto())
}

func (b *HashHeight) Deserialize(data []byte) error {
	pb := new(vitepb.BlockID)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}
	b.DeProto(pb)
	return nil
}
