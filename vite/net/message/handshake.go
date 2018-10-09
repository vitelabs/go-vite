package message

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/vitepb"
)

type HandShake struct {
	CmdSet  uint64
	Height  uint64
	Port    uint16
	Current types.Hash
	Genesis types.Hash
}

func (h *HandShake) Serialize() ([]byte, error) {
	pb := new(vitepb.Handshake)
	pb.CmdSet = h.CmdSet
	pb.Height = h.Height
	pb.Port = uint32(h.Port)
	pb.Current = h.Current[:]
	pb.Genesis = h.Genesis[:]

	return proto.Marshal(pb)
}

func (h *HandShake) Deserialize(data []byte) error {
	pb := new(vitepb.Handshake)

	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	h.CmdSet = pb.CmdSet
	h.Height = pb.Height
	h.Port = uint16(pb.Port)
	copy(h.Current[:], pb.Current)
	copy(h.Genesis[:], pb.Genesis)

	return nil
}
