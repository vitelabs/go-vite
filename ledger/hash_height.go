package ledger

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb"
)

type HashHeight struct {
	Height uint64     `json:"height"`
	Hash   types.Hash `json:"hash"`
}

func (b *HashHeight) Equal(hash types.Hash, height uint64) bool {
	return b.Hash == hash && b.Height == height
}

func (b *HashHeight) Proto() *vitepb.HashHeight {
	return &vitepb.HashHeight{
		Hash:   b.Hash[:],
		Height: b.Height,
	}
}

func (b *HashHeight) DeProto(pb *vitepb.HashHeight) (err error) {
	b.Hash, err = types.BytesToHash(pb.Hash)
	if err != nil {
		return
	}
	b.Height = pb.Height
	return nil
}

func (b *HashHeight) Serialize() ([]byte, error) {
	return proto.Marshal(b.Proto())
}

func (b *HashHeight) Deserialize(data []byte) error {
	pb := new(vitepb.HashHeight)
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}
	return b.DeProto(pb)
}
