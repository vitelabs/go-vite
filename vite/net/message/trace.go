package message

import (
	"github.com/golang/protobuf/proto"
	"github.com/vitelabs/go-vite/common/types"
)

type Tracer struct {
	Hash types.Hash
	Path []string
	TTL  int
}

func (t *Tracer) String() string {
	return "<trace " + t.Hash.String() + ">"
}

func (t *Tracer) Serialize() ([]byte, error) {
	pb := &Trace{
		Hash: t.Hash.Bytes(),
		Path: t.Path,
		TTL:  uint32(t.TTL),
	}

	return proto.Marshal(pb)
}

func (t *Tracer) Deserialize(data []byte) error {
	pb := &Trace{}
	err := proto.Unmarshal(data, pb)
	if err != nil {
		return err
	}

	t.Hash, err = types.BytesToHash(pb.Hash)
	if err != nil {
		return err
	}

	t.Path = pb.Path
	t.TTL = int(pb.TTL)
	return nil
}
